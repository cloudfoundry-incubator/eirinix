package testing

import (
	"context"

	"k8s.io/api/admission/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

type ParentExtension struct {
	Name string
}

type TestExtension struct {
	ParentExtension
}

func (c *Catalog) SimpleExtension() *TestExtension {

	return &TestExtension{
		ParentExtension{Name: "test"}}
}
func (e *TestExtension) Handle(context.Context, types.Request) types.Response {
	res := types.Response{Response: &v1beta1.AdmissionResponse{AuditAnnotations: map[string]string{"name": e.Name}}}
	return res
}
