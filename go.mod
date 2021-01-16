module github.com/Octops/gameserver-ingress-controller

go 1.14

require (
	agones.dev/agones v1.11.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.7.1
	k8s.io/api v0.17.14
	k8s.io/apimachinery v0.17.14
	k8s.io/client-go v0.17.14
	sigs.k8s.io/controller-runtime v0.5.0
)
