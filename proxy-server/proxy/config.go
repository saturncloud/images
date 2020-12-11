package proxy

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	kubecache "k8s.io/client-go/tools/cache"
)

var configKinds = struct {
	configMap string
	secret    string
}{
	configMap: "ConfigMap",
	secret:    "Secret",
}

var tlsSecretKeys = struct {
	cert string
	key  string
	ca   string
}{
	cert: "tls.crt",
	key:  "tls.key",
	ca:   "ca.crt",
}

type ProxyConfig struct {
	// Map of requested domain prefix to k8s dst address
	TargetMap map[string]string
	mutex     sync.Mutex
}

/*
	Get service address for proxy target
*/
func (pc *ProxyConfig) GetTarget(key string) string {
	pc.mutex.Lock()
	defer pc.mutex.Unlock()
	target, ok := pc.TargetMap[key]
	if !ok {
		return ""
	}
	return target
}

/*
	Load proxy target map from a ConfigMap
*/
func (pc *ProxyConfig) Load(configmap interface{}) error {
	pc.mutex.Lock()
	defer pc.mutex.Unlock()

	data, err := loadConfigMap(configmap)
	if err != nil {
		return err
	}

	// Check for changes to avoid spamming logs with "Loaded proxy config"
	targetsUpdated := len(pc.TargetMap) != len(data)
	for key, val := range data {
		if old, ok := pc.TargetMap[key]; !ok || old != val {
			targetsUpdated = true
		}
	}

	if targetsUpdated {
		log.Println("Loaded proxy config:")
		pc.TargetMap = data
		if len(pc.TargetMap) == 0 {
			log.Println("No proxy targets")
		}
		for key, val := range pc.TargetMap {
			log.Printf("Destination '%s' --> url '%s' \n", key, val)
		}
	}
	return nil
}

type SessionConfig struct {
	// Set of user IDs that are actively logged in
	UserSessions map[string]struct{}
	mutex        sync.Mutex
}

/*
	Check if user has active session
*/
func (sc *SessionConfig) CheckUser(userID string) bool {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	_, ok := sc.UserSessions[userID]
	return ok
}

/*
	Load user proxy sessions from configmap
*/
func (sc *SessionConfig) Load(configmap interface{}) error {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	data, err := loadConfigMap(configmap)
	if err != nil {
		return err
	}

	// Load sessions and check for changes
	userSessions := make(map[string]struct{})
	sessionsUpdated := len(data) != len(sc.UserSessions)
	for key, _ := range data {
		userSessions[key] = struct{}{}
		if _, ok := sc.UserSessions[key]; !ok {
			sessionsUpdated = true
		}
	}

	// Update onchanges
	if sessionsUpdated {
		log.Printf("Loaded user sessions: %d active proxy session(s)", len(userSessions))
		sc.UserSessions = userSessions
	}
	return nil
}

func NewHAProxyConfig(namespace, clusterDomain, baseDir, pidFile string) *HAProxyConfig {
	certsDir := filepath.Join(baseDir, "certs")
	configDir := filepath.Join(baseDir, "config")
	pending := make(chan bool, 1)
	os.MkdirAll(configDir, os.ModeDir)

	return &HAProxyConfig{
		TCPTargetMap:  map[string]TCPTarget{},
		TLSSecrets:    NewTLSSecrets(certsDir, pending),
		Namespace:     namespace,
		ClusterDomain: clusterDomain,
		BaseDir:       baseDir,
		ConfigDir:     configDir,
		PIDFile:       pidFile,
		Pending:       pending,
	}
}

type HAProxyConfig struct {
	TCPTargetMap  map[string]TCPTarget
	TLSSecrets    *TLSSecrets
	Namespace     string
	ClusterDomain string
	BaseDir       string
	ConfigDir     string
	PIDFile       string

	Pending chan bool
	mutex   sync.Mutex
}

/*
	Load HAProxy configuration from configmap
*/
func (hpc *HAProxyConfig) Load(configmap interface{}) error {
	hpc.mutex.Lock()
	defer hpc.mutex.Unlock()

	data, err := loadConfigMap(configmap)
	if err != nil {
		return err
	}

	// Load and check for changes
	tcpTargets := make(map[string]TCPTarget)
	targetsUpdated := false
	for hostname, targetYAML := range data {
		var target TCPTarget
		err := yaml.Unmarshal([]byte(targetYAML), &target)
		if err != nil {
			log.Printf("Error: Failed to load HAProxy config for %s", hostname)
			continue
		}
		if target.Port == 0 || target.ServicePort == 0 || target.ServiceName == "" {
			log.Printf("Error: Invalid TCP target configuration for %s", hostname)
			continue
		}
		if err := target.setFQDN(hpc.Namespace, hpc.ClusterDomain); err != nil {
			log.Printf("Error: Failed to configure target FQDN for %s: %s", hostname, err)
			continue
		}

		tcpTargets[hostname] = target
		if existing, ok := hpc.TCPTargetMap[hostname]; ok {
			targetsUpdated = targetsUpdated || !existing.Compare(target)
		} else {
			targetsUpdated = true
		}
	}
	targetsUpdated = targetsUpdated || len(tcpTargets) != len(hpc.TCPTargetMap)

	// Update onchanges
	if targetsUpdated {
		hpc.TCPTargetMap = tcpTargets
		log.Printf("Loaded HAProxy config: %d TCP targets", len(tcpTargets))
		// Signal for update
		select {
		case hpc.Pending <- true:
		default:
		}
	}
	return nil
}

func (hpc *HAProxyConfig) Update() error {
	hpc.mutex.Lock()
	defer hpc.mutex.Unlock()
	hpc.TLSSecrets.mutex.Lock()
	defer hpc.TLSSecrets.mutex.Unlock()

	certListFile := filepath.Join(hpc.ConfigDir, "crt-list.txt")
	configFile := filepath.Join(hpc.ConfigDir, "haproxy.cfg")

	configData := map[int]map[string]string{}
	certListData := []map[string]string{}
	for hostname, target := range hpc.TCPTargetMap {
		if target.SecretName != "" {
			tlsConfig, ok := hpc.TLSSecrets.TLSMap[target.SecretName]
			if !ok {
				log.Printf("Skipping TCP target %s: Missing TLS secret %s", hostname, target.SecretName)
				continue
			} else if err := verifyCert(tlsConfig.Cert, tlsConfig.CA, hostname); err != nil {
				log.Printf("Skipping TCP target %s: %s", hostname, err.Error())
				continue
			}

			// Update certificates
			if err := tlsConfig.Write(); err != nil {
				log.Printf(
					"Error: Failed to write certificate from secret \"%s\" to %s",
					target.SecretName,
					hpc.TLSSecrets.CertsDir,
				)
				continue
			}

			// Add to crt-list
			certListData = append(certListData, map[string]string{
				"certBundle": tlsConfig.BundleFilepath,
				"caFile":     tlsConfig.CAFilepath,
				"hostname":   hostname,
			})
		}
		if _, ok := configData[target.Port]; !ok {
			configData[target.Port] = map[string]string{}
		}
		configData[target.Port][hostname] = fmt.Sprintf("%s:%d", target.ServiceFQDN, target.ServicePort)
	}

	log.Printf("Writing HAProxy config to %s", configFile)
	if err := renderTemplate(haproxyConfigTemplate, configFile, configData); err != nil {
		return fmt.Errorf("Failed to render HAProxy config template: %s", err)
	}
	if err := renderTemplate(haproxyCertListTemplate, certListFile, certListData); err != nil {
		return fmt.Errorf("Failed to render HAProxy cert list template: %s", err)
	}

	if err := hpc.reload(); err != nil {
		return fmt.Errorf("Failed to reload HAProxy: %s", err)
	}
	return nil
}

func (hpc *HAProxyConfig) Clear() {
	hpc.mutex.Lock()
	defer hpc.mutex.Unlock()

	hpc.TCPTargetMap = map[string]TCPTarget{}
	log.Println("Removed HAProxy configuration")

	// Signal for update
	select {
	case hpc.Pending <- true:
	default:
	}
}

func (hpc *HAProxyConfig) getPID() (int, error) {
	f, err := os.Open(hpc.PIDFile)
	if err != nil {
		return -1, err
	}
	defer f.Close()

	var pid int
	_, err = fmt.Fscan(f, &pid)
	return pid, err
}

/*
	Retrieve the PID of HAProxy and signal to reload its configuration
*/
func (hpc *HAProxyConfig) reload() error {
	pid, err := hpc.getPID()
	if err != nil {
		return err
	}

	return syscall.Kill(pid, syscall.SIGUSR2)
}

type TCPTarget struct {
	Port        int    `yaml:"port"`
	ServiceName string `yaml:"serviceName"`
	ServicePort int    `yaml:"servicePort"`
	SecretName  string `yaml:"secretName"`

	// Required by HAProxy DNS resolver
	ServiceFQDN string
}

/*
	Return true if the targets match
*/
func (tt *TCPTarget) Compare(target TCPTarget) bool {
	return tt.Port == target.Port &&
		tt.ServiceName == target.ServiceName &&
		tt.ServicePort == target.ServicePort &&
		tt.SecretName == target.SecretName
}

func (tt *TCPTarget) setFQDN(namespace, clusterDomain string) error {
	switch len(strings.Split(tt.ServiceName, ".")) {
	case 1:
		tt.ServiceFQDN = fmt.Sprintf("%s.%s.svc.%s", tt.ServiceName, namespace, clusterDomain)
	case 2:
		tt.ServiceFQDN = fmt.Sprintf("%s.svc.%s", tt.ServiceName, clusterDomain)
	default:
		if !strings.HasSuffix(tt.ServiceName, fmt.Sprintf(".svc.%s", clusterDomain)) {
			return fmt.Errorf("Invalid service name \"%s\"", tt.ServiceName)
		}
	}
	return nil
}

func NewTLSSecrets(certsDir string, pending chan bool) *TLSSecrets {
	os.MkdirAll(certsDir, 0600)
	return &TLSSecrets{
		TLSMap:   map[string]*TLSConfig{},
		CertsDir: certsDir,
		pending:  pending,
	}
}

type TLSSecrets struct {
	TLSMap   map[string]*TLSConfig
	CertsDir string
	pending  chan bool
	mutex    sync.Mutex
}

/*
	Load TLS certificates from Secret
*/
func (ts *TLSSecrets) Load(secretObj interface{}) error {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	secret, err := loadSecret(secretObj)
	if err != nil {
		log.Printf("Error: Failed to load secret: %s", err)
		return err
	}
	secretName := secret.ObjectMeta.Name

	// Load and check for changes
	tlsConfig := &TLSConfig{
		Cert:           secret.Data[tlsSecretKeys.cert],
		Key:            secret.Data[tlsSecretKeys.key],
		CA:             secret.Data[tlsSecretKeys.ca],
		BundleFilepath: fmt.Sprintf("%s/%s-bundle.pem", ts.CertsDir, secretName),
		CAFilepath:     fmt.Sprintf("%s/%s-ca.pem", ts.CertsDir, secretName),
	}

	if existing, ok := ts.TLSMap[secretName]; !ok || !existing.Compare(tlsConfig) {
		ts.TLSMap[secretName] = tlsConfig
		log.Printf("Loaded HAProxy certificate: %s", secretName)

		// Signal for update
		select {
		case ts.pending <- true:
		default:
		}
	}

	return nil
}

/*
	Remove secret from TLS configuration, and delete associated cert files
*/
func (ts *TLSSecrets) Delete(secretObj interface{}) error {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	secret, err := loadSecret(secretObj)
	if err != nil {
		log.Printf("Error: Failed to load secret: %s", err)
		return err
	}
	secretName := secret.ObjectMeta.Name

	if tlsConfig, ok := ts.TLSMap[secretName]; ok {
		delete(ts.TLSMap, secretName)

		// Removing the cert files is not the most critical part of deleting the TLS configuration,
		// so just logging the error is fine. Removing the TLSConfig from the map will remove it from
		// the crt-list.txt on next update, so the certificates will not be in use.
		if err := os.Remove(tlsConfig.BundleFilepath); err != nil {
			if !os.IsNotExist(err) {
				log.Printf("Error: Failed to delete %s certificate bundle: %s", secretName, err)
			}
		}
		if err := os.Remove(tlsConfig.CAFilepath); err != nil {
			if !os.IsNotExist(err) {
				log.Printf("Error: Failed to delete %s CA: %s", secretName, err)
			}
		}

		log.Printf("Removed TLS config for %s", secretName)

		// Signal for update
		select {
		case ts.pending <- true:
		default:
		}
	}

	return nil
}

type TLSConfig struct {
	Cert []byte
	Key  []byte
	CA   []byte

	BundleFilepath string
	CAFilepath     string
}

/*
	Return true if the TLSConfigs match
*/
func (tc *TLSConfig) Compare(config *TLSConfig) bool {
	if diff := bytes.Compare(tc.Cert, config.Cert); diff != 0 {
		return false
	}
	if diff := bytes.Compare(tc.Key, config.Key); diff != 0 {
		return false
	}
	if diff := bytes.Compare(tc.CA, config.CA); diff != 0 {
		return false
	}
	return true
}

func (tc *TLSConfig) Write() error {
	f, err := os.OpenFile(tc.BundleFilepath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(tc.Cert); err != nil {
		return err
	}
	if !bytes.HasSuffix(tc.Cert, []byte("\n")) {
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}
	if _, err := f.Write(tc.Key); err != nil {
		return err
	}

	f2, err := os.OpenFile(tc.CAFilepath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f2.Close()

	if _, err := f2.Write(tc.CA); err != nil {
		return err
	}

	return nil
}

/*
	Create a new watcher for ConfigMaps or Secrets
*/
func NewConfigWatcher(kind, namespace string, client *kubernetes.Clientset) *ConfigWatcher {
	if kind != configKinds.configMap && kind != configKinds.secret {
		log.Panicf("Invalid config kind \"%s\"", kind)
	}
	return &ConfigWatcher{
		Kind:      kind,
		Namespace: namespace,
		client:    client,
	}
}

/*
	Watches for changes to ConfigMaps or Secrets
*/
type ConfigWatcher struct {
	Kind      string
	Namespace string
	client    *kubernetes.Clientset
}

/*
	Watch for updates to proxy configmaps, using the given event handlers to process the data.
	This will block execution, so it should be run on its own goroutine.
*/
func (cw *ConfigWatcher) Watch(eventHandler kubecache.ResourceEventHandlerFuncs, listOptions func(options *metav1.ListOptions)) {

	// Resync period, triggers UpdateFunc even when no change events were found
	defaultResync := time.Second * 30

	// Create configmap informer with name/namespace filter
	kubeInformerFactory := kubeinformers.NewSharedInformerFactoryWithOptions(
		cw.client, defaultResync, kubeinformers.WithNamespace(cw.Namespace),
		kubeinformers.WithTweakListOptions(listOptions),
	)
	var cfgInformer cache.SharedIndexInformer
	if cw.Kind == configKinds.configMap {
		cfgInformer = kubeInformerFactory.Core().V1().ConfigMaps().Informer()
	} else {
		cfgInformer = kubeInformerFactory.Core().V1().Secrets().Informer()
	}

	// Add handler functions for add/update/delete
	cfgInformer.AddEventHandler(eventHandler)

	// Start event watcher
	stop := make(chan struct{})
	defer close(stop)
	kubeInformerFactory.Start(stop)

	// Block until process ends
	sig := make(chan os.Signal, 0)
	signal.Notify(sig, os.Kill, os.Interrupt)
	<-sig
}

/*
	Load the data from a configmap obj interface (as passed in by ResourceEventHandlerFuncs)
*/
func loadConfigMap(configmapObj interface{}) (map[string]string, error) {
	configmap, ok := configmapObj.(*corev1.ConfigMap)
	if !ok {
		return nil, errors.New("Watcher returned non-configmap object")
	}
	return configmap.Data, nil
}

/*
	Load a secret from an interface (as passed in by ResourceEventHandlerFuncs)
*/
func loadSecret(secretObj interface{}) (*corev1.Secret, error) {
	secret, ok := secretObj.(*corev1.Secret)
	if !ok {
		return nil, errors.New("Watcher returned non-secret object")
	}
	return secret, nil
}

/*
	Validate a certificate for a given CA and hostname
*/
func verifyCert(certPEM, caCertPEM []byte, hostname string) error {
	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM([]byte(caCertPEM))
	if !ok {
		return fmt.Errorf("Failed to parse CA certificate")
	}

	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return fmt.Errorf("Failed to decode server certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("Failed to parse server certificate: %s", err.Error())
	}

	opts := x509.VerifyOptions{
		DNSName:   hostname,
		Roots:     roots,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	if _, err := cert.Verify(opts); err != nil {
		return fmt.Errorf("Failed to verify certificate: %s", err.Error())
	}

	return nil
}
