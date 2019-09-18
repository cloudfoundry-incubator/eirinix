package extension_test

import (
	. "github.com/SUSE/eirinix"
	catalog "github.com/SUSE/eirinix/testing"
	cfakes "github.com/SUSE/eirinix/testing/fakes"
	. "github.com/onsi/ginkgo"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"

	credsgen "code.cloudfoundry.org/cf-operator/pkg/credsgen"
	gfakes "code.cloudfoundry.org/cf-operator/pkg/credsgen/fakes"
	"code.cloudfoundry.org/cf-operator/testing"
	"context"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	. "github.com/onsi/gomega"
)

var _ = Describe("Webhook configuration implementation", func() {

	var (
		manager                             *cfakes.FakeManager
		client                              *cfakes.FakeClient
		ctx                                 context.Context
		generator                           *gfakes.FakeGenerator
		eirinixcatalog                      catalog.Catalog
		ServiceManager, Manager             Manager
		eiriniServiceManager, eiriniManager *DefaultExtensionManager
	)
	failurePolicy := admissionregistrationv1beta1.Fail

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

		ctx = testing.NewContext()

		eiriniManager.Context = ctx
		eiriniManager.KubeManager = manager
		eiriniManager.Options.Namespace = "eirini"
		eiriniManager.Credsgen = generator

		eiriniServiceManager.Context = ctx
		eiriniServiceManager.KubeManager = manager
		eiriniServiceManager.Options.Namespace = "eirini"
		eiriniServiceManager.Credsgen = generator
	})

	Context("With a fake extension with a Host specified", func() {
		It("generates correctly services metadata", func() {
			w := NewWebhook(eirinixcatalog.SimpleExtension(), eiriniManager)
			register := false
			err := w.RegisterAdmissionWebHook(nil, WebhookOptions{ID: "volume", ManagerOptions: ManagerOptions{

				FailurePolicy:       &failurePolicy,
				RegisterWebHook:     &register,
				Namespace:           "eirini",
				OperatorFingerprint: "eirini-x"}})
			Expect(err).ToNot(HaveOccurred())
			eiriniManager.GenWebHookServer()
			admissions := eiriniManager.WebhookConfig.GenerateAdmissionWebhook([]MutatingWebhook{w})
			Expect(len(admissions)).To(Equal(1))
			url := "https://127.0.0.1:90/volume"
			Expect(admissions[0].ClientConfig.URL).To(Equal(&url))
		})
	})

	Context("With a fake extension with a Service", func() {
		It("generates correctly services metadata", func() {
			w := NewWebhook(eirinixcatalog.SimpleExtension(), eiriniServiceManager)
			register := false
			err := w.RegisterAdmissionWebHook(nil, WebhookOptions{ID: "volume", ManagerOptions: ManagerOptions{

				FailurePolicy:       &failurePolicy,
				RegisterWebHook:     &register,
				Namespace:           "eirini",
				OperatorFingerprint: "eirini-x"}})
			Expect(err).ToNot(HaveOccurred())
			eiriniServiceManager.GenWebHookServer()
			admissions := eiriniServiceManager.WebhookConfig.GenerateAdmissionWebhook([]MutatingWebhook{w})
			Expect(len(admissions)).To(Equal(1))
			url := "/volume"

			Expect(admissions[0].ClientConfig.Service.Name).To(Equal("extension"))
			Expect(admissions[0].ClientConfig.Service.Namespace).To(Equal("cf"))
			Expect(admissions[0].ClientConfig.Service.Path).To(Equal(&url))
		})
	})
})
