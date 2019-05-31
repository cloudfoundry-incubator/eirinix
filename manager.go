package extension

import (
	"context"
	"strconv"
	"time"

	credsgen "code.cloudfoundry.org/cf-operator/pkg/credsgen/in_memory_generator"
	kubeConfig "code.cloudfoundry.org/cf-operator/pkg/kube/config"
	"code.cloudfoundry.org/cf-operator/pkg/kube/util/config"
	"github.com/SUSE/eirinix/util/ctxlog"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"go.uber.org/zap"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	machinerytypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"k8s.io/client-go/rest"
)

// DefaultExtensionManager represent an implementation of Manager
type DefaultExtensionManager struct {
	Extensions      []Extension
	Namespace, Host string
	Port            int32
	KubeConfig      string
	kubeConnection  *rest.Config
	kubeManager     manager.Manager
	Logger          *zap.SugaredLogger
	Context         context.Context
	Config          *config.Config
	WebHookConfig   *WebhookConfig
	WebHookServer   *webhook.Server
}

var addToSchemes = runtime.SchemeBuilder{}

// AddToScheme adds all Resources to the Scheme
func AddToScheme(s *runtime.Scheme) error {
	return addToSchemes.AddToScheme(s)
}

// NewManager returns a manager for the kubernetes cluster.
// the kubeconfig file and the logger are optional
func NewManager(namespace, host string, port int32, kubeConfigfile string, logger *zap.SugaredLogger) Manager {
	if logger == nil {
		logger = &zap.SugaredLogger{}
	}
	return &DefaultExtensionManager{Namespace: namespace, Host: host, Port: port, KubeConfig: kubeConfigfile, Logger: logger}
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
	// XXX: If kubeConfig Getter path is empty it will get from env
	restConfig, err := kubeConfig.NewGetter(m.Logger).Get(m.KubeConfig)
	if err != nil {
		return err
	}
	if err := kubeConfig.NewChecker(m.Logger).Check(restConfig); err != nil {
		return err
	}
	m.kubeConnection = restConfig

	return nil
}

func (m *DefaultExtensionManager) operatorSetup() error {

	err := setOperatorNamespaceLabel(m.Context, m.Config, m.kubeManager.GetClient())
	if err != nil {
		return errors.Wrap(err, "setting the operator namespace label")
	}

	err = m.WebHookConfig.setupCertificate(m.Context)
	if err != nil {
		return errors.Wrap(err, "setting up the webhook server certificate")
	}

	return nil
}

func setOperatorNamespaceLabel(ctx context.Context, config *config.Config, c client.Client) error {
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
	labels["eirini-extensions-ns"] = config.Namespace
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
				ID:        strconv.Itoa(k),
				Namespace: m.Namespace,
				// XXX: Rember, preferably it should be configurable
				FailurePolicy: admissionregistrationv1beta1.Fail,
				Manager:       m.kubeManager,
				WebHookServer: m.WebHookServer,
			})
		if err != nil {
			return err
		}
		webhooks = append(webhooks, admissionHook)
	}

	err := m.WebHookConfig.generateWebhookServerConfig(m.Context, webhooks)
	if err != nil {
		return errors.Wrap(err, "generating the webhook server configuration")
	}
	return nil
}

func (m *DefaultExtensionManager) setup() error {
	m.Context = ctxlog.NewManagerContext(m.Logger)

	m.Config = &config.Config{
		CtxTimeOut:        10 * time.Second,
		Namespace:         m.Namespace,
		WebhookServerHost: m.Host,
		Fs:                afero.NewOsFs(),
	}

	m.WebHookConfig = NewWebhookConfig(
		m.kubeManager.GetClient(),
		m.Config,
		credsgen.NewInMemoryGenerator(m.Logger),
		"eirini-extensions-mutating-hook-"+m.Namespace)

	kubeConn, err := m.KubeConnection()
	if err != nil {
		return errors.Wrap(err, "Failed connecting to kubernetes cluster")
	}

	disableConfigInstaller := true
	hookServer, err := webhook.NewServer("eirini-extensions", m.kubeManager, webhook.ServerOptions{
		Port:                          m.Config.WebhookServerPort,
		CertDir:                       m.WebHookConfig.CertDir,
		DisableWebhookConfigInstaller: &disableConfigInstaller,
		BootstrapOptions: &webhook.BootstrapOptions{
			MutatingWebhookConfigName: m.WebHookConfig.ConfigName,
			Host:                      &m.Config.WebhookServerHost,
			// The user should probably be able to use a service instead.
			// Service: ??
		},
	})
	if err != nil {
		return err
	}
	m.WebHookServer = hookServer

	mgr, err := manager.New(
		kubeConn,
		manager.Options{
			Namespace: m.Namespace,
		})
	if err != nil {
		return err
	}

	m.kubeManager = mgr

	return nil
}

// Start starts the WebHook server infinite loop, and returns an error on failure
func (m *DefaultExtensionManager) Start() error {
	defer m.Logger.Sync()

	if err := m.setup(); err != nil {
		return err
	}

	// Setup Scheme for all resources
	if err := AddToScheme(m.kubeManager.GetScheme()); err != nil {
		return err
	}

	if err := m.RegisterExtensions(); err != nil {
		return err
	}

	if err := m.operatorSetup(); err != nil {
		return err
	}

	return m.kubeManager.Start(signals.SetupSignalHandler())
}
