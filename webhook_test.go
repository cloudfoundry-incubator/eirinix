package extension_test

import (
	"context"

	. "github.com/SUSE/eirinix"
	catalog "github.com/SUSE/eirinix/testing"
	. "github.com/onsi/ginkgo"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	. "github.com/onsi/gomega"
)

var _ = Describe("Webhook implementation", func() {
	c := catalog.NewCatalog()
	m := c.SimpleManager()
	w := NewWebhook(c.SimpleExtension(), m)

	Context("With a fake extension", func() {
		It("It errors without a manager", func() {
			err := w.RegisterAdmissionWebHook(nil, WebhookOptions{ID: "volume", ManagerOptions: ManagerOptions{Namespace: "eirini", OperatorFingerprint: "eirini-x"}})
			Expect(err.Error()).To(Equal("No failure policy set"))
			failurePolicy := admissionregistrationv1beta1.Fail

			err = w.RegisterAdmissionWebHook(nil, WebhookOptions{ID: "volume", ManagerOptions: ManagerOptions{FailurePolicy: &failurePolicy, Namespace: "eirini", OperatorFingerprint: "eirini-x"}})
			Expect(err.Error()).To(Equal("The Mutating webhook needs a Webhook server to register to"))
		})

		It("Delegates to the Extension the handler", func() {
			ctx := context.Background()
			t := admission.Request{}
			res := w.Handle(ctx, t)
			annotations := res.AdmissionResponse.AuditAnnotations
			v, ok := annotations["name"]
			Expect(ok).To(Equal(true))
			Expect(v).To(Equal("test"))
		})

	})
})
