package main

import (
	"crypto/rand"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"
	"github.com/dgrijalva/jwt-go"
)


var targetURL = "http://localhost:80"         // protocol + host + port to be proxied to
var fallbackURL = "http://localhost/fallback" // not authorized fallback
var jwtKey []byte
var sharedKey []byte

var minutesExpire = 60 //  1 hour to expiration
var debug = true
var defaultPort = "8080"
var mutex sync.Mutex
var tokenMap = make(map[string]string)

const keyLength = 512 / 8

const (
	errGeneric = iota
	errNoCookie
	errBadSignature
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
func serveReverseProxy(res http.ResponseWriter, req *http.Request) {
	// Modify the headers for  SSL redirection
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))

	proxiedURL, _ := url.Parse(targetURL)
	proxy := httputil.NewSingleHostReverseProxy(proxiedURL)

	proxy.ServeHTTP(res, req) // non blocking
}

type extendedClaims struct {
	resourceAddress string
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
// func handleRequestFallBack(res http.ResponseWriter, req *http.Request) {
// 	res.WriteHeader(http.StatusUnauthorized)
// 	fmt.Fprintf(res, "\nWe are sorry, access denied.")
// 	return
// }

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

	uniqToken, _ := generateCookieSigningKey(40)

	mutex.Lock()
	tokenMap[string(uniqToken)] = ""
	log.Printf("Added token %s", uniqToken)
	mutex.Unlock()

	u := fallbackURL + "?orig_request=" + origURL + "&ret_token=" + string(uniqToken)

	if debug {
		log.Printf("Redirecting to fallback url: %s", u)
	}

	http.Redirect(res, req, u, 301)
}

// Given a request send it to the appropriate url
func handleRequestAndRedirect(res http.ResponseWriter, req *http.Request) {

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

		if len(claims.Issuer) == 0 {
			log.Printf("Authorization Error: invalid token from saturn (no issuers)")
			res.WriteHeader(http.StatusUnauthorized)
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
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		if validateAuthToken(claims) {
			setNewCookie(res, req)
			log.Printf("OK: Setting cookie")
			// 301 back to self to remove saturn_token from the URL
			q := req.URL.Query()
			q.Del("saturn_token")
			req.URL.RawQuery = q.Encode()
			http.Redirect(res, req, req.URL.String(), 301)
			log.Printf("OK: Redirecting to self (%s)", req.URL.String())
			return
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

	// here we finally OK
	serveReverseProxy(res, req)
	log.Printf("OK: Proxying to url: %s\n", req.URL.String())
}

type proxyServer struct{}

func (p *proxyServer) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	handleRequestAndRedirect(res, req)
}

func main() {

	debug, _ = strconv.ParseBool(getEnv("PROXY_DEBUG", "true"))
	targetURL = getEnv("PROXY_TARGET_URL", targetURL)
	fallbackURL = getEnv("PROXY_FALLBACK_URL",
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
	minutesExpire, _ = strconv.Atoi(getEnv("JWT_EXPIRE_MIN", "60"))
	log.Printf("JWT cookie expiration time: %d", minutesExpire)

	listAddr := ":" + getEnv("PROXY_LISTEN_PORT", defaultPort)

	log.Printf("Listening on %s", listAddr)

	log.Printf("Fallback URL:       %s", fallbackURL)

	err = http.ListenAndServe(listAddr, &proxyServer{})
	if err != nil {
		panic(err)
	}
}