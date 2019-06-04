package extension

import (
	"context"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

// Extension is the Eirini Extension interface
type Extension interface {
	Handle(context.Context, Manager, *corev1.Pod, types.Request) types.Response
}

// MutatingWebHook is the interface of the generated webhook
// from the Extension
type MutatingWebHook interface {
	Handle(context.Context, types.Request) types.Response
	InjectClient(c client.Client) error
	InjectDecoder(d types.Decoder) error
	RegisterAdmissionWebHook(WebHookOptions) (*admission.Webhook, error)
}

// Manager is the interface of the manager of registered Eirini extensions
type Manager interface {
	AddExtension(e Extension)
	Start() error
	ListExtensions() []Extension
	GetKubeConnection() (*rest.Config, error)
	GetLogger() *zap.SugaredLogger
}
