package extension_test

import (
	"context"

	. "github.com/SUSE/eirinix/pkg/extension"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/admission/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

type ParentExtension struct {
	Name string
}

// FIXME: Move to catalog
type TestExtension struct {
	ParentExtension
}

func (e *TestExtension) Handle(context.Context, types.Request) types.Response {
	res := types.Response{Response: &v1beta1.AdmissionResponse{AuditAnnotations: map[string]string{"name": e.Name}}}
	return res
}

var _ = Describe("Extension Manager", func() {

	Context("Object creation", func() {
		manager := NewExtensionManager("namespace", "127.0.0.1", 90, nil)
		It("Is an interface", func() {
			m, ok := manager.(*DefaultExtensionManager)
			Expect(ok).To(Equal(true))
			Expect(m.Namespace).To(Equal("namespace"))
			Expect(m.Host).To(Equal("127.0.0.1"))
			Expect(m.Port).To(Equal(int32(90)))
		})

		It("Adds extensions", func() {
			manager.AddExtension(&TestExtension{
				ParentExtension{Name: "test"}})
			Expect(len(manager.ListExtensions())).To(Equal(1))
		})
	})

})
