package extension

import (
	"context"
	"fmt"
	"strconv"
	"time"

	credsgen "code.cloudfoundry.org/cf-operator/pkg/credsgen"
	inmemorycredgen "code.cloudfoundry.org/cf-operator/pkg/credsgen/in_memory_generator"
	"go.uber.org/zap"

	kubeConfig "code.cloudfoundry.org/cf-operator/pkg/kube/config"
	"code.cloudfoundry.org/cf-operator/pkg/kube/util/config"
	"github.com/SUSE/eirinix/util/ctxlog"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	machinerytypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// DefaultExtensionManager represent an implementation of Manager
type DefaultExtensionManager struct {
	Extensions     []Extension
	kubeConnection *rest.Config
	KubeManager    manager.Manager
	Logger         *zap.SugaredLogger
	Context        context.Context
	Config         *config.Config
	WebHookConfig  *WebhookConfig
	WebHookServer  *webhook.Server
	Credsgen       credsgen.Generator
	Options        ManagerOptions
}

type ManagerOptions struct {
	Namespace, Host     string
	Port                int32
	KubeConfig          string
	Logger              *zap.SugaredLogger
	FailurePolicy       *admissionregistrationv1beta1.FailurePolicyType
	FilterEiriniApps    bool
	OperatorFingerprint string
}

var addToSchemes = runtime.SchemeBuilder{}

// AddToScheme adds all Resources to the Scheme
func AddToScheme(s *runtime.Scheme) error {
	return addToSchemes.AddToScheme(s)
}

// NewManager returns a manager for the kubernetes cluster.
// the kubeconfig file and the logger are optional
func NewManager(opts ManagerOptions) Manager {

	if opts.Logger == nil {
		z, e := zap.NewProduction()
		if e != nil {
			panic(errors.New("Cannot create logger"))
		}
		defer z.Sync() // flushes buffer, if any
		sugar := z.Sugar()
		opts.Logger = sugar
	}

	if opts.FailurePolicy == nil {
		failurePolicy := admissionregistrationv1beta1.Fail
		opts.FailurePolicy = &failurePolicy
	}

	opts.FilterEiriniApps = true
	opts.OperatorFingerprint = "eirini-x"
	return &DefaultExtensionManager{Options: opts}
}

// AddExtension adds an Erini extension to the manager
func (m *DefaultExtensionManager) AddExtension(e Extension) {
	m.Extensions = append(m.Extensions, e)
}

// ListExtensions returns the list of the Extensions added to the Manager
func (m *DefaultExtensionManager) ListExtensions() []Extension {
	return m.Extensions
}

func (m *DefaultExtensionManager) kubeSetup() error {
	restConfig, err := kubeConfig.NewGetter(m.Logger).Get(m.Options.KubeConfig)
	if err != nil {
		return err
	}
	if err := kubeConfig.NewChecker(m.Logger).Check(restConfig); err != nil {
		return err
	}
	m.kubeConnection = restConfig

	return nil
}

func (m *DefaultExtensionManager) OperatorSetup() error {
	disableConfigInstaller := true
	m.Context = ctxlog.NewManagerContext(m.Logger)
	m.WebHookConfig = NewWebhookConfig(
		m.KubeManager.GetClient(),
		m.Config,
		m.Credsgen,
		fmt.Sprintf("%s-mutating-hook-%s", m.Options.OperatorFingerprint, m.Options.Namespace))

	hookServer, err := webhook.NewServer(m.Options.OperatorFingerprint, m.KubeManager, webhook.ServerOptions{
		Port:                          m.Config.WebhookServerPort,
		CertDir:                       m.WebHookConfig.CertDir,
		DisableWebhookConfigInstaller: &disableConfigInstaller,
		BootstrapOptions: &webhook.BootstrapOptions{
			MutatingWebhookConfigName: m.WebHookConfig.ConfigName,
			Host:                      &m.Config.WebhookServerHost},
	})
	if err != nil {
		return err
	}
	m.WebHookServer = hookServer

	if err := m.setOperatorNamespaceLabel(m.Context, m.Config, m.KubeManager.GetClient()); err != nil {
		return errors.Wrap(err, "setting the operator namespace label")
	}

	err = m.WebHookConfig.setupCertificate(m.Context)
	if err != nil {
		return errors.Wrap(err, "setting up the webhook server certificate")
	}
	return nil
}

func (m *DefaultExtensionManager) setOperatorNamespaceLabel(ctx context.Context, config *config.Config, c client.Client) error {
	ns := &unstructured.Unstructured{}
	ns.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "",
		Kind:    "Namespace",
		Version: "v1",
	})
	err := c.Get(ctx, machinerytypes.NamespacedName{Name: config.Namespace}, ns)

	if err != nil {
		return errors.Wrap(err, "getting the namespace object")
	}

	labels := ns.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels[m.Options.getDefaultNamespaceLabel()] = config.Namespace
	ns.SetLabels(labels)
	err = c.Update(ctx, ns)

	if err != nil {
		return errors.Wrap(err, "updating the namespace object")
	}

	return nil
}

// KubeConnection sets up a connection to a Kubernetes cluster if not existing.
func (m *DefaultExtensionManager) KubeConnection() (*rest.Config, error) {
	if m.kubeConnection == nil {
		if err := m.kubeSetup(); err != nil {
			return nil, err
		}
	}
	return m.kubeConnection, nil
}

// RegisterExtensions it generates and register webhooks from the Extensions loaded in the Manager
func (m *DefaultExtensionManager) RegisterExtensions() error {
	webhooks := []*admission.Webhook{}
	for k, e := range m.Extensions {
		w := NewWebHook(e)
		admissionHook, err := w.RegisterAdmissionWebHook(
			WebHookOptions{
				ID:             strconv.Itoa(k),
				Manager:        m.KubeManager,
				WebHookServer:  m.WebHookServer,
				ManagerOptions: m.Options,
			})
		if err != nil {
			return err
		}
		webhooks = append(webhooks, admissionHook)
	}

	if err := m.WebHookConfig.generateWebhookServerConfig(m.Context, webhooks); err != nil {
		return errors.Wrap(err, "generating the webhook server configuration")
	}
	return nil
}

func (m *DefaultExtensionManager) Setup() error {
	m.Credsgen = inmemorycredgen.NewInMemoryGenerator(m.Logger)
	m.Config = &config.Config{
		CtxTimeOut:        10 * time.Second,
		Namespace:         m.Options.Namespace,
		WebhookServerHost: m.Options.Host,
		Fs:                afero.NewOsFs(),
	}

	kubeConn, err := m.KubeConnection()
	if err != nil {
		return errors.Wrap(err, "Failed connecting to kubernetes cluster")
	}

	mgr, err := manager.New(
		kubeConn,
		manager.Options{
			Namespace: m.Options.Namespace,
		})
	if err != nil {
		return err
	}

	m.KubeManager = mgr

	if err := m.OperatorSetup(); err != nil {
		return err
	}

	return nil
}

// Start starts the WebHook server infinite loop, and returns an error on failure
func (m *DefaultExtensionManager) Start() error {
	defer m.Logger.Sync()

	if err := m.Setup(); err != nil {
		return err
	}

	// Setup Scheme for all resources
	if err := AddToScheme(m.KubeManager.GetScheme()); err != nil {
		return err
	}

	if err := m.RegisterExtensions(); err != nil {
		return err
	}

	return m.KubeManager.Start(signals.SetupSignalHandler())
}

func (o *ManagerOptions) getDefaultNamespaceLabel() string {
	return fmt.Sprintf("%s-ns", o.OperatorFingerprint)
}
