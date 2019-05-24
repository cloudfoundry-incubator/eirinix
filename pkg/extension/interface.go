package extension

import (
	"context"

	"github.com/cloudflare/cfssl/config"
	"go.uber.org/zap"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

type Extension interface {
	Handle(context.Context, types.Request) types.Response
}

type WebHookHandler func(log *zap.SugaredLogger, config *config.Config, manager manager.Manager, server *webhook.Server) (*admission.Webhook, error)
type KubeHandler func(ctx context.Context, req types.Request) types.Response
type WebHookOptions struct {
	Path        string
	MatchLabels map[string]string
	// XXX: Rember it needs to be configurable
	FailurePolicy admissionregistrationv1beta1.FailurePolicyType
}
type MutatingWebHook interface {
	Extension
	InjectClient(c client.Client) error
	InjectDecoder(d types.Decoder) error
}

type DefaultMutatingWebHook struct {
	decoder types.Decoder
	client  client.Client
	//WebHookHandle WebHookHandler
	KubeHandle KubeHandler
}

type ExtensionManager interface{}

type DefaultExtensionManager struct {
	Extensions []Extension
}

func NewWebHook() MutatingWebHook {
	return &DefaultMutatingWebHook{}
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

// NewExtensionManager returns the default ExtensionManager
func NewExtensionManager() ExtensionManager {
	return &DefaultExtensionManager{}
}

func (m *DefaultExtensionManager) AddExtension(e Extension) {
	m.Extensions = append(m.Extensions, e)
}

//func (d *DefaultMutatingWebHook) Handle(log *zap.SugaredLogger, config *config.Config, manager manager.Manager, server *webhook.Server) (*admission.Webhook, error) {
//return d.Handle(log, config, manager, server)
//}

func (d *DefaultMutatingWebHook) Handle(ctx context.Context, req types.Request) types.Response {
	return d.KubeHandle(ctx, req)
}

func (m *DefaultExtensionManager) Start() {

	for _, _ = range m.Extensions {
		_ = NewWebHook()
	}

}
