package extension

import (
	"context"

	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

// Extension is the Eirini Extension interface
type Extension interface {
	Handle(context.Context, types.Request) types.Response
}

// MutatingWebHook is the interface of the generated webhook
// from the Extension
type MutatingWebHook interface {
	Extension
	InjectClient(c client.Client) error
	InjectDecoder(d types.Decoder) error
	RegisterAdmissionWebHook(WebHookOptions) (*admission.Webhook, error)
}

// Manager is the interface of the manager of registered Eirini extensions
type Manager interface {
	AddExtension(e Extension)
	Start() error
	ListExtensions() []Extension
	KubeConnection() (*rest.Config, error)
	//Logger(*zap.SugaredLogger)
}
