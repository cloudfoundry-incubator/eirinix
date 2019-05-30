package extension_test

import (
	"context"

	. "github.com/SUSE/eirinix/pkg/extension"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

// FIXME: Move to catalog
type TestExtension struct{}

func (e *TestExtension) Handle(context.Context, types.Request) types.Response { return types.Response{} }

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
			manager.AddExtension(&TestExtension{})
			Expect(len(manager.ListExtensions())).To(Equal(1))
		})
	})

})
