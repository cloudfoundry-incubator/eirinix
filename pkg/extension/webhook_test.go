package extension_test

import (
	. "github.com/SUSE/eirinix/pkg/extension"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// type WebHookOptions struct {
// 	Id          string // Webhook path will be generated out of that
// 	MatchLabels map[string]string
// 	// XXX: Rember it needs to be configurable
// 	FailurePolicy admissionregistrationv1beta1.FailurePolicyType
// 	Namespace     string
// 	Manager       manager.Manager
// 	WebHookServer *webhook.Server
// }

var _ = Describe("WebHook implementation", func() {

	Context("Object creation", func() {
		e := &TestExtension{}
		w := NewWebHook(e.Handle)
		It("Is an interface", func() {
			_, err := w.RegisterAdmissionWebHook(WebHookOptions{Id: "volume", Namespace: "eirini"})
			Expect(err).To(Not(BeNil()))

			//	a, err = w.RegisterAdmissionWebHook(WebHookOptions{Id: "volume", Namespace: "eirini", Manager: &manager.Manager{}})
			//s		Expect(err).To(BeNil())
		})
	})
})
