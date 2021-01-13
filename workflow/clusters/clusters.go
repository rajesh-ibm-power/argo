package clusters

import (
	"context"
	"encoding/json"
	"fmt"

	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo/config/clusters"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func GetConfigs(ctx context.Context, restConfig *rest.Config, kubeclientset kubernetes.Interface, clusterName wfv1.ClusterName, namespace, managedNamespace string) (map[wfv1.ClusterNamespaceKey]*rest.Config, map[wfv1.ClusterNamespaceKey]kubernetes.Interface, map[wfv1.ClusterNamespaceKey]dynamic.Interface, error) {
	clusterNamespace := wfv1.NewClusterNamespaceKey(clusterName, managedNamespace)
	restConfigs := map[wfv1.ClusterNamespaceKey]*rest.Config{}
	if restConfig != nil {
		restConfigs[clusterNamespace] = restConfig
	}
	kubernetesInterfaces := map[wfv1.ClusterNamespaceKey]kubernetes.Interface{clusterNamespace: kubeclientset}
	dynamicInterfaces := map[wfv1.ClusterNamespaceKey]dynamic.Interface{}
	if restConfig != nil {
		dynamicInterface, err := dynamic.NewForConfig(restConfig)
		if err != nil {
			return nil, nil, nil, err
		}
		dynamicInterfaces[clusterNamespace] = dynamicInterface
	}
	secret, err := kubeclientset.CoreV1().Secrets(namespace).Get(ctx, "rest-config", metav1.GetOptions{})
	if apierr.IsNotFound(err) {
	} else if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get secret/clusters: %w", err)
	} else {
		for key, data := range secret.Data {
			clusterNamespace, err := wfv1.ParseClusterNamespaceKey(key)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("failed parse key %s: %w", key, err)
			}
			c := &clusters.Config{}
			err = json.Unmarshal(data, c)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("failed unmarshall JSON for cluster %s: %w", key, err)
			}
			restConfigs[clusterNamespace] = c.RestConfig()
			clientset, err := kubernetes.NewForConfig(restConfigs[clusterNamespace])
			if err != nil {
				return nil, nil, nil, fmt.Errorf("failed create new kube client for cluster %s: %w", key, err)
			}
			dy, err := dynamic.NewForConfig(restConfigs[clusterNamespace])
			if err != nil {
				return nil, nil, nil, fmt.Errorf("failed create new dynamic client for cluster %s: %w", key, err)
			}
			kubernetesInterfaces[clusterNamespace] = clientset
			dynamicInterfaces[clusterNamespace] = dy
		}
	}
	return restConfigs, kubernetesInterfaces, dynamicInterfaces, nil
}