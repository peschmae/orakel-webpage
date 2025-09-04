package k8s

import (
	"context"
	"fmt"

	checksv1alpha1 "github.com/fhnw-imvs/fhnw-kubeseccontext/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func GetNamespaceHardeningChecks() ([]checksv1alpha1.NamespaceHardeningCheck, error) {
	client, err := GetDynamicClient()
	if err != nil {
		return nil, err
	}

	gvr, err := GetGvr("NamespaceHardeningCheck")
	if err != nil {
		return nil, err
	}

	unstructuredList, err := client.Resource(gvr).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var checks []checksv1alpha1.NamespaceHardeningCheck
	for _, item := range unstructuredList.Items {
		var check checksv1alpha1.NamespaceHardeningCheck
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(item.UnstructuredContent(), &check)
		if err != nil {
			return nil, fmt.Errorf("failed to convert unstructured to NamespaceHardeningCheck: %w", err)
		}
		checks = append(checks, check)
	}

	return checks, nil
}

func GetWorkloadHardeningChecks() ([]checksv1alpha1.WorkloadHardeningCheck, error) {
	client, err := GetDynamicClient()
	if err != nil {
		return nil, err
	}

	gvr, err := GetGvr("WorkloadHardeningCheck")
	if err != nil {
		return nil, err
	}

	unstructuredList, err := client.Resource(gvr).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var checks []checksv1alpha1.WorkloadHardeningCheck
	for _, item := range unstructuredList.Items {
		var check checksv1alpha1.WorkloadHardeningCheck
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(item.UnstructuredContent(), &check)
		if err != nil {
			return nil, fmt.Errorf("failed to convert unstructured to NamespaceHardeningCheck: %w", err)
		}
		checks = append(checks, check)
	}

	return checks, nil
}
