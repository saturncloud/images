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

// HTTPConfig stores a map of requested domain prefix to k8s service address
type HTTPConfig struct {
	TargetMap map[string]string
	mutex     sync.Mutex
}

// GetTarget returns the service address for requested domain
func (hc *HTTPConfig) GetTarget(key string) string {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()
	target, ok := hc.TargetMap[key]
	if !ok {
		return ""
	}
	return target
}

// Load proxy target map from a ConfigMap
func (hc *HTTPConfig) Load(configmap interface{}) error {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	data, err := loadConfigMap(configmap)
	if err != nil {
		return err
	}

	// Check for changes to avoid spamming logs with "Loaded proxy config"
	targetsUpdated := len(hc.TargetMap) != len(data)
	for key, val := range data {
		if old, ok := hc.TargetMap[key]; !ok || old != val {
			targetsUpdated = true
		}
	}

	if targetsUpdated {
		log.Println("Loaded proxy config:")
		hc.TargetMap = data
		if len(hc.TargetMap) == 0 {
			log.Println("No proxy targets")
		}
		for key, val := range hc.TargetMap {
			log.Printf("Destination '%s' --> url '%s' \n", key, val)
		}
	}
	return nil
}

// Watch configures a configmap watcher to stream updates from k8s
func (hc *HTTPConfig) Watch(name, namespace string, client *kubernetes.Clientset) {
	targetWatcher := NewConfigWatcher(
		configKinds.configMap,
		namespace,
		client,
	)
	go targetWatcher.Watch(kubecache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			// Update proxy target config
			hc.Load(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			// Update proxy target config
			hc.Load(newObj)
		},
		DeleteFunc: func(obj interface{}) {
			// Clear proxy target config
			hc.mutex.Lock()
			defer hc.mutex.Unlock()
			hc.TargetMap = make(map[string]string)
			log.Println("Deleted proxy target configuration")
		},
	}, func(options *metav1.ListOptions) {
		options.FieldSelector = fmt.Sprintf("metadata.name=%s", name)
	})
}

// SessionConfig stores the set of active user session tokens
type SessionConfig struct {
	UserSessions map[string]struct{}
	mutex        sync.Mutex
}

// CheckSession returns true if user has active session
func (sc *SessionConfig) CheckSession(sessionToken string) bool {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	_, ok := sc.UserSessions[sessionToken]
	return ok
}

// Load user proxy sessions from configmap
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
	for key := range data {
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

// Watch configures a configmap watcher to stream updates from k8s
func (sc *SessionConfig) Watch(name, namespace string, client *kubernetes.Clientset) {
	sessionWatcher := NewConfigWatcher(
		configKinds.configMap,
		namespace,
		client,
	)
	go sessionWatcher.Watch(kubecache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			// Update user sessions
			sc.Load(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			// Update user sessions
			sc.Load(newObj)
		},
		DeleteFunc: func(obj interface{}) {
			// Clear user sessions
			sc.mutex.Lock()
			defer sc.mutex.Unlock()
			sc.UserSessions = make(map[string]struct{})
			log.Println("Deleted user sessions")
		},
	}, func(options *metav1.ListOptions) {
		options.FieldSelector = fmt.Sprintf("metadata.name=%s", name)
	})
}

// NewHAProxyConfig creates an HAProxyConfig object for TCP proxy
func NewHAProxyConfig(namespace, clusterDomain, baseDir, pidFile string, defaultListeners []int) *HAProxyConfig {
	certsDir := filepath.Join(baseDir, "certs")
	configDir := filepath.Join(baseDir, "config")
	pending := make(chan bool, 1)
	os.MkdirAll(configDir, os.ModeDir)

	hpc := &HAProxyConfig{
		TCPTargetMap:     map[string]*TCPTarget{},
		TLSSecrets:       NewTLSSecrets(certsDir, pending),
		DefaultListeners: defaultListeners,
		Namespace:        namespace,
		ClusterDomain:    clusterDomain,
		BaseDir:          baseDir,
		ConfigDir:        configDir,
		PIDFile:          pidFile,
		pending:          pending,
	}
	if len(hpc.DefaultListeners) > 0 {
		// Signal for update
		select {
		case hpc.pending <- true:
		default:
		}
	}
	return hpc
}

// HAProxyConfig stores TCP target and TLS configuration settings for HAProxy
type HAProxyConfig struct {
	TCPTargetMap map[string]*TCPTarget
	TLSSecrets   *TLSSecrets
	// Set of ports to listen on regardless of targets
	DefaultListeners []int
	Namespace        string
	ClusterDomain    string
	BaseDir          string
	ConfigDir        string
	PIDFile          string

	pending chan bool
	mutex   sync.Mutex
}

// Load HAProxy configuration from configmap
func (hpc *HAProxyConfig) Load(configmap interface{}) error {
	hpc.mutex.Lock()
	defer hpc.mutex.Unlock()

	data, err := loadConfigMap(configmap)
	if err != nil {
		return err
	}

	// Load and check for changes
	tcpTargets := make(map[string]*TCPTarget)
	targetsUpdated := false
	for hostname, targetYAML := range data {
		if targetYAML == "" {
			// Sometimes deleted entries end up blank instead of removed. No need to log about it.
			continue
		}
		target, err := NewTCPTarget(targetYAML, hpc.Namespace, hpc.ClusterDomain)
		if err != nil {
			log.Printf("Error loading %s TCP config: %s", hostname, err.Error())
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
		case hpc.pending <- true:
		default:
		}
	}
	return nil
}

// Update generates HAProxy config, writes TLS certs to disk, and soft-reloads HAProxy
func (hpc *HAProxyConfig) Update() error {
	hpc.mutex.Lock()
	defer hpc.mutex.Unlock()
	hpc.TLSSecrets.mutex.Lock()
	defer hpc.TLSSecrets.mutex.Unlock()

	configFile := filepath.Join(hpc.ConfigDir, "haproxy.cfg")

	configData := map[int][]map[string]string{}
	for _, port := range hpc.DefaultListeners {
		configData[port] = []map[string]string{}
	}
	for hostname, target := range hpc.TCPTargetMap {
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

		if _, ok := configData[target.Port]; !ok {
			configData[target.Port] = []map[string]string{}
		}
		configData[target.Port] = append(configData[target.Port], map[string]string{
			"hostname":       hostname,
			"serviceAddress": fmt.Sprintf("%s:%d", target.ServiceFQDN, target.ServicePort),
			"serviceName":    target.ServiceName,
			"certBundle":     tlsConfig.BundleFilepath,
			"caFile":         tlsConfig.CAFilepath,
		})
	}

	log.Printf("Writing HAProxy config to %s", configFile)
	if err := renderTemplate(haproxyConfigTemplate, configFile, configData); err != nil {
		return fmt.Errorf("Failed to render HAProxy config template: %s", err)
	}

	if err := hpc.reload(); err != nil {
		return fmt.Errorf("Failed to reload HAProxy: %s", err)
	}
	return nil
}

// Clear removes all TCP target configuration
func (hpc *HAProxyConfig) Clear() {
	hpc.mutex.Lock()
	defer hpc.mutex.Unlock()

	hpc.TCPTargetMap = map[string]*TCPTarget{}
	log.Println("Removed HAProxy configuration")

	// Signal for update
	select {
	case hpc.pending <- true:
	default:
	}
}

// getPID returns the PID of the HAProxy master process
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

// reload signals HAProxy master process to soft-reload its config
func (hpc *HAProxyConfig) reload() error {
	pid, err := hpc.getPID()
	if err != nil {
		return err
	}

	return syscall.Kill(pid, syscall.SIGUSR2)
}

// Watch configures a configmap and secrets watcher to stream updates from k8s
// 	It also starts an extra goroutine to syncronize updating the HAProxy process
// 	based on configmap and secret changes.
func (hpc *HAProxyConfig) Watch(
	name,
	namespace,
	tlsLabelSelector string,
	reloadRateLimit time.Duration,
	client *kubernetes.Clientset,
) {
	// Watch for changes to TCP proxy configmap
	haproxyWatcher := NewConfigWatcher(
		configKinds.configMap,
		namespace,
		client,
	)
	go haproxyWatcher.Watch(kubecache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			// Update HAProxy config
			hpc.Load(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			// Update HAProxy config
			hpc.Load(newObj)
		},
		DeleteFunc: func(obj interface{}) {
			// Clear HAProxy config
			hpc.Clear()
		},
	}, func(options *metav1.ListOptions) {
		options.FieldSelector = fmt.Sprintf("metadata.name=%s", name)
	})

	// Watch for changes to TLS secrets
	tlsSecretWatcher := NewConfigWatcher(
		configKinds.secret,
		namespace,
		client,
	)
	go tlsSecretWatcher.Watch(kubecache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			// Add TLS Secret
			hpc.TLSSecrets.Load(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			// Update TLS Secret
			hpc.TLSSecrets.Load(newObj)
		},
		DeleteFunc: func(obj interface{}) {
			// Remove TLS Secret
			hpc.TLSSecrets.Delete(obj)
		},
	}, func(options *metav1.ListOptions) {
		options.LabelSelector = tlsLabelSelector
	})

	// Watch for pending updates in haproxy config + TLS certs
	go func() {
		exit := make(chan os.Signal, 0)
		signal.Notify(exit, os.Kill, os.Interrupt)
		ticker := time.NewTicker(reloadRateLimit)
		for {
			select {
			case <-exit:
				break
			case <-hpc.pending:
				// Rate limit updates, while still checking for exit signal
				select {
				case <-exit:
					break
				case <-ticker.C:
				}
				// Update and reload HAProxy
				if err := hpc.Update(); err != nil {
					log.Printf("Error: %s", err)
					// Retry on failure
					select {
					case hpc.pending <- true:
					default:
					}
				}
			}
		}
	}()
}

// NewTCPTarget unmarshals YAML configuration for a TCP target and parses the service FQDN
func NewTCPTarget(targetYAML, namespace, clusterDomain string) (*TCPTarget, error) {
	var tt TCPTarget
	err := yaml.Unmarshal([]byte(targetYAML), &tt)
	if err != nil {
		return nil, fmt.Errorf("Failed to load TCP target YAML")
	}
	if tt.Port == 0 || tt.ServicePort == 0 || tt.ServiceName == "" {
		return nil, fmt.Errorf("Invalid TCP target configuration")
	}

	switch len(strings.Split(tt.ServiceName, ".")) {
	case 1:
		tt.ServiceFQDN = fmt.Sprintf("%s.%s.svc.%s", tt.ServiceName, namespace, clusterDomain)
	case 2:
		tt.ServiceFQDN = fmt.Sprintf("%s.svc.%s", tt.ServiceName, clusterDomain)
	default:
		if !strings.HasSuffix(tt.ServiceName, fmt.Sprintf(".svc.%s", clusterDomain)) {
			return nil, fmt.Errorf("Invalid service name \"%s\"", tt.ServiceName)
		}
	}
	return &tt, nil
}

// TCPTarget stores information for a TCP backend service
type TCPTarget struct {
	Port        int    `yaml:"port"`
	ServiceName string `yaml:"serviceName"`
	ServicePort int    `yaml:"servicePort"`
	SecretName  string `yaml:"secretName"`

	// Required by HAProxy DNS resolver
	ServiceFQDN string
}

// Compare returns true if the TCPTargets match
func (tt *TCPTarget) Compare(target *TCPTarget) bool {
	return tt.Port == target.Port &&
		tt.ServiceName == target.ServiceName &&
		tt.ServicePort == target.ServicePort &&
		tt.SecretName == target.SecretName
}

// NewTLSSecrets returns a TLSSecrets struct and ensures the certsDir exists
func NewTLSSecrets(certsDir string, pending chan bool) *TLSSecrets {
	os.MkdirAll(certsDir, 0600)
	return &TLSSecrets{
		TLSMap:   map[string]*TLSConfig{},
		CertsDir: certsDir,
		pending:  pending,
	}
}

// TLSSecrets stores TLS certificates mapped by secret name
type TLSSecrets struct {
	TLSMap   map[string]*TLSConfig
	CertsDir string
	pending  chan bool
	mutex    sync.Mutex
}

// Load TLS certificates from Secret
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

// Delete removes a secret from the TLSMap, and deletes associated cert files
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
		// so just logging the error is fine. Removing the TLSConfig from the map will make any related
		// TCP proxy get skipped on next update, so the certficiates will not be in use.
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

// TLSConfig stores TLS certificate bytes and paths
type TLSConfig struct {
	Cert []byte
	Key  []byte
	CA   []byte

	BundleFilepath string
	CAFilepath     string
}

// Compare returns true if the TLSConfigs match
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

// Write creates/updates TLS certificate files on disk
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

// NewConfigWatcher creates a new watcher for ConfigMaps or Secrets
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

// ConfigWatcher streams changes to ConfigMaps or Secrets
type ConfigWatcher struct {
	Kind      string
	Namespace string
	client    *kubernetes.Clientset
}

// Watch for updates to proxy configmaps, using the given event handlers to process the data.
// 	This will block execution, so it should be run on its own goroutine.
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

// loadConfigMap loads the data from a configmap obj interface (as passed in by ResourceEventHandlerFuncs)
func loadConfigMap(configmapObj interface{}) (map[string]string, error) {
	configmap, ok := configmapObj.(*corev1.ConfigMap)
	if !ok {
		return nil, errors.New("Watcher returned non-configmap object")
	}
	return configmap.Data, nil
}

// loadSecret loads a secret from an interface (as passed in by ResourceEventHandlerFuncs)
func loadSecret(secretObj interface{}) (*corev1.Secret, error) {
	secret, ok := secretObj.(*corev1.Secret)
	if !ok {
		return nil, errors.New("Watcher returned non-secret object")
	}
	return secret, nil
}

// verifyVery validates a certificate for a given CA and hostname
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
