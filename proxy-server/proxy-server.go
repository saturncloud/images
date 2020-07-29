package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
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
)

const keyLength = 512 / 8
const accessKeyRetentionTime = 10 * time.Minute

var fallbackURL = "http://localhost/fallback"            // not authorized fallback
var tokenVerificationURL = "http://localhost/key-verify" //
var useHTTPSForSelfRedirect = true
var jwtKey []byte
var sharedKey []byte

var jwtExpirationSeconds = 3600 //  1 hour to expiration
var debug = true
var defaultPort = "8080"
var tokenMapMutex sync.Mutex
var tokenMap = make(map[string]string)
var configPathName = "/etc/config/proxy.json"
var targetMap = make(map[string]string)
var configMutex sync.Mutex
var urlCommonSuffix = ".localhost"
var authTokenMutex sync.Mutex
var authTokenCache = cache.New(accessKeyRetentionTime, accessKeyRetentionTime)

var (
	errExpired        = errors.New("Token expired")
	errTokenInvalid   = errors.New("Unauthorized atteimpt to access resource: invalid token")
	errNoIssuers      = errors.New("Invalid token from saturn (no issuers)")
	errInvalidIssuers = errors.New("Invalid token from saturn (invalid issuer)")
)

// Get env var or default
func getEnv(key, dflt string) string {
	value, ok := os.LookupEnv(key)
	if ok {
		return value
	}
	return dflt
}

var configTimeStamp time.Time

func respondWithError(res http.ResponseWriter, status int, message string) {
	log.Printf("Error %d: %s", status, message)
	res.Header().Add("Content-Type", "text/html;charset=utf-8")
	res.WriteHeader(status)
	fmt.Fprintf(res, "<html><head><title>%[1]d</title></head><body>Error %[1]d: %[2]s</body></html>", status, message)
}

func maybeReadProxyConfig() {

	fi, err := os.Stat(configPathName)
	if err != nil {
		return
	}
	cfgTime := fi.ModTime()
	if configTimeStamp == cfgTime {
		return // file not changed
	}

	file, _ := ioutil.ReadFile(configPathName)
	data := make(map[string]string)
	err = json.Unmarshal([]byte(file), &data)

	if err != nil {
		return
	}

	log.Printf("Reading proxy config file %s", configPathName)
	for k, v := range data {
		log.Printf("Destination '%s' --> url '%s' \n", k, v)
	}

	configMutex.Lock()
	defer configMutex.Unlock()
	targetMap = make(map[string]string)
	for key, value := range data {
		targetMap[key] = value
	}
	configTimeStamp = cfgTime
}

func configReadingLoop() {
	// it's ok to loop frequently - the function checks if the file has been modified
	// before loading it
	for range time.Tick(1 * time.Second) {
		maybeReadProxyConfig()
	}
}

func extractTargetURLKey(hostname string) string {

	tmp := strings.SplitN(hostname, ":", 2)[0]
	if strings.HasSuffix(tmp, urlCommonSuffix) {
		tmp = tmp[:len(tmp)-len(urlCommonSuffix)]
	}

	return tmp
}

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

// Serve a reverse proxy for a given url
func serveReverseProxy(res http.ResponseWriter, req *http.Request, tmpTargetURL string) {
	// Modify the headers for  SSL redirection and cache control
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
	req.Header.Set("Cache-Control", "no-cache, no-store, no-transform, must-revalidate, max-age=0")

	proxiedURL, _ := url.Parse(tmpTargetURL)
	proxy := httputil.NewSingleHostReverseProxy(proxiedURL)
	proxy.ServeHTTP(res, req) // non blocking
}

func setNewCookie(res http.ResponseWriter) error {
	expirationTime := time.Now().Add(time.Duration(jwtExpirationSeconds) * time.Second)
	claims := &jwt.StandardClaims{
		ExpiresAt: expirationTime.Unix(),
		Issuer:    "saturn-auth-proxy",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return err
	}

	// Set the new token as the users `saturn_token` cookie
	http.SetCookie(res, &http.Cookie{
		Name:    "saturn_token",
		Value:   tokenString,
		Expires: expirationTime,
		Path:    "/",
	})

	return nil
}

func validateSaturnToken(saturnToken string, key []byte) (*jwt.StandardClaims, error) {
	claims := &jwt.StandardClaims{}

	// Parse the JWT string and store the result in `claims`.
	tkn, err := jwt.ParseWithClaims(saturnToken, claims, func(token *jwt.Token) (interface{}, error) {
		return key, nil
	})
	if err != nil {
		return nil, err
	}
	if !tkn.Valid {
		return nil, errTokenInvalid
	}
	if len(claims.Issuer) == 0 {
		return nil, errNoIssuers
	}
	if time.Unix(claims.ExpiresAt, 0).Sub(time.Now()) < 0 {
		return nil, errExpired
	}
	return claims, nil
}

type proxyAuthResponse struct {
	SaturnToken string `json:"saturn_token"`
}

func authenticate(res http.ResponseWriter, req *http.Request) bool {
	res.Header().Add("Cache-Control", "no-cache")

	protocol := "https://"
	if !useHTTPSForSelfRedirect {
		protocol = "http://"
	}

	origURL := protocol + req.Host + req.URL.RequestURI()
	if debug {
		log.Printf("fallback: %s", origURL)
	}

	uniqToken, _ := generateCookieSigningKey(40)

	proxyParams := url.Values{}
	proxyParams.Add("ret_token", string(uniqToken))
	proxyParams.Add("proxy_auth", "true")
	proxyAuthUrl := fallbackURL + "?" + proxyParams.Encode()

	// Check for session cookie
	if pdcSessionCookie, err := req.Cookie("PDC_SESSION"); err == nil {
		// Proxy auth request to Atlas
		if debug {
			log.Printf("Proxying authentication request for %s", origURL)
		}
		authReq, _ := http.NewRequest("GET", proxyAuthUrl, nil)
		authReq.Header.Set(
			"Cookie", fmt.Sprintf("%s=%s;", pdcSessionCookie.Name, pdcSessionCookie.Value),
		)
		client := &http.Client{}
		authResp, err := client.Do(authReq)
		if err != nil {
			log.Printf("Error: %s\n", err)
			return false
		}
		defer authResp.Body.Close()
		var result proxyAuthResponse
		json.NewDecoder(authResp.Body).Decode(&result)

		// Check for valid token in response
		if result.SaturnToken != "" {
			if _, err = validateSaturnToken(result.SaturnToken, sharedKey); err != nil {
				log.Printf("Invalid token returned: %s", err)
			} else {
				setNewCookie(res)
				return true
			}
		}
	}

	// If proxy auth req failed or PDC_COOKIE not found, fallback to redirect
	redirectParams := url.Values{}
	redirectParams.Add("next", origURL)
	redirectParams.Add("ret_token", string(uniqToken))
	redirectUrl := fallbackURL + "?" + redirectParams.Encode()

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

// Given a request send it to the appropriate url
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

		claims, err := validateSaturnToken(satToken, sharedKey)
		if err != nil {
			fmt.Printf("Authorization Erorr: %s", err)
			respondWithError(res, http.StatusUnauthorized, "Invalid token.")
		}

		tokenMapMutex.Lock()
		_, ok := tokenMap[claims.Issuer]
		if ok {
			delete(tokenMap, claims.Issuer)
			log.Printf("Deleted token %s", claims.Issuer)

		}
		tokenMapMutex.Unlock()

		if !ok {
			log.Printf("Authorization Error: %s", errInvalidIssuers)
			respondWithError(res, http.StatusUnauthorized, "Invalid token.")
			return
		}

		err = setNewCookie(res)
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
			_, err = validateSaturnToken(tokenCookie.Value, jwtKey)
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
		configMutex.Lock()
		targetKey := extractTargetURLKey(req.Host)
		tmp, ok := targetMap[targetKey]
		configMutex.Unlock()
		if ok {
			tmpTargetURL = tmp
			if debug {
				log.Printf("Debug: target url is %s", tmp)
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

// Check token validity for the requested resource. Returns true if no error has been thrown.
// If false is returned, the caller should assume that a response has already been written out.
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
		panic(err)
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
	request, err := http.NewRequest("GET", tokenVerificationURL+"?targetResource="+target, nil)
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

func main() {

	debug, _ = strconv.ParseBool(getEnv("PROXY_DEBUG", "true"))

	configPathName = getEnv("PROXY_CONFIG", configPathName)
	urlCommonSuffix = getEnv("PROXY_SUFFIX", urlCommonSuffix)

	fallbackURL = getEnv("PROXY_FALLBACK_URL",
		"http://localhost:"+getEnv("PROXY_LISTEN_PORT", defaultPort)+"/fallback")
	tokenVerificationURL = getEnv("PROXY_AUTH_VERIFICATION_URL", tokenVerificationURL)
	useHTTPSForSelfRedirect = getEnv("HTTPS_SELF_REDIRECT", "true") == "true"

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

	jwtKey = tmpKey
	jwtExpirationSeconds, _ = strconv.Atoi(getEnv("JWT_EXPIRE_SEC", "3600"))
	log.Printf("JWT cookie expiration time: %ds", jwtExpirationSeconds)

	listAddr := ":" + getEnv("PROXY_LISTEN_PORT", defaultPort)

	log.Printf("Listening on %s", listAddr)

	log.Printf("Authentication URL:       %s", fallbackURL)

	maybeReadProxyConfig()
	go configReadingLoop()

	err = http.ListenAndServe(listAddr, &proxyServer{})
	if err != nil {
		panic(err)
	}
}
