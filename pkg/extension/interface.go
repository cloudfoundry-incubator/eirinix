package extension

import (
	"context"

	"code.cloudfoundry.org/cf-operator/pkg/kube/util/config"
	"go.uber.org/zap"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

type Extension interface {
	Handle(context.Context, types.Request) types.Response
}

type MutatingWebHook interface {
	Extension
	InjectClient(c client.Client) error
	InjectDecoder(d types.Decoder) error
	RegisterAdmissionWebHook(WebHookOptions) (*admission.Webhook, error)
}

type WebHookHandler func(log *zap.SugaredLogger, config *config.Config, manager manager.Manager, server *webhook.Server) (*admission.Webhook, error)
type KubeHandler func(ctx context.Context, req types.Request) types.Response

type ExtensionManager interface {
	AddExtension(e Extension)
	Start() error
	ListExtensions() []Extension
	Kube() (*rest.Config, error)
	//Logger(*zap.SugaredLogger)
}

func NewWebHook(h KubeHandler) MutatingWebHook {
	return &DefaultMutatingWebHook{KubeHandle: h}
}

// InjectClient injects the client.
func (m *DefaultMutatingWebHook) InjectClient(c client.Client) error {
	m.client = c
	return nil
}

// InjectDecoder injects the decoder.
func (m *DefaultMutatingWebHook) InjectDecoder(d types.Decoder) error {
	m.decoder = d
	return nil
}

//func (d *DefaultMutatingWebHook) Handle(log *zap.SugaredLogger, config *config.Config, manager manager.Manager, server *webhook.Server) (*admission.Webhook, error) {
//return d.Handle(log, config, manager, server)
//}

func (d *DefaultMutatingWebHook) Handle(ctx context.Context, req types.Request) types.Response {
	return d.KubeHandle(ctx, req)
}
