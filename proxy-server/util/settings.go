package util

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v2"
)

// LoadSettings sets defaults, and parses an input YAML file into a Settings struct
func LoadSettings(settingsFile string) (*Settings, error) {
	// Set defaults
	haproxy := &HAProxy{
		Enabled:            false,
		BaseDir:            "/etc/haproxy",
		PIDFile:            "/etc/haproxy/haproxy.pid",
		ReloadRateLimitStr: "3s",
		TLSLabelSelector:   "saturncloud.io/certificate=server",
		DefaultListeners:   []int{},
	}
	proxyConfigMaps := &ProxyConfigMaps{
		HTTPTargets:  "saturn-auth-proxy",
		TCPTargets:   "saturn-tcp-proxy",
		UserSessions: "saturn-proxy-sessions",
	}
	proxyURLs := &ProxyURLs{
		BaseURL:      "http://dev.localtest.me:8888",
		LoginPath:    "/api/auth/login",
		RefreshPath:  "/auth/refresh",
		TokenPath:    "/api/deployments/auth",
		CommonSuffix: ".localtest.me",
	}
	settings := &Settings{
		AccessKeyExpirationStr:    "10m",
		ClusterDomain:             "cluster.local",
		Debug:                     false,
		HAProxy:                   haproxy,
		HTTPSRedirect:             false,
		Namespace:                 getEnv("NAMESPACE", "main-namespace"),
		ProxyConfigMaps:           proxyConfigMaps,
		ProxyPort:                 8080,
		ProxyURLs:                 proxyURLs,
		RefreshTokenExpirationStr: "3600s",
		SaturnTokenExpirationStr:  "86400s",

		SharedKey: []byte(getEnv("PROXY_SHARED_KEY", "")),
		KeyLength: 512 / 8,
	}

	// Load YAML
	absFilePath, err := filepath.Abs(settingsFile)
	if err != nil {
		return nil, err
	}
	settingsYAML, err := ioutil.ReadFile(absFilePath)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal([]byte(settingsYAML), settings); err != nil {
		return nil, err
	}
	if err := settings.parse(); err != nil {
		return nil, err
	}

	return settings, nil
}

// Settings stores configuration for the proxy server
type Settings struct {
	// YAML file settings
	ClusterDomain             string           `yaml:"clusterDomain"`
	Debug                     bool             `yaml:"debug"`
	HAProxy                   *HAProxy         `yaml:"haProxy"`
	HTTPSRedirect             bool             `yaml:"httpsRedirect"`
	Namespace                 string           `yaml:"namespace"`
	ProxyConfigMaps           *ProxyConfigMaps `yaml:"proxyConfigMaps"`
	ProxyPort                 int              `yaml:"proxyPort"`
	ProxyURLs                 *ProxyURLs       `yaml:"proxyURLs"`
	AccessKeyExpirationStr    string           `yaml:"accessKeyExpiration"`
	RefreshTokenExpirationStr string           `yaml:"refreshTokenExpiration"`
	SaturnTokenExpirationStr  string           `yaml:"saturnTokenExpiration"`

	// Parsed settings
	ListenAddr             string
	AccessKeyExpiration    time.Duration
	RefreshTokenExpiration time.Duration
	SaturnTokenExpiration  time.Duration
	SharedKey              []byte
	JWTKey                 []byte
	KeyLength              int
}

func (s *Settings) parse() error {
	var err error

	// Keys
	lenSharedKey := len(s.SharedKey)
	if lenSharedKey == 0 {
		if s.Debug {
			s.SharedKey = []byte("debugKeyForTestOnlydNeverUseInProduction123456789012345678901234567890")
			log.Printf("WARNING! WARNING! Running in debug mode with predefined weak key!\n - set debug=false if you are running this in production. ")
		} else {
			return fmt.Errorf("Critical error: unable to obtain shared saturn signing key.\n - set environment variable PROXY_SHARED_KEY")
		}
	} else if lenSharedKey < s.KeyLength {
		return fmt.Errorf("Critical error: shared saturn signing key is too short (%d),\n - set environment variable PROXY_SHARED_KEY", lenSharedKey)
	}
	log.Printf("Saturn signing key obtained and has length %d bytes", lenSharedKey)
	if s.JWTKey, err = s.GenerateCookieSigningKey(); err != nil {
		log.Panic("Critical error: unable to generate JWT signing key")
	}
	log.Printf("JWT signing key generated and has length %d bytes", len(s.JWTKey))

	// Expiration times
	if s.AccessKeyExpiration, err = time.ParseDuration(s.AccessKeyExpirationStr); err != nil {
		return fmt.Errorf("Invalid accessKeyExpiration: %s", err.Error())
	}
	if s.RefreshTokenExpiration, err = time.ParseDuration(s.RefreshTokenExpirationStr); err != nil {
		return fmt.Errorf("Invalid refreshTokenExpiration: %s", err.Error())
	}
	if s.SaturnTokenExpiration, err = time.ParseDuration(s.SaturnTokenExpirationStr); err != nil {
		return fmt.Errorf("Invalid saturnTokenExpiration: %s", err.Error())
	}
	log.Printf("Access key expiration time: %s", s.AccessKeyExpiration.String())
	log.Printf("JWT cookie expiration time: %s", s.SaturnTokenExpiration.String())
	log.Printf("Refresh cookie expiration time: %s", s.RefreshTokenExpiration.String())

	// URLs and ports
	s.ListenAddr = fmt.Sprintf(":%d", s.ProxyPort)
	if err = s.ProxyURLs.parse(); err != nil {
		return err
	}
	log.Printf("Listening on %s", s.ListenAddr)
	log.Printf("Redirect URL:    %s", s.ProxyURLs.Login)
	log.Printf("Refresh URL:     %s", s.ProxyURLs.Refresh)

	// HAProxy
	if err = s.HAProxy.parse(); err != nil {
		return err
	}
	log.Printf("HAProxy enabled: %v", s.HAProxy.Enabled)
	return nil
}

// GenerateCookieSigningKey create a key for signing JWTs
func (s *Settings) GenerateCookieSigningKey() ([]byte, error) {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-"
	b := make([]byte, s.KeyLength)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	for i, b1 := range b {
		b[i] = letters[b1%byte(len(letters))]
	}
	return b, nil
}

// ProxyURLs stores and parses URLs for interacting with Atlas
type ProxyURLs struct {
	BaseURL      string `yaml:"baseURL"`
	LoginPath    string `yaml:"loginPath"`
	RefreshPath  string `yaml:"refreshPath"`
	TokenPath    string `yaml:"tokenPath"`
	CommonSuffix string `yaml:"commonSuffix"`

	Base    *url.URL
	Login   *url.URL
	Refresh *url.URL
	Token   *url.URL
}

func (pu *ProxyURLs) parse() error {
	var err error
	if pu.Base, err = url.Parse(pu.BaseURL); err != nil {
		return err
	}
	if pu.Login, err = pu.Base.Parse(pu.LoginPath); err != nil {
		return err
	}
	if pu.Refresh, err = pu.Base.Parse(pu.RefreshPath); err != nil {
		return err
	}
	if pu.Token, err = pu.Base.Parse(pu.TokenPath); err != nil {
		return err
	}

	return nil
}

// ProxyConfigMaps configmap names for updating proxy sessions and targets
type ProxyConfigMaps struct {
	HTTPTargets  string `yaml:"httpTargets"`
	TCPTargets   string `yaml:"tcpTargets"`
	UserSessions string `yaml:"userSessions"`
}

// HAProxy settings for interacting with an HAProxy sidecar container for TCP proxy
type HAProxy struct {
	Enabled            bool   `yaml:"enabled"`
	BaseDir            string `yaml:"baseDir"`
	PIDFile            string `yaml:"pidFile"`
	ReloadRateLimitStr string `yaml:"reloadRateLimit"`
	TLSLabelSelector   string `yaml:"tlsLabelSelector"`
	DefaultListeners   []int  `yaml:"defaultListeners"`

	ReloadRateLimit time.Duration
}

func (h *HAProxy) parse() error {
	var err error
	if h.ReloadRateLimit, err = time.ParseDuration(h.ReloadRateLimitStr); err != nil {
		return err
	}
	return nil
}

// getEnv returns env var value or default
func getEnv(key, dflt string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return dflt
}
