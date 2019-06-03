package extension_test

import (
	"context"
	"encoding/base64"

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
	"k8s.io/client-go/kubernetes/scheme"

	cfakes "github.com/SUSE/eirinix/testing/fakes"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/cf-operator/pkg/credsgen"
	gfakes "code.cloudfoundry.org/cf-operator/pkg/credsgen/fakes"
	"code.cloudfoundry.org/cf-operator/pkg/kube/util/config"
	"code.cloudfoundry.org/cf-operator/testing"
)

var _ = Describe("Extension Manager", func() {
	c := catalog.NewCatalog()
	Manager := c.SimpleManager()
	eiriniManager, _ := Manager.(*DefaultExtensionManager)

	Context("Object creation", func() {
		manager := c.SimpleManager()
		It("Is an interface", func() {
			m, ok := manager.(*DefaultExtensionManager)
			Expect(ok).To(Equal(true))
			Expect(m.Options.Namespace).To(Equal("namespace"))
			Expect(m.Options.Host).To(Equal("127.0.0.1"))
			Expect(m.Options.Port).To(Equal(int32(90)))
		})
	})

	var (
		manager   *cfakes.FakeManager
		client    *cfakes.FakeClient
		ctx       context.Context
		config    *config.Config
		generator *gfakes.FakeGenerator
		env       testing.Catalog
	)

	BeforeEach(func() {
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

		config = env.DefaultConfig()
		ctx = testing.NewContext()

		eiriniManager.Context = ctx
		eiriniManager.Config = config
		eiriniManager.KubeManager = manager
		eiriniManager.Credsgen = generator
		eiriniManager.Options.Namespace = "default"
	})

	It("sets the operator namespace label", func() {
		client.UpdateCalls(func(_ context.Context, object runtime.Object) error {
			ns := object.(*unstructured.Unstructured)
			labels := ns.GetLabels()
			Expect(labels["eirini-extensions-ns"]).To(Equal(config.Namespace))

			return nil
		})
		err := eiriniManager.OperatorSetup()
		Expect(err).ToNot(HaveOccurred())

		err = eiriniManager.RegisterExtensions()
		Expect(err).ToNot(HaveOccurred())
	})

	Context("if there is no cert secret yet", func() {
		It("generates and persists the certificates on disk and in a secret", func() {
			Expect(afero.Exists(config.Fs, "/tmp/eirini-extensions-certs/key.pem")).To(BeFalse())
			err := eiriniManager.OperatorSetup()
			Expect(err).ToNot(HaveOccurred())

			err = eiriniManager.RegisterExtensions()
			Expect(err).ToNot(HaveOccurred())

			Expect(afero.Exists(config.Fs, "/tmp/eirini-extensions-certs/key.pem")).To(BeTrue())
			Expect(generator.GenerateCertificateCallCount()).To(Equal(2)) // Generate CA and certificate
			Expect(client.CreateCallCount()).To(Equal(2))                 // Persist secret and the webhook config
		})
	})

	Context("if there is a persisted cert secret already", func() {
		BeforeEach(func() {
			secret := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name":      "eirini-extensions-webhook-server-cert",
						"namespace": config.Namespace,
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
			Expect(client.CreateCallCount()).To(Equal(1)) // webhook config
		})

		It("generates the webhook configuration", func() {
			client.CreateCalls(func(context context.Context, object runtime.Object) error {
				config := object.(*admissionregistrationv1beta1.MutatingWebhookConfiguration)
				Expect(config.Name).To(Equal("eirini-extensions-mutating-hook-" + config.Namespace))
				Expect(len(config.Webhooks)).To(Equal(1))

				wh := config.Webhooks[0]
				Expect(wh.Name).To(Equal("0.eirinix.org"))
				Expect(*wh.ClientConfig.URL).To(Equal("https://foo.com:1234/0"))
				Expect(wh.ClientConfig.CABundle).To(ContainSubstring("the-ca-cert"))
				Expect(*wh.FailurePolicy).To(Equal(admissionregistrationv1beta1.Fail))
				return nil
			})
			err := eiriniManager.OperatorSetup()
			Expect(err).ToNot(HaveOccurred())

			eiriniManager.AddExtension(c.SimpleExtension())
			err = eiriniManager.RegisterExtensions()
			Expect(err).ToNot(HaveOccurred())
		})
	})

})
