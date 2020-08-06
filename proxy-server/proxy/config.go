package proxy

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	kubecache "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

type ProxyConfig struct {
	UserSessions map[string]bool   `json:"user_sessions"`
	ServiceMap   map[string]string `json:"service_map"`
	mutex        sync.Mutex
}

/*
	Check if user has active session
*/
func (p *ProxyConfig) CheckUser(userID string) bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	_, ok := p.UserSessions[userID]
	return ok
}

/*
	Get service address for proxy target
*/
func (p *ProxyConfig) GetService(targetKey string) string {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	service, ok := p.ServiceMap[targetKey]
	if !ok {
		return ""
	}
	return service
}

func newProxyConfig() *ProxyConfig {
	return &ProxyConfig{
		UserSessions: make(map[string]bool),
		ServiceMap:   make(map[string]string),
	}
}

func NewConfigWatcher(name, namespace string) *ConfigWatcher {
	// Load kubeconfig
	kubeconfigPath := "~/.kube/config"
	var kubeconfig *rest.Config
	if _, err := os.Stat(kubeconfigPath); err == nil {
		kubeconfig, err = clientcmd.BuildConfigFromFlags("", "")
		if err != nil {
			log.Panic(err)
		}
	} else {
		kubeconfig, err = rest.InClusterConfig()
		if err != nil {
			log.Panic(err)
		}
	}

	// Create kube client
	client, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		log.Panic(err)
	}

	// Create watcher
	return &ConfigWatcher{
		Name:        name,
		Namespace:   namespace,
		ProxyConfig: newProxyConfig(),
		client:      client,
	}
}

type ConfigWatcher struct {
	Name        string
	Namespace   string
	ProxyConfig *ProxyConfig
	client      *kubernetes.Clientset
}

/*
	Watch for updates to proxy configmaps.
	This will block execution, so it should be run on its own goroutine.
*/
func (cw *ConfigWatcher) Watch() {

	// Resync period, triggers UpdateFunc even when no change events were found
	defaultResync := time.Second * 30

	// Create configmap informer with name/namespace filter
	kubeInformerFactory := kubeinformers.NewSharedInformerFactoryWithOptions(
		cw.client, defaultResync, kubeinformers.WithNamespace(cw.Namespace),
		kubeinformers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.FieldSelector = fmt.Sprintf("metadata.name=%s", cw.Name)
		}),
	)
	cfgInformer := kubeInformerFactory.Core().V1().ConfigMaps().Informer()

	// Add handler functions for add/update/delete
	cfgInformer.AddEventHandler(kubecache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			// Update proxy config
			cw.parseConfig(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			// Update proxy config
			cw.parseConfig(newObj)
		},
		DeleteFunc: func(obj interface{}) {
			// Clear proxy config
			cw.ProxyConfig.mutex.Lock()
			defer cw.ProxyConfig.mutex.Unlock()
			cw.ProxyConfig = newProxyConfig()
			log.Println("Deleted proxy configuration")
		},
	})

	// Start event watcher
	stop := make(chan struct{})
	defer close(stop)
	kubeInformerFactory.Start(stop)

	log.Println("Started factory")

	// Block until process ends
	sig := make(chan os.Signal, 0)
	signal.Notify(sig, os.Kill, os.Interrupt)
	<-sig
}

/*
	Load configmap data into struct
*/
func (cw *ConfigWatcher) parseConfig(configmap interface{}) {
	log.Println("Loading config")
	cfg, ok := configmap.(*corev1.ConfigMap)
	if !ok {
		log.Println("Watcher returned non-configmap object")
		return
	}

	cw.ProxyConfig.mutex.Lock()
	defer cw.ProxyConfig.mutex.Unlock()
	// proxyConfig := newProxyConfig()
	userSessions := make(map[string]bool)
	serviceMap := make(map[string]string)

	if userSessionsStr, ok := cfg.Data["user_sessions"]; ok {
		if err := json.Unmarshal([]byte(userSessionsStr), &userSessions); err != nil {
			log.Printf("Failed to unmarshal user sessions: %s", err)
		}
	}
	// Always set user sessions
	// if something went wrong we don't want to authenticate a logged-out user.
	cw.ProxyConfig.UserSessions = userSessions

	if serviceMapStr, ok := cfg.Data["service_map"]; ok {
		if err := json.Unmarshal([]byte(serviceMapStr), &serviceMap); err != nil {
			log.Printf("Failed to unmarshal service map: %s", err)
		} else {
			// Only set service map on successful unmarshal
			// Worst case on failure is some services might not be valid any more,
			// but others will still work.
			cw.ProxyConfig.ServiceMap = serviceMap
		}
	}

	log.Println("Loaded proxy config:")
	for k, v := range cw.ProxyConfig.ServiceMap {
		log.Printf("Destination '%s' --> url '%s' \n", k, v)
	}
}
