package proxy

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/patrickmn/go-cache"
	kubecache "k8s.io/client-go/tools/cache"
)

const keyLength = 512 / 8
const accessKeyRetentionTime = 10 * time.Minute

// Atlas URLs for authenticating
var proxyURLs = struct {
	Base    string
	Login   string
	Refresh string
	Token   string
}{}
var useHTTPSForSelfRedirect = true
var jwtKey []byte
var sharedKey []byte

var saturnTokenExpirationSeconds = 3600   //  1 hour to expiration
var refreshTokenExpirationSeconds = 86400 //  24 hours to expiration
var debug = true
var defaultPort = "8888"
var tokenMapMutex sync.Mutex
var tokenMap = make(map[string]string)
var urlCommonSuffix = ".localhost"
var authTokenMutex sync.Mutex
var authTokenCache = cache.New(accessKeyRetentionTime, accessKeyRetentionTime)

var namespace = "main-namespace"
var userSessionConfigMapName = "saturn-proxy-sessions"
var proxyConfigMapName = "saturn-auth-proxy"

var proxyConfig *ProxyConfig
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

/*
	Get env var or default
*/
func getEnv(key, dflt string) string {
	value, ok := os.LookupEnv(key)
	if ok {
		return value
	}
	return dflt
}

/*
	Basic error page
*/
func respondWithError(res http.ResponseWriter, status int, message string) {
	log.Printf("Error %d: %s", status, message)
	res.Header().Add("Content-Type", "text/html;charset=utf-8")
	res.WriteHeader(status)
	fmt.Fprintf(res, "<html><head><title>%[1]d</title></head><body>Error %[1]d: %[2]s</body></html>", status, message)
}

func extractTargetURLKey(hostname string) string {

	tmp := strings.SplitN(hostname, ":", 2)[0]
	if strings.HasSuffix(tmp, urlCommonSuffix) {
		tmp = tmp[:len(tmp)-len(urlCommonSuffix)]
	}

	return tmp
}

/*
	Generate a key for signing JWTs
*/
func generateCookieSigningKey(n int) ([]byte, error) {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-"
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	for i, b1 := range b {
		b[i] = letters[b1%byte(len(letters))]
	}
	return b, nil
}

/*
	Serve a reverse proxy for a given url
*/
func serveReverseProxy(res http.ResponseWriter, req *http.Request, tmpTargetURL string) {
	// Modify the headers for  SSL redirection and cache control
	req.Header.Set("X-Forwarded-Host", req.Host)
	req.Header.Set("Cache-Control", "no-cache, no-store, no-transform, must-revalidate, max-age=0")

	proxiedURL, _ := url.Parse(tmpTargetURL)
	proxy := httputil.NewSingleHostReverseProxy(proxiedURL)
	proxy.ServeHTTP(res, req) // non blocking
}

// Extend standard claims for saturn specific requirements
type SaturnClaims struct {
	Resource      string `json:"resource"`
	RedirectToken string `json:"redirect_token,omitempty"`
	jwt.StandardClaims
}

/*
	Create JWT for proxy authentication or auth refresh
*/
func createToken(host, user_id string, expiration time.Time, refreshToken bool) (string, error) {
	var audience string
	var key []byte
	if refreshToken {
		// Audience indicates that this token is a refresh token
		// and cannot be used for authenticating in the proxy
		audience = jwtPrincipals.Atlas
		key = sharedKey
	} else {
		audience = jwtPrincipals.SaturnAuthProxy
		key = jwtKey
	}
	claims := &SaturnClaims{
		Resource: host,
		StandardClaims: jwt.StandardClaims{
			Audience:  audience,
			ExpiresAt: expiration.Unix(),
			Issuer:    jwtPrincipals.SaturnAuthProxy,
			Subject:   user_id,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(key)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

/*
	Create new JWT cookies for session auth and refresh token
*/
func setNewCookies(res http.ResponseWriter, req *http.Request, user_id string) error {
	// Create a new refresh_token
	refreshExpiration := time.Now().Add(time.Duration(refreshTokenExpirationSeconds) * time.Second)
	refreshTokenString, err := createToken(req.Host, user_id, refreshExpiration, true)
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
	expirationTime := time.Now().Add(time.Duration(saturnTokenExpirationSeconds) * time.Second)
	tokenString, err := createToken(req.Host, user_id, expirationTime, false)
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

/*
	Validate JWT tokens based on issuer, audience, and request host
*/
func validateSaturnToken(saturnToken, issuer string, req *http.Request) (*SaturnClaims, error) {
	claims := &SaturnClaims{}

	var key []byte
	switch issuer {
	case jwtPrincipals.Atlas:
		key = sharedKey
	case jwtPrincipals.SaturnAuthProxy:
		key = jwtKey
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
	} else if !sessionConfig.CheckUser(claims.Subject) {
		log.Printf("Failed to auth %s", claims.Subject)
		log.Printf("Sessions: %v", sessionConfig.UserSessions)
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

/*
	Check for a refresh token cookie to reauthenticate, or redirect to Atlas login
*/
func authenticate(res http.ResponseWriter, req *http.Request) bool {
	res.Header().Add("Cache-Control", "no-cache")

	protocol := "https://"
	if !useHTTPSForSelfRedirect {
		protocol = "http://"
	}

	origURL := protocol + req.Host + req.URL.RequestURI()
	if debug {
		log.Printf("Authenticating URL: %s", origURL)
	}

	uniqToken, _ := generateCookieSigningKey(40)

	// Check for refresh token
	if refreshCookie, err := req.Cookie(proxyCookies.RefreshToken); err == nil {
		// Proxy auth request to Atlas
		if debug {
			log.Println("Found refresh_token, proxying authentication request.")
		}
		buffer := new(bytes.Buffer)
		json.NewEncoder(buffer).Encode(&proxyAuthRequest{
			RefreshToken: refreshCookie.Value,
		})
		authResp, err := http.Post(proxyURLs.Refresh, "application/json", buffer)
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
				setNewCookies(res, req, claims.Subject)
				return true
			}
		}
	}

	// If proxy auth req failed or refresh_token not found, fallback to redirect
	redirectParams := url.Values{}
	redirectParams.Add("next", origURL)
	redirectParams.Add("redirect_token", string(uniqToken))
	redirectUrl := proxyURLs.Login + "?" + redirectParams.Encode()

	tokenMapMutex.Lock()
	tokenMap[string(uniqToken)] = ""
	log.Printf("Added token %s", uniqToken)
	tokenMapMutex.Unlock()

	if debug {
		log.Printf("Redirecting to fallback url: %s", redirectUrl)
	}
	http.Redirect(res, req, redirectUrl, 302)
	return false
}

/*
	Given a request, check for valid authentication in headers, URL params, and cookies.
	If no valid authentication is found, attempt to reauthenticate before sending request
	to its destination.
*/
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

		err = setNewCookies(res, req, claims.Subject)
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
		service := proxyConfig.GetTarget(targetKey)
		if service != "" {
			tmpTargetURL = service
			if debug {
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

/*
	Check authorization header validity for the requested resource. Returns true if no error has been thrown.
	If false is returned, the caller should assume that a response has already been written out.

	This is used only for customers to access their deployments via Authorization header with a fixed token,
	(e.g. for creating automation against a deployment) so that they don't have to mess with cookies and token
	expiration.
*/
func checkTokenAuth(res http.ResponseWriter, req *http.Request, authHeader string) bool {
	target := extractTargetURLKey(req.Host)
	cacheKey := fmt.Sprintf("%s/%s", target, authHeader)
	// check if the header is already in our cache for this host
	authTokenMutex.Lock()
	_, found := authTokenCache.Get(cacheKey)
	authTokenMutex.Unlock()
	if found {
		// valid key, valid target - good to go
		if debug {
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
		authTokenCache.Set(cacheKey, true, accessKeyRetentionTime)
		authTokenMutex.Unlock()
		return true
	}
	res.WriteHeader(403)
	fmt.Fprint(res, "This token is not valid for this resource.")
	return false
}

func checkTokenValidity(authHeader, target string) (bool, error) {
	client := &http.Client{}
	request, err := http.NewRequest("GET", proxyURLs.Token+"?targetResource="+target, nil)
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

func Run() {

	debug, _ = strconv.ParseBool(getEnv("PROXY_DEBUG", "true"))

	urlCommonSuffix = getEnv("PROXY_SUFFIX", urlCommonSuffix)

	defaultPort = getEnv("PROXY_LISTEN_PORT", defaultPort)

	// Parse URLs
	baseURL, err := url.Parse(getEnv("PROXY_BASE_URL", "http://dev.localtest.me:8888"))
	if err != nil {
		log.Panic(err)
	}
	loginURL, err := baseURL.Parse(getEnv("PROXY_LOGIN_PATH", "/auth/login"))
	if err != nil {
		log.Panic(err)
	}
	refreshURL, err := baseURL.Parse(getEnv("PROXY_REFRESH_PATH", "/auth/refresh"))
	if err != nil {
		log.Panic(err)
	}
	tokenURL, err := baseURL.Parse(getEnv("PROXY_TOKEN_PATH", "/api/deployments/auth"))
	if err != nil {
		log.Panic(err)
	}
	proxyURLs.Base = baseURL.String()
	proxyURLs.Login = loginURL.String()
	proxyURLs.Refresh = refreshURL.String()
	proxyURLs.Token = tokenURL.String()

	useHTTPSForSelfRedirect = getEnv("HTTPS_SELF_REDIRECT", "true") == "true"

	// Parse and generate keys
	sharedKey = []byte(getEnv("PROXY_SHARED_KEY", ""))
	if len(sharedKey) == 0 {
		if debug {
			sharedKey = []byte("debugKeyForTestOnlydNeverUseInProduction123456789012345678901234567890")
			log.Printf("WARNING! WARNING! Running in debug mode with predefined weak key!\n - set PROXY_DEBUG=false if you are running this in production. ")
		} else {
			log.Printf("Critical error: unable to obtain shared saturn signing key.\n - set environment variable PROXY_SHARED_KEY")
			return
		}
	}

	if len(sharedKey) < keyLength {
		log.Printf("Critical error: shared saturn signing key is too short (%d),\n - set environment variable PROXY_SHARED_KEY", len(sharedKey))
		return
	}

	log.Printf("Saturn signing key obtained and has length %d bytes", len(sharedKey))

	tmpKey, err := generateCookieSigningKey(keyLength)
	if err != nil {
		log.Printf("Critical error: unable to generate JWT signing key")
		return
	}

	log.Printf("JWT signing key generated and has length %d bytes", len(tmpKey))

	// Token/Cookie expirations
	jwtKey = tmpKey
	saturnTokenExpirationSeconds, _ = strconv.Atoi(getEnv("SATURN_TOKEN_EXPIRE_SEC", "3600"))
	refreshTokenExpirationSeconds, _ = strconv.Atoi(getEnv("REFRESH_TOKEN_EXPIRE_SEC", "86400"))
	log.Printf("JWT cookie expiration time: %ds", saturnTokenExpirationSeconds)
	log.Printf("Refresh cookie expiration time: %ds", refreshTokenExpirationSeconds)

	listAddr := fmt.Sprintf(":%s", defaultPort)

	namespace = getEnv("NAMESPACE", namespace)

	log.Printf("Listening on %s", listAddr)
	log.Printf("Redirect URL:    %s", proxyURLs.Login)
	log.Printf("Refresh URL:     %s", proxyURLs.Refresh)

	proxyConfigMapName = getEnv("PROXY_CONFIGMAP", proxyConfigMapName)
	userSessionConfigMapName = getEnv("SESSIONS_CONFIGMAP", userSessionConfigMapName)

	// Watch for changes to proxy target configmap
	proxyConfig = &ProxyConfig{TargetMap: make(map[string]string)}
	targetWatcher := NewConfigWatcher(proxyConfigMapName, namespace)
	go targetWatcher.Watch(kubecache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			// Update proxy target config
			proxyConfig.Load(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			// Update proxy target config
			proxyConfig.Load(newObj)
		},
		DeleteFunc: func(obj interface{}) {
			// Clear proxy target config
			proxyConfig.mutex.Lock()
			defer proxyConfig.mutex.Unlock()
			proxyConfig.TargetMap = make(map[string]string)
			log.Println("Deleted proxy target configuration")
		},
	})

	// Watch for changes to user proxy sessions configmap
	sessionConfig = &SessionConfig{UserSessions: make(map[string]struct{})}
	sessionWatcher := NewConfigWatcher(userSessionConfigMapName, namespace)
	go sessionWatcher.Watch(kubecache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			// Update user sessions
			sessionConfig.Load(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			// Update user sessions
			sessionConfig.Load(newObj)
		},
		DeleteFunc: func(obj interface{}) {
			// Clear user sessions
			sessionConfig.mutex.Lock()
			defer sessionConfig.mutex.Unlock()
			sessionConfig.UserSessions = make(map[string]struct{})
			log.Println("Deleted user sessions")
		},
	})

	err = http.ListenAndServe(listAddr, &proxyServer{})
	if err != nil {
		log.Panic(err)
	}
}
