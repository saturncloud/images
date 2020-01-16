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

var default_url = "http://localhost"    // resource default url. port 80 of the localhost
var fallback_url = "http://localhost/fallback"   // not authorized fallback
var jwtKey = []byte("my_jwt_key")
var minutes_expire = 60  //  1 hour to expiration
var debug = true
var defalut_port = "8080"

const (
   ErrGeneric = 0
   ErrNoCookie = 1
   ErrBadSignature = 2
   ErrExpired = 3
   ErrTokenInvalid = 4
   ErrSignatureInvalid  = 5

)


// Get env var or default
func getEnv(key, dflt string) string {
	value, ok := os.LookupEnv(key) 
	if ok {
		return value
	}
	return dflt
}

func GenerateCookieSigningKey(n int) ([]byte, error) {
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
func serveReverseProxy(target_url string, res http.ResponseWriter, req *http.Request) {
	url, _ := url.Parse(target_url)
	proxy := httputil.NewSingleHostReverseProxy(url)

	// Modify the headers for  SSL redirection
	req.URL.Host = url.Host
	req.URL.Scheme = url.Scheme
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
	req.Host = url.Host
	proxy.ServeHTTP(res, req)  // non blocking
}


type ExtendedClaims struct {
	resource_address string `json:"resource"`
	is_shared string `json:"shared"`
	jwt.StandardClaims
}


func setNewCookie(res http.ResponseWriter, req *http.Request) {

	claims := &ExtendedClaims{}
	expirationTime := time.Now().Add(time.Duration(minutes_expire)*time.Minute)
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
	fmt.Fprintf(res, "Hello, cookie is set and will be valid for %d minuted", minutes_expire)

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
		case ErrNoCookie:
			log.Printf( "Error: no cookie token presented or bad cookie")
		case ErrExpired:
			log.Printf("Error: cookie expired")
		case ErrSignatureInvalid:
			log.Printf("Error: Unauthorized attempt to access resource: wrong signature\n")
		case ErrTokenInvalid:
			log.Printf("Error: Unauthorized attempt to access resource: invalid token\n")
		case ErrGeneric:
			log.Printf( "Error: bad request")
	}

	if debug {
		log.Printf("Redirecting to fallback url: %s", fallback_url)
	}
	http.Redirect(res, req, fallback_url, 301)
}


// Given a request send it to the appropriate url
func handleRequestAndRedirect(res http.ResponseWriter, req *http.Request) {

	c, err := req.Cookie("saturn_token")
	if err != nil {

		errcode := ErrGeneric
		if err == http.ErrNoCookie {
			errcode=ErrNoCookie
		}

		redirectToFallBack(res,req, errcode)
		return
	}

	// Get the JWT string from the cookie
	tknStr := c.Value
	claims := &ExtendedClaims{}

	// Parse the JWT string and store the result in `claims`.
	tkn, err := jwt.ParseWithClaims(tknStr, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		errcode := ErrGeneric
		if err == jwt.ErrSignatureInvalid {
			errcode = ErrSignatureInvalid
		}
		redirectToFallBack(res,req,errcode)
		return
	}
	if !tkn.Valid {
		redirectToFallBack(res,req,ErrTokenInvalid)
		return
	}
	if time.Unix(claims.ExpiresAt, 0).Sub(time.Now()) > time.Duration(minutes_expire)*time.Minute {
		redirectToFallBack(res,req,ErrExpired)
		return
		}

	target_url := default_url	
	if len(claims.resource_address)	!=0 {
			target_url = claims.resource_address
	}

	// here we finally OK
	serveReverseProxy(target_url, res, req)
	log.Printf("OK: Proxying to url: %s\n", target_url)
}

func main() {

	default_url = getEnv("PROXY_RESOURCE_URL",default_url)
	fallback_url = getEnv("PROXY_FALLBACK_URL",
		"http://localhost:"+getEnv("PROXY_LISTEN_PORT", defalut_port)+"/fallback")

	tmp_key, err := GenerateCookieSigningKey(4096/8)
	if err != nil {
		log.Printf("Critical error: unable to generate JWT signing key")
		return
	} else {
		log.Printf("JWT signing key generated and has length %d bytes: \"%20.20s...\"" , len(tmp_key),tmp_key)
	}

	jwtKey = tmp_key
	minutes_expire , _ = strconv.Atoi(getEnv("JWT_MINUTES_EXPIRE","60"))
	list_addr :=  ":" + getEnv("PROXY_LISTEN_PORT", defalut_port)
	log.Printf("Listening on %s",list_addr)
	
	log.Printf("Default target URL: %s",default_url)
	log.Printf("Fallback URL:       %s",fallback_url)

	http.HandleFunc("/test", handleRequestAndSetCookie)
	http.HandleFunc("/fallback", handleRequestFallBack)
	http.HandleFunc("/resource", handleRequestAndRedirect)
	
	
	err = http.ListenAndServe(list_addr, nil) 
	if err != nil {
		panic(err)
	}
}

