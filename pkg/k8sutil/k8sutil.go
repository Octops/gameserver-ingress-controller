package k8sutil

import (
	"fmt"
	"github.com/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

// KubeConfigEnv (optionally) specify the location of kubeconfig file
const KubeConfigEnv = "KUBECONFIG"

func NewClientSet(kubeconfig string) (kubernetes.Interface, error) {
	clientConf, err := NewClusterConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(clientConf)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create kubernetes client")
	}

	return client, nil
}

func NewClusterConfig(kubeconfig string) (*rest.Config, error) {
	var cfg *rest.Config
	var err error

	if len(kubeconfig) == 0 {
		kubeconfig = os.Getenv(KubeConfigEnv)
	}

	cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	// Increase QPS and Burst - https://github.com/prometheus-operator/prometheus-operator/blob/main/pkg/k8sutil/k8sutil.go#L96-L97
	cfg.QPS = 1000
	cfg.Burst = 1000

	return cfg, nil
}

func Namespaced(obj v1.Object) string {
	return fmt.Sprintf("%s/%s", obj.GetNamespace(), obj.GetName())
}
