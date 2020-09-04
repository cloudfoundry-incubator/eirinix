package testing

import (
	"context"
	"errors"
	"net/http"

	eirinix "code.cloudfoundry.org/eirinix"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type parentExtension struct {
	Name string
}

type testExtension struct {
	parentExtension
}

func (e *testExtension) Handle(context.Context, eirinix.Manager, *corev1.Pod, admission.Request) admission.Response {
	res := admission.Response{AdmissionResponse: v1beta1.AdmissionResponse{AuditAnnotations: map[string]string{"name": e.Name}}}
	return res
}

type EditEnvExtension struct{}

func (e *EditEnvExtension) Handle(ctx context.Context, eiriniManager eirinix.Manager, pod *corev1.Pod, req admission.Request) admission.Response {
	if pod == nil {
		return admission.Errored(http.StatusBadRequest, errors.New("No pod could be decoded from the request"))
	}
	podCopy := pod.DeepCopy()
	for i := range podCopy.Spec.Containers {
		c := &podCopy.Spec.Containers[i]
		for _, e := range c.Env {
			if e.Name == "STICKY_MESSAGE" {
				// was already patched
				return eiriniManager.PatchFromPod(req, podCopy)
			}
		}
		c.Env = append(c.Env, corev1.EnvVar{Name: "STICKY_MESSAGE", Value: "Eirinix is awesome!"})
	}
	return eiriniManager.PatchFromPod(req, podCopy)
}
