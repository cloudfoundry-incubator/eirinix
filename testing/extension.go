package testing

import (
	"context"

	eirinix "github.com/SUSE/eirinix"
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
