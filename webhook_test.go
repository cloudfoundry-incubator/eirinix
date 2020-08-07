package extension_test

import (
	"context"

	credsgen "code.cloudfoundry.org/quarks-utils/pkg/credsgen"
	gfakes "code.cloudfoundry.org/quarks-utils/pkg/credsgen/fakes"
	. "github.com/SUSE/eirinix"
	catalog "github.com/SUSE/eirinix/testing"
	cfakes "github.com/SUSE/eirinix/testing/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var _ = Describe("Webhook implementation", func() {
	var (
		manager                             *cfakes.FakeManager
		client                              *cfakes.FakeClient
		ctx                                 context.Context
		generator                           *gfakes.FakeGenerator
		eirinixcatalog                      catalog.Catalog
		ServiceManager, Manager             Manager
		eiriniServiceManager, eiriniManager *DefaultExtensionManager
		w                                   MutatingWebhook
	)

	BeforeEach(func() {
		eirinixcatalog = catalog.NewCatalog()
		ServiceManager = eirinixcatalog.SimpleManagerService()

		eiriniServiceManager, _ = ServiceManager.(*DefaultExtensionManager)
		Manager = eirinixcatalog.SimpleManager()
		eiriniManager, _ = Manager.(*DefaultExtensionManager)
		AddToScheme(scheme.Scheme)
		client = &cfakes.FakeClient{}
		restMapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{})
		restMapper.Add(schema.GroupVersionKind{Group: "", Kind: "Pod", Version: "v1"}, meta.RESTScopeNamespace)

		manager = &cfakes.FakeManager{}
		manager.GetSchemeReturns(scheme.Scheme)
		manager.GetClientReturns(client)
		manager.GetRESTMapperReturns(restMapper)
		manager.GetWebhookServerReturns(&webhook.Server{})

		generator = &gfakes.FakeGenerator{}
		generator.GenerateCertificateReturns(credsgen.Certificate{Certificate: []byte("thecert")}, nil)

		ctx = catalog.NewContext()

		eiriniManager.Context = ctx
		eiriniManager.KubeManager = manager
		eiriniManager.Options.Namespace = "eirini"
		eiriniManager.Credsgen = generator
		eiriniManager.GenWebHookServer()

		eiriniServiceManager.Context = ctx
		eiriniServiceManager.KubeManager = manager
		eiriniServiceManager.Options.Namespace = "eirini"
		eiriniServiceManager.Credsgen = generator
		w = NewWebhook(eirinixcatalog.SimpleExtension(), eiriniManager)

	})

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

		It("It does generate correctly the webhook details", func() {

			err := w.RegisterAdmissionWebHook(nil, WebhookOptions{ID: "volume", ManagerOptions: ManagerOptions{Namespace: "eirini", OperatorFingerprint: "eirini-x"}})
			Expect(err.Error()).To(Equal("No failure policy set"))
			failurePolicy := admissionregistrationv1beta1.Fail

			err = w.RegisterAdmissionWebHook(nil, WebhookOptions{ID: "volume", ManagerOptions: ManagerOptions{FailurePolicy: &failurePolicy, Namespace: "eirini", OperatorFingerprint: "eirini-x"}})
			Expect(err.Error()).To(Equal("The Mutating webhook needs a Webhook server to register to"))

			Expect(w.GetFailurePolicy()).To(Equal(failurePolicy))

			err = w.RegisterAdmissionWebHook(eiriniManager.WebhookServer, WebhookOptions{ID: "volume", ManagerOptions: ManagerOptions{
				FailurePolicy:       &failurePolicy,
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
