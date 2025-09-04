package k8s

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
)

func GetDynamicClient() (*dynamic.DynamicClient, error) {
	config, err := getKubeConfig()
	if err != nil {
		return nil, err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return dynamicClient, nil

}

func mapGvkToGvr(gvk schema.GroupVersionKind) (schema.GroupVersionResource, error) {
	config, err := getKubeConfig()
	if err != nil {
		return schema.GroupVersionResource{}, err
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(discoveryClient))

	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}

	return mapping.Resource, nil
}

func GetGvr(kind string) (schema.GroupVersionResource, error) {
	discoveryClient, err := GetDiscoveryClient()
	if err != nil {
		return schema.GroupVersionResource{}, err
	}

	gvk := schema.GroupVersionResource{
		Group:    "",
		Version:  "",
		Resource: kind,
	}

	// Use the discovery client to get the mapping for the resource

	cachedDiscoveryClient := memory.NewMemCacheClient(discoveryClient)

	discoveryRestMapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscoveryClient)

	mapping, err := discoveryRestMapper.ResourcesFor(gvk)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}

	if len(mapping) == 0 {
		return schema.GroupVersionResource{}, fmt.Errorf("no mapping found for resource: %s", kind)
	}

	return mapping[0], nil
}
