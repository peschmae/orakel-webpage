package k8s

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	v1 "k8s.io/api/core/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	//
	// Uncomment to load all auth plugins
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	//
	// Or uncomment to load specific auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	//_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

func getKubeConfig() (*rest.Config, error) {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()

	if err != nil {
		homeDir, _ := os.UserHomeDir()
		config, err = clientcmd.BuildConfigFromFlags("", filepath.Join(homeDir, ".kube", "config"))

		if err != nil {
			return nil, fmt.Errorf("couldn't load configuration to connect to cluster: %v", err)
		}
	}

	return config, nil
}

func GetClientset() (*kubernetes.Clientset, error) {
	config, err := getKubeConfig()
	if err != nil {
		return nil, fmt.Errorf("couldn't get kubeconfig: %v", err)

	}

	// Increate QPS and Burst to avoid throttling
	config.QPS = 1000
	config.Burst = 1000

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("couldn't create kubernetes client: %v", err)
	}

	return clientset, nil

}

func GetRestClient() (*rest.RESTClient, error) {
	config, err := getKubeConfig()
	if err != nil {
		return nil, fmt.Errorf("couldn't get kubeconfig: %v", err)
	}

	restClient, err := rest.RESTClientFor(config)
	if err != nil {
		return nil, fmt.Errorf("couldn't create REST client: %v", err)
	}

	return restClient, nil
}

func GetDiscoveryClient() (*discovery.DiscoveryClient, error) {
	config, err := getKubeConfig()
	if err != nil {
		return nil, fmt.Errorf("couldn't get kubeconfig: %v", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("couldn't create discovery client: %v", err)
	}

	return discoveryClient, nil
}

func GetCrdClient() (*apiextensionsclientset.Clientset, error) {
	config, err := getKubeConfig()
	if err != nil {
		return nil, err
	}

	return apiextensionsclientset.NewForConfig(config)
}

func NamespaceExists(namespace string) (bool, error) {
	client, err := GetClientset()
	if err != nil {
		return false, fmt.Errorf("couldn't get kubernetes client: %v", err)
	}

	_, err = client.CoreV1().Namespaces().Get(context.TODO(), namespace, metav1.GetOptions{})
	if err != nil {
		return false, nil
	}

	return true, nil
}

func GetNodes() (*v1.NodeList, error) {
	client, err := GetClientset()
	if err != nil {
		return nil, fmt.Errorf("couldn't get kubernetes client: %v", err)
	}

	nodes, err := client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("couldn't list nodes: %v", err)
	}

	return nodes, nil
}
