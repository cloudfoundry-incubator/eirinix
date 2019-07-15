package extension_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	. "github.com/SUSE/eirinix"
	catalog "github.com/SUSE/eirinix/testing"
	. "github.com/onsi/ginkgo"
	"github.com/spf13/afero"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"

	cfakes "github.com/SUSE/eirinix/testing/fakes"
	. "github.com/onsi/gomega"

	credsgen "code.cloudfoundry.org/cf-operator/pkg/credsgen"
	gfakes "code.cloudfoundry.org/cf-operator/pkg/credsgen/fakes"
	"code.cloudfoundry.org/cf-operator/testing"
)

var _ = Describe("Extension Manager", func() {

	var (
		manager        *cfakes.FakeManager
		client         *cfakes.FakeClient
		ctx            context.Context
		generator      *gfakes.FakeGenerator
		eirinixcatalog catalog.Catalog
		Manager        Manager
		eiriniManager  *DefaultExtensionManager
	)

	BeforeEach(func() {
		eirinixcatalog = catalog.NewCatalog()
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

		generator = &gfakes.FakeGenerator{}
		generator.GenerateCertificateReturns(credsgen.Certificate{Certificate: []byte("thecert")}, nil)

		ctx = testing.NewContext()

		eiriniManager.Context = ctx
		eiriniManager.KubeManager = manager
		eiriniManager.Options.Namespace = "default"
		eiriniManager.Credsgen = generator
	})

	Context("DefaultExtensionManager", func() {
		It("satisfies the Manager interface", func() {
			m, ok := Manager.(*DefaultExtensionManager)
			Expect(ok).To(Equal(true))
			Expect(m.Options.Namespace).To(Equal("default"))
			Expect(m.Options.Host).To(Equal("127.0.0.1"))
			Expect(m.Options.Port).To(Equal(int32(90)))
			defaultPolicy := admissionregistrationv1beta1.Fail
			Expect(m.Options.FailurePolicy).To(Equal(&defaultPolicy))
			Expect(m.Options.OperatorFingerprint).To(Equal("eirini-x"))
			Expect(m.Options.KubeConfig).To(Equal(""))
			Expect(m.Options.Logger).NotTo(Equal(nil))
			Expect(m.Logger).NotTo(Equal(nil))
			Expect(*m.Options.FilterEiriniApps).To(BeTrue())

			Expect(Manager.GetLogger()).ToNot(BeNil())
			Expect(Manager.ListExtensions()).To(BeEmpty())
		})
		It("Setups correctly the operator structures", func() {
			err := eiriniManager.OperatorSetup()
			Expect(err).ToNot(HaveOccurred())
			Expect(eiriniManager.WebhookServer.Port).To(Equal(eiriniManager.Options.Port))
			Expect(eiriniManager.WebhookServer.Host).To(Equal(&eiriniManager.Options.Host))
		})

		It("called from the interface fails to start with no kube connection", func() {
			_, err := Manager.GetKubeConnection()
			Expect(err).ToNot(BeNil())
			err = Manager.Start()
			Expect(err).ToNot(BeNil())
		})
	})

	Context("if there is no cert secret yet", func() {
		It("generates and persists the certificates on disk and in a secret", func() {
			Expect(eiriniManager.Options.SetupCertificateName).To(Equal("eirini-x-setupcertificate"))
			eiriniManager.Options.SetupCertificateName = "test-setupcert"

			os.RemoveAll(fmt.Sprintf("/tmp/%s", eiriniManager.Options.SetupCertificateName))
			defer os.RemoveAll(fmt.Sprintf("/tmp/%s", eiriniManager.Options.SetupCertificateName))

			Expect(eiriniManager.WebhookServer).To(BeNil())
			Expect(afero.Exists(afero.NewOsFs(), fmt.Sprintf("/tmp/%s/key.pem", eiriniManager.Options.SetupCertificateName))).To(BeFalse())

			err := eiriniManager.OperatorSetup()
			Expect(err).ToNot(HaveOccurred())

			err = eiriniManager.RegisterExtensions()
			Expect(err).ToNot(HaveOccurred())

			Expect(eiriniManager.WebhookServer.CertDir).To(Equal(fmt.Sprintf("/tmp/%s", eiriniManager.Options.SetupCertificateName)))
			Expect(eiriniManager.WebhookServer.BootstrapOptions.MutatingWebhookConfigName).To(Equal("eirini-x-mutating-hook-default"))
			Expect(eiriniManager.WebhookConfig.CertDir).To(Equal(eiriniManager.WebhookServer.CertDir))
			Expect(eiriniManager.WebhookConfig.ConfigName).To(Equal(eiriniManager.WebhookServer.BootstrapOptions.MutatingWebhookConfigName))

			Expect(afero.Exists(afero.NewOsFs(), fmt.Sprintf("/tmp/%s/key.pem", eiriniManager.Options.SetupCertificateName))).To(BeTrue())
			Expect(generator.GenerateCertificateCallCount()).To(Equal(2)) // Generate CA and certificate
			Expect(client.CreateCallCount()).To(Equal(2))                 // Persist secret and the webhook config
		})
	})

	It("sets the operator namespace label", func() {
		client.UpdateCalls(func(_ context.Context, object runtime.Object) error {
			ns := object.(*unstructured.Unstructured)
			labels := ns.GetLabels()
			Expect(labels["eirini-x-ns"]).To(Equal(eiriniManager.Options.Namespace))

			return nil
		})
		err := eiriniManager.OperatorSetup()
		Expect(err).ToNot(HaveOccurred())

		err = eiriniManager.RegisterExtensions()
		Expect(err).ToNot(HaveOccurred())

	})

	Context("if there is a persisted cert secret already", func() {
		BeforeEach(func() {
			secret := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name":      "eirinix",
						"namespace": eiriniManager.Options.Namespace,
					},
					"data": map[string]interface{}{
						"certificate":    base64.StdEncoding.EncodeToString([]byte("the-cert")),
						"private_key":    base64.StdEncoding.EncodeToString([]byte("the-key")),
						"ca_certificate": base64.StdEncoding.EncodeToString([]byte("the-ca-cert")),
						"ca_private_key": base64.StdEncoding.EncodeToString([]byte("the-ca-key")),
					},
				},
			}
			client.GetCalls(func(context context.Context, nn types.NamespacedName, object runtime.Object) error {
				switch object.(type) {
				case *unstructured.Unstructured:
					secret.DeepCopyInto(object.(*unstructured.Unstructured))
					return nil
				}
				return apierrors.NewNotFound(schema.GroupResource{}, nn.Name)
			})

		})

		It("does not overwrite the existing secret", func() {
			err := eiriniManager.OperatorSetup()
			Expect(err).ToNot(HaveOccurred())
			err = eiriniManager.RegisterExtensions()
			Expect(err).ToNot(HaveOccurred())
			Expect(client.CreateCallCount()).To(Equal(1))                 // webhook config
			Expect(generator.GenerateCertificateCallCount()).To(Equal(0)) // Generate CA and certificate
		})

		It("generates the webhook configuration", func() {
			client.CreateCalls(func(context context.Context, object runtime.Object) error {
				config := object.(*admissionregistrationv1beta1.MutatingWebhookConfiguration)
				Expect(config.Name).To(Equal("eirini-x-mutating-hook-" + config.Namespace))
				Expect(len(config.Webhooks)).To(Equal(1))

				wh := config.Webhooks[0]
				Expect(wh.Name).To(Equal("0.eirini-x.org"))
				Expect(*wh.ClientConfig.URL).To(Equal(fmt.Sprintf("https://%s:%d/0", eiriniManager.Options.Host, eiriniManager.Options.Port)))
				Expect(wh.ClientConfig.CABundle).To(ContainSubstring("the-ca-cert"))
				Expect(*wh.FailurePolicy).To(Equal(admissionregistrationv1beta1.Fail))
				return nil
			})
			err := eiriniManager.OperatorSetup()
			Expect(err).ToNot(HaveOccurred())

			eiriniManager.AddExtension(eirinixcatalog.SimpleExtension())
			err = eiriniManager.RegisterExtensions()
			Expect(err).ToNot(HaveOccurred())

			Expect(Manager.ListExtensions()).ToNot(BeEmpty())
		})
	})

	Context("Watchers", func() {
		w := eirinixcatalog.SimpleWatcher()
		BeforeEach(func() {
			w = eirinixcatalog.SimpleWatcher()
		})
		It("Registers new watchers correctly", func() {
			eiriniManager.AddWatcher(w)
			Expect(len(eiriniManager.ListWatchers())).To(Equal(1))
			eiriniManager.AddWatcher(w)
			Expect(len(eiriniManager.ListWatchers())).To(Equal(2))
		})

		It("Handles events correctly", func() {
			eiriniManager.AddWatcher(w)
			eiriniManager.HandleEvent(watch.Event{Type: watch.EventType("test")})
			sw, ok := w.(*catalog.SimpleWatch)
			Expect(ok).To(Equal(true))

			Expect(len(sw.Handled)).To(Equal(1))
			Expect(string(sw.Handled[0].Type)).To(Equal("test"))
		})

	})
})
