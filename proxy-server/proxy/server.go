package proxy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"saturncloud/proxy-server/util"
	"strings"
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/patrickmn/go-cache"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var settings *util.Settings

var tokenMapMutex sync.Mutex
var tokenMap = make(map[string]string)
var authTokenMutex sync.Mutex
var authTokenCache *cache.Cache

var httpConfig *HTTPConfig
var sessionConfig *SessionConfig

// Used for identifying principal actors in JWTs
var jwtPrincipals = struct {
	Atlas           string
	SaturnAuthProxy string
}{
	Atlas:           "atlas",
	SaturnAuthProxy: "saturn-auth-proxy",
}

// Cookies used for proxy auth
var proxyCookies = struct {
	RefreshToken string
	SaturnToken  string
}{
	RefreshToken: "refresh_token",
	SaturnToken:  "saturn_token",
}

var (
	errExpired              = errors.New("Token expired")
	errTokenInvalid         = errors.New("Unauthorized attempt to access resource: invalid token")
	errNoIssuers            = errors.New("Invalid token, missing issuer")
	errInvalidIssuers       = errors.New("Invalid token issuer")
	errInvalidAudience      = errors.New("Invalid token audience")
	errInvalidResource      = errors.New("Token is not valid for the requested host")
	errInvalidRedirectToken = errors.New("Invalid redirect token")
	errInvalidUserSession   = errors.New("Invalid session, user is not logged in")
)

// respondWithError renders a basic error page
func respondWithError(res http.ResponseWriter, status int, message string) {
	log.Printf("Error %d: %s", status, message)
	res.Header().Add("Content-Type", "text/html;charset=utf-8")
	res.WriteHeader(status)
	fmt.Fprintf(res, "<html><head><title>%[1]d</title></head><body>Error %[1]d: %[2]s</body></html>", status, message)
}

// extractTargetURLKey returns hostname with the CommonSuffix removed
func extractTargetURLKey(hostname string) string {
	tmp := strings.SplitN(hostname, ":", 2)[0]
	if strings.HasSuffix(tmp, settings.ProxyURLs.CommonSuffix) {
		tmp = tmp[:len(tmp)-len(settings.ProxyURLs.CommonSuffix)]
	}
	return tmp
}

// serveReverseProxy proxies a request to the given target
func serveReverseProxy(res http.ResponseWriter, req *http.Request, tmpTargetURL string) {
	// Modify the headers for  SSL redirection and cache control
	req.Header.Set("X-Forwarded-Host", req.Host)
	req.Header.Set("Cache-Control", "no-cache, no-store, no-transform, must-revalidate, max-age=0")

	proxiedURL, _ := url.Parse(tmpTargetURL)
	proxy := httputil.NewSingleHostReverseProxy(proxiedURL)
	proxy.ServeHTTP(res, req) // non blocking
}

// SaturnClaims extend standard JWT claims for saturn specific requirements
type SaturnClaims struct {
	Resource      string `json:"resource"`
	RedirectToken string `json:"redirect_token,omitempty"`
	jwt.StandardClaims
}

// Expiration returns the claim's expiration as a datetime
func (sc *SaturnClaims) Expiration() time.Time {
	return time.Unix(sc.ExpiresAt, 0)
}

// createToken Creates a JWT for proxy authentication or auth refresh
func createToken(host, sessionID string, expiration time.Time, refreshToken bool) (string, error) {
	var audience string
	var key []byte
	if refreshToken {
		// Audience indicates that this token is a refresh token
		// and cannot be used for authenticating in the proxy
		audience = jwtPrincipals.Atlas
		key = settings.SharedKey
	} else {
		audience = jwtPrincipals.SaturnAuthProxy
		key = settings.JWTKey
	}
	claims := &SaturnClaims{
		Resource: host,
		StandardClaims: jwt.StandardClaims{
			Audience:  audience,
			ExpiresAt: expiration.Unix(),
			Issuer:    jwtPrincipals.SaturnAuthProxy,
			Subject:   sessionID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(key)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// setNewCookies creates new JWT cookies for session auth and refresh token
func setNewCookies(res http.ResponseWriter, req *http.Request, claims *SaturnClaims) error {
	// Create a new refresh_token
	refreshExpiration := time.Now().Add(settings.RefreshTokenExpiration)
	refreshTokenString, err := createToken(req.Host, claims.Subject, refreshExpiration, true)
	if err != nil {
		return err
	}
	http.SetCookie(res, &http.Cookie{
		Name:    proxyCookies.RefreshToken,
		Value:   refreshTokenString,
		Expires: refreshExpiration,
		Path:    "/",
	})

	// Create a new saturn_token
	expirationTime := time.Now().Add(settings.SaturnTokenExpiration)
	claimExpiration := claims.Expiration()
	if claimExpiration.Before(expirationTime) {
		// If claim expiration is before default, use shorter expiration
		expirationTime = claimExpiration
	}
	tokenString, err := createToken(req.Host, claims.Subject, expirationTime, false)
	if err != nil {
		return err
	}
	http.SetCookie(res, &http.Cookie{
		Name:    proxyCookies.SaturnToken,
		Value:   tokenString,
		Expires: expirationTime,
		Path:    "/",
	})

	return nil
}

// validateSaturnToken validates JWT tokens based on issuer, audience, and request host
func validateSaturnToken(saturnToken, issuer string, req *http.Request) (*SaturnClaims, error) {
	claims := &SaturnClaims{}

	var key []byte
	switch issuer {
	case jwtPrincipals.Atlas:
		key = settings.SharedKey
	case jwtPrincipals.SaturnAuthProxy:
		key = settings.JWTKey
	}

	// Parse the JWT string and store the result in `claims`.
	tkn, err := jwt.ParseWithClaims(saturnToken, claims, func(token *jwt.Token) (interface{}, error) {
		return key, nil
	})
	if err != nil {
		return nil, err
	} else if !tkn.Valid {
		return nil, errTokenInvalid
	} else if len(claims.Issuer) == 0 {
		return nil, errNoIssuers
	} else if claims.Issuer != issuer {
		return nil, errInvalidIssuers
	} else if claims.Audience != jwtPrincipals.SaturnAuthProxy {
		return nil, errInvalidAudience
	} else if claims.Resource != req.Host {
		log.Printf("\n\n%s\n\n", claims.Resource)
		return nil, errInvalidResource
	} else if time.Unix(claims.ExpiresAt, 0).Sub(time.Now()) < 0 {
		return nil, errExpired
	} else if !sessionConfig.CheckSession(claims.Subject) {
		return nil, errInvalidUserSession
	}
	return claims, nil
}

type proxyAuthRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type proxyAuthResponse struct {
	SaturnToken string `json:"saturn_token"`
}

// authenticate checks for a refresh token cookie to reauthenticate, or redirect to Atlas login
func authenticate(res http.ResponseWriter, req *http.Request) bool {
	res.Header().Add("Cache-Control", "no-cache")

	protocol := "https://"
	if !settings.HTTPSRedirect {
		protocol = "http://"
	}

	origURL := protocol + req.Host + req.URL.RequestURI()
	if settings.Debug {
		log.Printf("Authenticating URL: %s", origURL)
	}

	uniqToken, _ := settings.GenerateCookieSigningKey()

	// Check for refresh token
	if refreshCookie, err := req.Cookie(proxyCookies.RefreshToken); err == nil {
		// Proxy auth request to Atlas
		if settings.Debug {
			log.Println("Found refresh_token, proxying authentication request.")
		}
		buffer := new(bytes.Buffer)
		json.NewEncoder(buffer).Encode(&proxyAuthRequest{
			RefreshToken: refreshCookie.Value,
		})
		authResp, err := http.Post(settings.ProxyURLs.Refresh.String(), "application/json", buffer)
		if err != nil {
			log.Printf("Authentication failed: %s", err)
			return false
		}
		defer authResp.Body.Close()
		var result proxyAuthResponse
		json.NewDecoder(authResp.Body).Decode(&result)

		// Check for valid token in response
		if result.SaturnToken != "" {
			if claims, err := validateSaturnToken(result.SaturnToken, jwtPrincipals.Atlas, req); err != nil {
				log.Printf("Invalid token returned: %s", err)
			} else if claims.Issuer != jwtPrincipals.Atlas {
				log.Printf("Authentication failed: %s, expected %s", errInvalidIssuers, jwtPrincipals.Atlas)
			} else {
				setNewCookies(res, req, claims)
				return true
			}
		}
	}

	// If proxy auth req failed or refresh_token not found, fallback to redirect
	redirectParams := url.Values{}
	redirectParams.Add("next", origURL)
	redirectParams.Add("redirect_token", string(uniqToken))
	redirectURL := settings.ProxyURLs.Login.String() + "?" + redirectParams.Encode()

	tokenMapMutex.Lock()
	tokenMap[string(uniqToken)] = ""
	log.Printf("Added token %s", uniqToken)
	tokenMapMutex.Unlock()

	if settings.Debug {
		log.Printf("Redirecting to fallback url: %s", redirectURL)
	}
	http.Redirect(res, req, redirectURL, 302)
	return false
}

// handleRequestAndRedirect checks for valid auth, or attempts reauthentication before proxying
func handleRequestAndRedirect(res http.ResponseWriter, req *http.Request) {
	tmpTargetURL := ""

	authHeader := req.Header.Get("Authorization")
	satToken := req.URL.Query().Get("saturn_token")
	if authHeader != "" {
		if !checkTokenAuth(res, req, authHeader) {
			return
		}
	} else if satToken != "" {
		// if we receive authorization token in the request : the user authorized by the backend

		claims, err := validateSaturnToken(satToken, jwtPrincipals.Atlas, req)
		if err != nil {
			fmt.Printf("Authorization Error: %s", err)
			respondWithError(res, http.StatusUnauthorized, "Invalid token.")
			return
		}

		tokenMapMutex.Lock()
		_, ok := tokenMap[claims.RedirectToken]
		if ok {
			delete(tokenMap, claims.RedirectToken)
			log.Printf("Deleted token %s", claims.RedirectToken)

		}
		tokenMapMutex.Unlock()

		if !ok {
			log.Printf("Authorization Error: %s", errInvalidRedirectToken)
			respondWithError(res, http.StatusUnauthorized, "Invalid token.")
			return
		}

		err = setNewCookies(res, req, claims)
		if err != nil {
			log.Printf("Error setting cookie: %+v", err)
			respondWithError(res, http.StatusInternalServerError, "An internal error has occurred.")
			return
		}
		log.Printf("OK: Setting cookie")
		// redirect back to self to remove saturn_token from the URL
		q := req.URL.Query()
		q.Del("saturn_token")
		req.URL.RawQuery = q.Encode()
		res.Header().Add("Cache-Control", "no-cache")
		http.Redirect(res, req, req.URL.String(), 302)
		log.Printf("OK: Redirecting to self (%s)", req.URL.String())
		return
	}

	if authHeader == "" {
		// Check for valid cookie token
		var tokenCookie *http.Cookie
		var err error
		if tokenCookie, err = req.Cookie("saturn_token"); err == nil {
			// Validate cookie
			_, err = validateSaturnToken(tokenCookie.Value, jwtPrincipals.SaturnAuthProxy, req)
		}
		if err != nil {
			// Cookie is invalid or missing
			log.Printf("Invalid cookie: %s", err)
			if !authenticate(res, req) {
				return
			}
		}
	}

	if len(tmpTargetURL) == 0 {
		targetKey := extractTargetURLKey(req.Host)
		service := httpConfig.GetTarget(targetKey)
		if service != "" {
			tmpTargetURL = service
			if settings.Debug {
				log.Printf("Debug: target url is %s", service)
			}
		} else {
			log.Printf("Unknown target url for %s", req.Host)
			respondWithError(res, http.StatusBadRequest, "Unable to route request to a valid resource.")
			return
		}
	}

	// here we finally OK
	serveReverseProxy(res, req, tmpTargetURL)
	log.Printf("OK: Proxying to url: %s %s\n", tmpTargetURL, req.URL.String())
}

// checkTokenAuth checks authorization header validity for the requested resource. Returns false if response already written out.
func checkTokenAuth(res http.ResponseWriter, req *http.Request, authHeader string) bool {
	// This is used only for customers to access their deployments via Authorization header with a fixed token,
	// (e.g. for creating automation against a deployment) so that they don't have to mess with cookies and token
	// expiration.

	target := extractTargetURLKey(req.Host)
	cacheKey := fmt.Sprintf("%s/%s", target, authHeader)
	// check if the header is already in our cache for this host
	authTokenMutex.Lock()
	_, found := authTokenCache.Get(cacheKey)
	authTokenMutex.Unlock()
	if found {
		// valid key, valid target - good to go
		if settings.Debug {
			log.Printf("Valid cache hit for %s", target)
		}
		return true
	}

	valid, err := checkTokenValidity(authHeader, target)
	if err != nil {
		log.Panic(err)
	}
	if valid {
		// add to cache
		log.Printf("Caching token for %s", target)
		authTokenMutex.Lock()
		authTokenCache.Set(cacheKey, true, settings.AccessKeyExpiration)
		authTokenMutex.Unlock()
		return true
	}
	res.WriteHeader(403)
	fmt.Fprint(res, "This token is not valid for this resource.")
	return false
}

func checkTokenValidity(authHeader, target string) (bool, error) {
	client := &http.Client{}
	request, err := http.NewRequest("GET", settings.ProxyURLs.Token.String()+"?targetResource="+target, nil)
	if err != nil {
		return false, err
	}
	request.Header.Add("Authorization", authHeader)
	resp, err := client.Do(request)
	if err != nil {
		return false, err
	}
	if resp.StatusCode == 204 {
		return true, nil
	}
	log.Printf("Rejecting token - received response code %d", resp.StatusCode)
	return false, nil
}

type proxyServer struct{}

func (p *proxyServer) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	handleRequestAndRedirect(res, req)
}

// Run loads settings, start configmap and secrets watchers, and starts the proxy server
func Run(settingsFile string) {
	var err error
	if settings, err = util.LoadSettings(settingsFile); err != nil {
		log.Panicf("Error loading settings: %s", err.Error())
	}

	authTokenCache = cache.New(settings.AccessKeyExpiration, settings.AccessKeyExpiration)

	// Load kubeconfig
	kubeconfigPath := "~/.kube/config"
	var kubeconfig *rest.Config
	if _, err := os.Stat(kubeconfigPath); err == nil {
		kubeconfig, err = clientcmd.BuildConfigFromFlags("", "")
		if err != nil {
			log.Panic(err)
		}
	} else {
		kubeconfig, err = rest.InClusterConfig()
		if err != nil {
			log.Panic(err)
		}
	}

	// Create kube client
	client, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		log.Panic(err)
	}

	// Watch for changes to proxy target configmap
	httpConfig = &HTTPConfig{TargetMap: make(map[string]string)}
	httpConfig.Watch(settings.ProxyConfigMaps.HTTPTargets, settings.Namespace, client)

	// Watch for changes to user proxy sessions configmap
	sessionConfig = &SessionConfig{UserSessions: make(map[string]struct{})}
	sessionConfig.Watch(settings.ProxyConfigMaps.UserSessions, settings.Namespace, client)

	// Watch for changes to TCP proxy configmap if HAProxy enabled
	if settings.HAProxy.Enabled {
		haproxyConfig := NewHAProxyConfig(
			settings.Namespace,
			settings.ClusterDomain,
			settings.HAProxy.BaseDir,
			settings.HAProxy.PIDFile,
			settings.HAProxy.DefaultListeners,
		)
		haproxyConfig.Watch(
			settings.ProxyConfigMaps.TCPTargets,
			settings.Namespace,
			settings.HAProxy.TLSLabelSelector,
			settings.HAProxy.ReloadRateLimit,
			client,
		)
	}

	err = http.ListenAndServe(settings.ListenAddr, &proxyServer{})
	if err != nil {
		log.Panic(err)
	}
}
