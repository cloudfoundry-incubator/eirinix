package extension

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/builder"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

type DefaultMutatingWebHook struct {
	decoder types.Decoder
	client  client.Client
	//WebHookHandle WebHookHandler
	EiriniExtension Extension
}

type WebHookOptions struct {
	Id            string // Webhook path will be generated out of that
	MatchLabels   map[string]string
	FailurePolicy admissionregistrationv1beta1.FailurePolicyType
	Namespace     string
	Manager       manager.Manager
	WebHookServer *webhook.Server
}

func NewWebHook(e Extension) MutatingWebHook {
	return &DefaultMutatingWebHook{EiriniExtension: e}
}

func (m *DefaultMutatingWebHook) getNamespaceSelector(opts WebHookOptions) *metav1.LabelSelector {
	if len(opts.MatchLabels) == 0 {
		return &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"eirini-extensions-ns": opts.Namespace,
			},
		}
	}
	return &metav1.LabelSelector{MatchLabels: opts.MatchLabels}
}

func (m *DefaultMutatingWebHook) RegisterAdmissionWebHook(opts WebHookOptions) (*admission.Webhook, error) {
	//	mutatingWebhook := builder.NewWebhookBuilder()

	mutatingWebhook, err := builder.NewWebhookBuilder().
		Path(fmt.Sprintf("/%s", opts.Id)).
		Mutating().
		NamespaceSelector(m.getNamespaceSelector(opts)).
		ForType(&corev1.Pod{}).
		Handlers(m).
		WithManager(opts.Manager).
		FailurePolicy(admissionregistrationv1beta1.Fail).
		Build()

	if err != nil {
		return nil, errors.Wrap(err, "couldn't build a new webhook")
	}
	// XXX: in opts?
	err = opts.WebHookServer.Register(mutatingWebhook)
	if err != nil {
		return nil, errors.Wrap(err, "unable to register the hook with the admission server")
	}
	return mutatingWebhook, nil
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

func (d *DefaultMutatingWebHook) Handle(ctx context.Context, req types.Request) types.Response {
	return d.EiriniExtension.Handle(ctx, req)
}
