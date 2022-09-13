package reconcilers

import (
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"github.com/Octops/gameserver-ingress-controller/pkg/gameserver"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"strings"
	"text/template"
)

type ServiceOption func(gs *agonesv1.GameServer, service *corev1.Service) error

func WithCustomServiceAnnotationsTemplate() ServiceOption {
	return func(gs *agonesv1.GameServer, service *corev1.Service) error {
		data := struct {
			Name string
			Port int32
		}{
			Name: gs.Name,
			Port: gameserver.GetGameServerPort(gs).Port,
		}

		annotations := service.Annotations
		for k, v := range gs.Annotations {
			if strings.HasPrefix(k, gameserver.OctopsAnnotationCustomServicePrefix) {
				custom := strings.TrimPrefix(k, gameserver.OctopsAnnotationCustomServicePrefix)
				if len(custom) == 0 {
					return errors.Errorf("custom annotation %s does not contain a suffix", k)
				}

				if !strings.Contains(v, "{{") || !strings.Contains(v, "}}") {
					continue
				}

				t, err := template.New("gs").Parse(v)
				if err != nil {
					return errors.Errorf("%s:%s does not contain a valid template", custom, v)
				}

				b := new(strings.Builder)
				err = t.Execute(b, data)
				if parsed := b.String(); len(parsed) > 0 {
					annotations[custom] = parsed
				}
			}
		}

		service.SetAnnotations(annotations)

		return nil
	}
}

func WithCustomServiceAnnotations() ServiceOption {
	return func(gs *agonesv1.GameServer, service *corev1.Service) error {
		annotations := service.Annotations
		for k, v := range gs.Annotations {
			if strings.HasPrefix(k, gameserver.OctopsAnnotationCustomServicePrefix) {
				custom := strings.TrimPrefix(k, gameserver.OctopsAnnotationCustomServicePrefix)
				if len(custom) == 0 {
					return errors.Errorf("custom annotation %s does not contain a suffix", k)
				}
				annotations[custom] = v
			}
		}

		service.SetAnnotations(annotations)

		return nil
	}
}
