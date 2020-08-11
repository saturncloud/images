package proxy

import (
	"errors"
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
		Name:      name,
		Namespace: namespace,
		client:    client,
	}
}

type ConfigWatcher struct {
	Name      string
	Namespace string
	client    *kubernetes.Clientset
}

/*
	Watch for updates to proxy configmaps, using the given event handlers to process the data.
	This will block execution, so it should be run on its own goroutine.
*/
func (cw *ConfigWatcher) Watch(eventHandler kubecache.ResourceEventHandlerFuncs) {

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
func loadConfigMap(configmap interface{}) (map[string]string, error) {
	cfg, ok := configmap.(*corev1.ConfigMap)
	if !ok {
		return nil, errors.New("Watcher returned non-configmap object")
	}
	return cfg.Data, nil
}
