package testing

import (
	"context"

	eirinix "github.com/SUSE/eirinix"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

type parentExtension struct {
	Name string
}

type testExtension struct {
	parentExtension
}

func (e *testExtension) Handle(context.Context, eirinix.Manager, *corev1.Pod, types.Request) types.Response {
	res := types.Response{Response: &v1beta1.AdmissionResponse{AuditAnnotations: map[string]string{"name": e.Name}}}
	return res
}
