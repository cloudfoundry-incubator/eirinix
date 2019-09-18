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

	Context("With a fake extension", func() {
		c := catalog.NewCatalog()
		m := c.SimpleManager()
		w := NewWebhook(c.SimpleExtension(), m)
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

		It("It does generate correctly the webhook details", func() {
			err := w.RegisterAdmissionWebHook(nil, WebhookOptions{ID: "volume", ManagerOptions: ManagerOptions{Namespace: "eirini", OperatorFingerprint: "eirini-x"}})
			Expect(err.Error()).To(Equal("No failure policy set"))
			failurePolicy := admissionregistrationv1beta1.Fail

			err = w.RegisterAdmissionWebHook(nil, WebhookOptions{ID: "volume", ManagerOptions: ManagerOptions{FailurePolicy: &failurePolicy, Namespace: "eirini", OperatorFingerprint: "eirini-x"}})
			Expect(err.Error()).To(Equal("The Mutating webhook needs a Webhook server to register to"))

			Expect(w.GetFailurePolicy()).To(Equal(failurePolicy))
			register := false
			err = w.RegisterAdmissionWebHook(nil, WebhookOptions{ID: "volume", ManagerOptions: ManagerOptions{

				FailurePolicy:       &failurePolicy,
				RegisterWebHook:     &register,
				Namespace:           "eirini",
				OperatorFingerprint: "eirini-x"}})
			Expect(err).ToNot(HaveOccurred())
			mutatingWebHook, ok := w.(*DefaultMutatingWebhook)
			Expect(ok).To(BeTrue())
			Expect(mutatingWebHook.Path).To(Equal("/volume"))
			Expect(mutatingWebHook.Name).To(Equal("volume.eirini-x.org"))
			Expect(mutatingWebHook.NamespaceSelector.MatchLabels).To(Equal(map[string]string{"eirini-x-ns": "eirini"}))
			Expect(mutatingWebHook.Webhook.Handler).ToNot(BeNil())
			Expect(mutatingWebHook.Rules[0].Rule.APIGroups).To(Equal([]string{""}))
			Expect(mutatingWebHook.Rules[0].Rule.APIVersions).To(Equal([]string{"v1"}))
			Expect(mutatingWebHook.Rules[0].Rule.Resources).To(Equal([]string{"pods"}))
			Expect(mutatingWebHook.Rules[0].Operations).To(Equal(
				[]admissionregistrationv1beta1.OperationType{
					"CREATE",
					"UPDATE",
				}))
			Expect(*mutatingWebHook.Rules[0].Rule.Scope).To(Equal(admissionregistrationv1beta1.ScopeType("*")))

		})

	})
})
