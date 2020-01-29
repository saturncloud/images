package main

import (
	"crypto/rand"
	"encoding/json"
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
)

var fallbackURL = "http://localhost/fallback" // not authorized fallback
var useHTTPSForSelfRedirect = true
var jwtKey []byte
var sharedKey []byte

var minutesExpire = 60 //  1 hour to expiration
var debug = true
var defaultPort = "8080"
var mutex sync.Mutex
var tokenMap = make(map[string]string)
var configPathName = "/etc/config/proxy.json"
var targetMap = make(map[string]string)
var configMutex sync.Mutex
var urlCommonSuffix = ".localhost"

const keyLength = 512 / 8

const (
	errGeneric = iota
	errNoCookie
	errExpired
	errTokenInvalid
	errSignatureInvalid
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
	for range time.Tick(30 * time.Second) {
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

	claims := &jwt.StandardClaims{}
	expirationTime := time.Now().Add(time.Duration(minutesExpire) * time.Minute)
	claims.ExpiresAt = expirationTime.Unix()
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
	})

	return nil
}

func validateAuthToken(token *jwt.StandardClaims) bool {
	// token extra validation. its signature is already verified at this point
	// considerations: -
	//			check expiration time for this token
	//          any extra user access limitations
	return true
}

func redirectToFallBack(res http.ResponseWriter, req *http.Request, error int, origURL string) {

	if debug {
		log.Printf("fallback code: %d %s", error, origURL)
	}
	switch error {
	case errNoCookie:
		log.Printf("Fallback: no cookie token presented or bad cookie")
	case errExpired:
		log.Printf("Fallback: cookie expired")
	case errSignatureInvalid:
		log.Printf("Fallback: Unauthorized attempt to access resource: wrong signature\n")
	case errTokenInvalid:
		log.Printf("Fallback: Unauthorized attempt to access resource: invalid token\n")
	case errGeneric:
		log.Printf("Fallback: bad request")
	}

	uniqToken, _ := generateCookieSigningKey(40)

	mutex.Lock()
	tokenMap[string(uniqToken)] = ""
	log.Printf("Added token %s", uniqToken)
	mutex.Unlock()

	qs := url.Values{}

	qs.Add("next", origURL)
	qs.Add("ret_token", string(uniqToken))

	u := fallbackURL + "?" + qs.Encode()

	if debug {
		log.Printf("Redirecting to fallback url: %s", u)
	}

	res.Header().Add("Cache-Control", "no-cache")
	http.Redirect(res, req, u, 302)
}

// Given a request send it to the appropriate url
func handleRequestAndRedirect(res http.ResponseWriter, req *http.Request) {

	// if we receive authorization token in the request : the user authorized by the backend
	satToken := req.URL.Query().Get("saturn_token")
	tmpTargetURL := ""

	if satToken != "" {
		claims := &jwt.StandardClaims{}
		tkn, err := jwt.ParseWithClaims(satToken, claims, func(token *jwt.Token) (interface{}, error) {
			return sharedKey, nil
		})

		if err != nil || !tkn.Valid {
			log.Printf("Authorization Error: invalid token from saturn")
			// Breaking the loop. No more redirects. All is bad for this request.
			respondWithError(res, http.StatusUnauthorized, "Invalid token.")
			return
		}

		// in case authorization body wants to proxy to some different url
		if len(claims.Subject) != 0 {
			tmpTargetURL = claims.Subject
			return
		}

		if len(claims.Issuer) == 0 {
			log.Printf("Authorization Error: invalid token from saturn (no issuers)")
			respondWithError(res, http.StatusUnauthorized, "Invalid token.")
			return
		}

		mutex.Lock()
		_, prs := tokenMap[claims.Issuer]
		if prs {
			delete(tokenMap, claims.Issuer)
			log.Printf("Deleted token %s", claims.Issuer)

		}
		mutex.Unlock()

		if !prs {
			log.Printf("Authorization Error: invalid token from saturn (invalid issuer)")
			respondWithError(res, http.StatusUnauthorized, "Invalid token.")
			return
		}

		if validateAuthToken(claims) {
			err = setNewCookie(res)
			if err != nil {
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
	}

	if len(tmpTargetURL) == 0 {
		configMutex.Lock()
		tmp, ok := targetMap[extractTargetURLKey(req.Host)]
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

	c, err := req.Cookie("saturn_token")
	protocol := "https://"
	if !useHTTPSForSelfRedirect {
		protocol = "http://"
	}
	origURL := protocol + req.Host + req.URL.Path

	if err != nil {
		errcode := errGeneric
		if err == http.ErrNoCookie {
			errcode = errNoCookie
		}
		redirectToFallBack(res, req, errcode, origURL)
		return
	}

	// Get the JWT string from the cookie
	tknStr := c.Value
	claims := &jwt.StandardClaims{}

	// Parse the JWT string and store the result in `claims`.
	tkn, err := jwt.ParseWithClaims(tknStr, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		errcode := errGeneric
		if err == jwt.ErrSignatureInvalid {
			errcode = errSignatureInvalid
		}
		redirectToFallBack(res, req, errcode, origURL)
		return
	}
	if !tkn.Valid {
		redirectToFallBack(res, req, errTokenInvalid, origURL)
		return
	}
	if time.Unix(claims.ExpiresAt, 0).Sub(time.Now()) > time.Duration(minutesExpire)*time.Minute {
		redirectToFallBack(res, req, errExpired, origURL)
		return
	}

	// here we finally OK
	serveReverseProxy(res, req, tmpTargetURL)
	log.Printf("OK: Proxying to url: %s %s\n", tmpTargetURL, req.URL.String())
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

	log.Printf("Saturn signing key obtained and has length %d bytes: \"%20.20s...\"", len(sharedKey), sharedKey)

	tmpKey, err := generateCookieSigningKey(keyLength)
	if err != nil {
		log.Printf("Critical error: unable to generate JWT signing key")
		return
	}

	log.Printf("JWT signing key generated and has length %d bytes: \"%20.20s...\"", len(tmpKey), tmpKey)

	jwtKey = tmpKey
	minutesExpire, _ = strconv.Atoi(getEnv("JWT_EXPIRE_MIN", "60"))
	log.Printf("JWT cookie expiration time: %d", minutesExpire)

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
