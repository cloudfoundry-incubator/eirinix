package extension

import (
	"time"

	"code.cloudfoundry.org/cf-operator/pkg/kube/util/config"
	"github.com/SUSE/eirinix/pkg/operator"
	"github.com/SUSE/eirinix/pkg/util/ctxlog"

	kubeConfig "code.cloudfoundry.org/cf-operator/pkg/kube/config"
	"github.com/spf13/afero"
	"go.uber.org/zap"

	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

// NewExtensionManager returns the default ExtensionManager
func NewExtensionManager(namespace, host string, port int32) ExtensionManager {
	return &DefaultExtensionManager{Namespace: namespace, Host: host, Port: port}
}

func (m *DefaultExtensionManager) AddExtension(e Extension) {
	m.Extensions = append(m.Extensions, e)
}

func (m *DefaultExtensionManager) ListExtensions() []Extension {
	return m.Extensions
}

func (m *DefaultExtensionManager) Start(log *zap.SugaredLogger) {
	defer log.Sync()

	// XXX: If kubeConfig Getter path is empty it will get from env
	restConfig, err := kubeConfig.NewGetter(log).Get(m.KubeConfig)
	if err != nil {
		log.Fatal(err)
	}
	if err := kubeConfig.NewChecker(log).Check(restConfig); err != nil {
		log.Fatal(err)
	}

	config := &config.Config{
		CtxTimeOut:        10 * time.Second,
		Namespace:         m.Namespace,
		WebhookServerHost: m.Host,
		WebhookServerPort: m.Port,
		Fs:                afero.NewOsFs(),
	}
	ctx := ctxlog.NewManagerContext(log)

	mgr, err := operator.NewManager(ctx, config, restConfig, manager.Options{Namespace: m.Namespace})
	if err != nil {
		log.Fatal(err)
	}

	for _, _ = range m.Extensions {
		_ = NewWebHook()
	}

	log.Fatal(mgr.Start(signals.SetupSignalHandler()))

}
