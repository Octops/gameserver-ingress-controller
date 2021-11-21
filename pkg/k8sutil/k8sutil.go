package k8sutil

import (
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

// KubeConfigEnv (optionally) specify the location of kubeconfig file
const KubeConfigEnv = "KUBECONFIG"

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
	cfg.QPS = 100
	cfg.Burst = 100

	return cfg, nil
}
