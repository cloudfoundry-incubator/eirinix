package extension

import (
	"context"

	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/client"
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
type ExtensionManager interface {
	AddExtension(e Extension)
	Start() error
	ListExtensions() []Extension
	Kube() (*rest.Config, error)
	//Logger(*zap.SugaredLogger)
}
