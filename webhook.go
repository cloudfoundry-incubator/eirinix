package extension

import (
	"context"
	"fmt"
	"net/http"

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

// DefaultMutatingWebHook is the implementation of the WebHook generated out of the Eirini Extension
type DefaultMutatingWebHook struct {
	decoder types.Decoder
	client  client.Client
	//WebHookHandle WebHookHandler
	EiriniExtension  Extension
	FilterEiriniApps bool
}

// GetPod retrieves a pod from a types.Request
func (w *DefaultMutatingWebHook) GetPod(req types.Request) (*corev1.Pod, error) {
	pod := &corev1.Pod{}
	err := w.decoder.Decode(req, pod)
	return pod, err
}

// WebHookOptions are the options required to register a WebHook to the WebHook server
type WebHookOptions struct {
	ID               string // Webhook path will be generated out of that
	MatchLabels      map[string]string
	FailurePolicy    admissionregistrationv1beta1.FailurePolicyType
	Namespace        string
	Manager          manager.Manager
	WebHookServer    *webhook.Server
	FilterEiriniApps bool
}

// NewWebHook returns a MutatingWebHook out of an Eirini Extension
func NewWebHook(e Extension) MutatingWebHook {
	return &DefaultMutatingWebHook{EiriniExtension: e}
}

func (w *DefaultMutatingWebHook) getNamespaceSelector(opts WebHookOptions) *metav1.LabelSelector {
	if len(opts.MatchLabels) == 0 {
		return &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"eirini-extensions-ns": opts.Namespace,
			},
		}
	}
	return &metav1.LabelSelector{MatchLabels: opts.MatchLabels}
}

// RegisterAdmissionWebHook registers the Mutating WebHook to the WebHook Server and returns the generated Admission Webhook
func (w *DefaultMutatingWebHook) RegisterAdmissionWebHook(opts WebHookOptions) (*admission.Webhook, error) {

	w.FilterEiriniApps = opts.FilterEiriniApps
	mutatingWebhook, err := builder.NewWebhookBuilder().
		Path(fmt.Sprintf("/%s", opts.ID)).
		Mutating().
		NamespaceSelector(w.getNamespaceSelector(opts)).
		ForType(&corev1.Pod{}).
		Handlers(w).
		WithManager(opts.Manager).
		FailurePolicy(opts.FailurePolicy).
		Build()

	if err != nil {
		return nil, errors.Wrap(err, "couldn't build a new webhook")
	}
	err = opts.WebHookServer.Register(mutatingWebhook)
	if err != nil {
		return nil, errors.Wrap(err, "unable to register the hook with the admission server")
	}
	return mutatingWebhook, nil
}

// InjectClient injects the client.
func (w *DefaultMutatingWebHook) InjectClient(c client.Client) error {
	w.client = c
	return nil
}

// InjectDecoder injects the decoder.
func (w *DefaultMutatingWebHook) InjectDecoder(d types.Decoder) error {
	w.decoder = d
	return nil
}

// Handle delegates the Handle function to the Eirini Extension
func (w *DefaultMutatingWebHook) Handle(ctx context.Context, req types.Request) types.Response {
	if !w.FilterEiriniApps {
		return w.EiriniExtension.Handle(ctx, req)
	}

	pod, err := w.GetPod(req)
	if err != nil {
		return admission.ErrorResponse(http.StatusBadRequest, err)
	}
	podCopy := pod.DeepCopy()

	// Patch only applications pod created by Eirini
	if v, ok := pod.GetLabels()["source_type"]; ok && v == "APP" {

		return w.EiriniExtension.Handle(ctx, req)
	}

	return admission.PatchResponse(pod, podCopy)

}
