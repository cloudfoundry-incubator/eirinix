package extension_test

import (
	"context"

	. "github.com/SUSE/eirinix"
	catalog "github.com/SUSE/eirinix/testing"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

var _ = Describe("WebHook implementation", func() {

	c := catalog.NewCatalog()
	w := NewWebHook(c.SimpleExtension())

	Context("With a fake extension", func() {
		It("It errors without a manager", func() {
			_, err := w.RegisterAdmissionWebHook(WebHookOptions{ID: "volume", Namespace: "eirini"})
			Expect(err).To(Not(BeNil()))
		})

		It("Delegates to the Extension the handler", func() {
			ctx := context.Background()
			t := types.Request{}
			res := w.Handle(ctx, t)
			annotations := res.Response.AuditAnnotations
			v, ok := annotations["name"]
			Expect(ok).To(Equal(true))
			Expect(v).To(Equal("test"))
		})
	})
})
