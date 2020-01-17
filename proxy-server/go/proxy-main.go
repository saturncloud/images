package main

import (
	"crypto/rand"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"time"
)

//SECURITY NOTICE: Some older versions of Go have a security issue in the cryotp/elliptic.
//Recommendation is to upgrade to at least 1.8.3.

var defaultURL = "http://localhost"           // resource default url. port 80 of the localhost
var fallbackURL = "http://localhost/fallback" // not authorized fallback
var jwtKey = []byte("my_jwt_key")
var sharedKey = []byte("my_jwt_key")

var minutesExpire = 60 //  1 hour to expiration
var debug = true
var defaultPort = "8080"

const keyLength = 512 / 8

const (
	errGeneric          = iota
	errNoCookie         = iota
	errBadSignature     = iota
	errExpired          = iota
	errTokenInvalid     = iota
	errSignatureInvalid = iota
)

// Get env var or default
func getEnv(key, dflt string) string {
	value, ok := os.LookupEnv(key)
	if ok {
		return value
	}
	return dflt
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
func serveReverseProxy(targetURL string, res http.ResponseWriter, req *http.Request) {
	url, _ := url.Parse(targetURL)
	proxy := httputil.NewSingleHostReverseProxy(url)

	// Modify the headers for  SSL redirection
	req.URL.Host = url.Host
	req.URL.Scheme = url.Scheme
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
	req.Host = url.Host
	proxy.ServeHTTP(res, req) // non blocking
}

type extendedClaims struct {
	resourceAddress string `json:"resource"`
	jwt.StandardClaims
}

func setNewCookie(res http.ResponseWriter, req *http.Request) {

	claims := &extendedClaims{}
	expirationTime := time.Now().Add(time.Duration(minutesExpire) * time.Minute)
	claims.ExpiresAt = expirationTime.Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Set the new token as the users `saturn_token` cookie
	http.SetCookie(res, &http.Cookie{
		Name:    "saturn_token",
		Value:   tokenString,
		Expires: expirationTime,
	})
}

// The default fallback handler.
func handleRequestFallBack(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(http.StatusUnauthorized)
	//		res.WriteHeader(http.StatusBadRequest)
	fmt.Fprintf(res, "\nWe are sorry, access denied.")
	return
}

// Mocked resource handler.
func handleRequestMockResource(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(http.StatusOK)
	if debug {
		log.Printf("Mocked resorce url hit")
	}

	fmt.Fprintf(res, "\nOK, access granted.")
	return
}

func validateAuthToken(token *extendedClaims) bool {
	// token extra validation. its signature is already verified at this point
	// considerations: -
	//			check expiration time for this token
	//          any extra user access limitations
	return true
}

func redirectToFallBack(res http.ResponseWriter, req *http.Request, error int, origURL string) {

	if debug {
		log.Printf("fallback code: %d", error)
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

	url := fallbackURL + "?orig_request=" + origURL
	if debug {
		log.Printf("Redirecting to fallback url: %s", url)
	}

	http.Redirect(res, req, url, 301)
}

// Given a request send it to the appropriate url
func handleRequestAndRedirect(res http.ResponseWriter, req *http.Request) {

	targetURL := defaultURL

	// if we receive authorization token in the request : the user authorized by the backend
	satToken := req.URL.Query().Get("saturn_token")

	if satToken != "" {
		claims := &extendedClaims{}
		tkn, err := jwt.ParseWithClaims(satToken, claims, func(token *jwt.Token) (interface{}, error) {
			return sharedKey, nil
		})

		if err != nil || !tkn.Valid {
			log.Printf("Authorization Error: invalid token from saturn")
			// Breaking the loop. No more redirects. All is bad for this request.
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		if len(claims.resourceAddress) != 0 {
			targetURL = claims.resourceAddress
		}

		if validateAuthToken(claims) {
			setNewCookie(res, req)
			log.Printf("OK: Setting cookie")
			serveReverseProxy(targetURL, res, req)
			log.Printf("OK: Proxying to url: %s", targetURL)
		}
	}

	c, err := req.Cookie("saturn_token")
	origURL := req.Host + req.URL.Path

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
	claims := &extendedClaims{}

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

	if len(claims.resourceAddress) != 0 {
		targetURL = claims.resourceAddress
	}

	// here we finally OK
	serveReverseProxy(targetURL, res, req)
	log.Printf("OK: Proxying to url: %s\n", targetURL)
}

func main() {

	debug, _ = strconv.ParseBool(getEnv("PROXY_DEBUG", "true"))
	defaultURL = getEnv("PROXY_RESOURCE_URL", defaultURL)
	fallbackURL = getEnv("PROXY_fallbackURL",
		"http://localhost:"+getEnv("PROXY_LISTEN_PORT", defaultPort)+"/fallback")

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
	minutesExpire, _ = strconv.Atoi(getEnv("JWT_minutesExpire", "60"))
	listAddr := ":" + getEnv("PROXY_LISTEN_PORT", defaultPort)

	log.Printf("Listening on %s", listAddr)

	log.Printf("Default target URL: %s", defaultURL)
	log.Printf("Fallback URL:       %s", fallbackURL)

	http.HandleFunc("/fallback", handleRequestFallBack)
	http.HandleFunc("/resource", handleRequestAndRedirect)

	err = http.ListenAndServe(listAddr, nil)
	if err != nil {
		panic(err)
	}
}
