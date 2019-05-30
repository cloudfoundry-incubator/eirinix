package extension

import (
	"context"
	"strconv"
	"time"

	credsgen "code.cloudfoundry.org/cf-operator/pkg/credsgen/in_memory_generator"
	kubeConfig "code.cloudfoundry.org/cf-operator/pkg/kube/config"
	"code.cloudfoundry.org/cf-operator/pkg/kube/util/config"
	"github.com/SUSE/eirinix/pkg/util/ctxlog"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"go.uber.org/zap"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	machinerytypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"k8s.io/client-go/rest"
)

type DefaultExtensionManager struct {
	Extensions      []Extension
	Namespace, Host string
	Port            int32
	KubeConfig      string
	KubeConnection  *rest.Config
	kubeManager     manager.Manager
	Logger          *zap.SugaredLogger
	Context         context.Context
	Config          *config.Config
	WebHookConfig   *WebhookConfig
}

// XXX: Kubernetes runtime code
var addToSchemes = runtime.SchemeBuilder{}

// AddToScheme adds all Resources to the Scheme
func AddToScheme(s *runtime.Scheme) error {
	return addToSchemes.AddToScheme(s)
}

func NewExtensionManager(namespace, host string, port int32, logger *zap.SugaredLogger) ExtensionManager {
	if logger == nil {
		logger = &zap.SugaredLogger{}
	}
	return &DefaultExtensionManager{Namespace: namespace, Host: host, Port: port, Logger: logger}
}

func (m *DefaultExtensionManager) AddExtension(e Extension) {
	m.Extensions = append(m.Extensions, e)
}

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
	m.KubeConnection = restConfig

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

func (m *DefaultExtensionManager) Kube() (*rest.Config, error) {
	if m.KubeConnection == nil {

		if err := m.kubeSetup(); err != nil {
			return nil, err
		}
	}

	return m.KubeConnection, nil
}

func (m *DefaultExtensionManager) RegisterExtensions() error {
	webhooks := []*admission.Webhook{}
	for k, e := range m.Extensions {
		w := NewWebHook(e.Handle)
		// TODO: Fill all the options
		admissionHook, err := w.RegisterAdmissionWebHook(WebHookOptions{Id: strconv.Itoa(k)})
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

func (m *DefaultExtensionManager) Start() error {
	defer m.Logger.Sync()
	m.Context = ctxlog.NewManagerContext(m.Logger)
	m.Config = &config.Config{
		CtxTimeOut:        10 * time.Second,
		Namespace:         m.Namespace,
		WebhookServerHost: m.Host,
		Fs:                afero.NewOsFs(),
	}

	m.WebHookConfig = NewWebhookConfig(m.kubeManager.GetClient(), m.Config, credsgen.NewInMemoryGenerator(m.Logger), "eirini-extensions-mutating-hook-"+m.Namespace)
	kubeConn, err := m.Kube()
	if err != nil {
		return errors.Wrap(err, "Failed connecting to kubernetes cluster")
	}
	mgr, err := manager.New(
		kubeConn,
		manager.Options{
			Namespace: m.Namespace,
		})
	if err != nil {
		return err
	}

	m.kubeManager = mgr

	// Setup Scheme for all resources
	if err = AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	err = m.RegisterExtensions()
	if err != nil {
		return err
	}
	err = m.operatorSetup()
	if err != nil {
		return err
	}

	return mgr.Start(signals.SetupSignalHandler())
}
