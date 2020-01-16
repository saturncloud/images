package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"crypto/rand"
	"net/url"
	"os"
	"strconv"
	"time"
	"fmt"
	"github.com/dgrijalva/jwt-go"
)

//SECURITY NOTICE: Some older versions of Go have a security issue in the cryotp/elliptic. 
//Recommendation is to upgrade to at least 1.8.3.

var defaultURL = "http://localhost"    // resource default url. port 80 of the localhost
var fallbackURL = "http://localhost/fallback"   // not authorized fallback
var jwtKey = []byte("my_jwt_key")
var minutesExpire = 60  //  1 hour to expiration
var debug = true
var defaultPort = "8080"


const (
   errGeneric = 0
   errNoCookie = 1
   errBadSignature = 2
   errExpired = 3
   errTokenInvalid = 4
   errSignatureInvalid  = 5

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
	proxy.ServeHTTP(res, req)  // non blocking
}


type extendedClaims struct {
	resourceAddress string `json:"resource"`
	isShared string `json:"shared"`
	jwt.StandardClaims
}


func setNewCookie(res http.ResponseWriter, req *http.Request) {

	claims := &extendedClaims{}
	expirationTime := time.Now().Add(time.Duration(minutesExpire)*time.Minute)
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


func handleRequestAndSetCookie(res http.ResponseWriter, req *http.Request) {
	setNewCookie(res,req)
	fmt.Fprintf(res, "Hello, cookie is set and will be valid for %d minuted", minutesExpire)

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


func redirectToFallBack(res http.ResponseWriter, req *http.Request, error int) {

	if debug {
		log.Printf("fallback code: %d", error)
	}
	switch error {
		case errNoCookie:
			log.Printf( "error: no cookie token presented or bad cookie")
		case errExpired:
			log.Printf("error: cookie expired")
		case errSignatureInvalid:
			log.Printf("error: Unauthorized attempt to access resource: wrong signature\n")
		case errTokenInvalid:
			log.Printf("error: Unauthorized attempt to access resource: invalid token\n")
		case errGeneric:
			log.Printf( "error: bad request")
	}

	if debug {
		log.Printf("Redirecting to fallback url: %s", fallbackURL)
	}
	http.Redirect(res, req, fallbackURL, 301)
}


// Given a request send it to the appropriate url
func handleRequestAndRedirect(res http.ResponseWriter, req *http.Request) {

	c, err := req.Cookie("saturn_token")
	if err != nil {

		errcode := errGeneric
		if err == http.ErrNoCookie {
			errcode=errNoCookie
		}

		redirectToFallBack(res,req, errcode)
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
		redirectToFallBack(res,req,errcode)
		return
	}
	if !tkn.Valid {
		redirectToFallBack(res,req,errTokenInvalid)
		return
	}
	if time.Unix(claims.ExpiresAt, 0).Sub(time.Now()) > time.Duration(minutesExpire)*time.Minute {
		redirectToFallBack(res,req,errExpired)
		return
		}

	targetURL := defaultURL	
	if len(claims.resourceAddress)	!=0 {
			targetURL = claims.resourceAddress
	}

	// here we finally OK
	serveReverseProxy(targetURL, res, req)
	log.Printf("OK: Proxying to url: %s\n", targetURL)
}

func main() {

	defaultURL = getEnv("PROXY_RESOURCE_URL",defaultURL)
	fallbackURL = getEnv("PROXY_fallbackURL",
		"http://localhost:"+getEnv("PROXY_LISTEN_PORT", defaultPort)+"/fallback")

	tmpKey, err := generateCookieSigningKey(4096/8)
	if err != nil {
		log.Printf("Critical error: unable to generate JWT signing key")
		return
	}

	log.Printf("JWT signing key generated and has length %d bytes: \"%20.20s...\"" , len(tmpKey),tmpKey)


	jwtKey = tmpKey
	minutesExpire , _ = strconv.Atoi(getEnv("JWT_minutesExpire","60"))
	listAddr :=  ":" + getEnv("PROXY_LISTEN_PORT", defaultPort)
	log.Printf("Listening on %s",listAddr)
	
	log.Printf("Default target URL: %s",defaultURL)
	log.Printf("Fallback URL:       %s",fallbackURL)

	http.HandleFunc("/test", handleRequestAndSetCookie)
	http.HandleFunc("/fallback", handleRequestFallBack)
	http.HandleFunc("/resource", handleRequestAndRedirect)
	
	
	err = http.ListenAndServe(listAddr, nil) 
	if err != nil {
		panic(err)
	}
}

