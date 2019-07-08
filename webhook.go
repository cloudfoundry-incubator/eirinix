package extension

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/builder"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

type setReferenceFunc func(owner, object metav1.Object, scheme *runtime.Scheme) error

// DefaultMutatingWebhook is the implementation of the Webhook generated out of the Eirini Extension
type DefaultMutatingWebhook struct {
	decoder types.Decoder
	client  client.Client

	// EiriniExtension is the Eirini extension associated with the webhook
	EiriniExtension Extension

	// EiriniExtensionManager is the Manager which will be injected into the Handle.
	EiriniExtensionManager Manager

	// FilterEiriniApps indicates if the webhook will filter Eirini apps or not.
	FilterEiriniApps bool
	setReference     setReferenceFunc
}

// GetPod retrieves a pod from a types.Request
func (w *DefaultMutatingWebhook) GetPod(req types.Request) (*corev1.Pod, error) {
	pod := &corev1.Pod{}
	if w.decoder == nil {
		return nil, errors.New("No decoder injected")
	}
	err := w.decoder.Decode(req, pod)
	return pod, err
}

// WebhookOptions are the options required to register a WebHook to the WebHook server
type WebhookOptions struct {
	ID             string // Webhook path will be generated out of that
	MatchLabels    map[string]string
	Manager        manager.Manager
	WebhookServer  *webhook.Server
	ManagerOptions ManagerOptions
}

// NewWebhook returns a MutatingWebhook out of an Eirini Extension
func NewWebhook(e Extension, m Manager) MutatingWebhook {
	return &DefaultMutatingWebhook{EiriniExtensionManager: m, EiriniExtension: e, setReference: controllerutil.SetControllerReference}
}

func (w *DefaultMutatingWebhook) getNamespaceSelector(opts WebhookOptions) *metav1.LabelSelector {
	if len(opts.MatchLabels) == 0 {
		return &metav1.LabelSelector{
			MatchLabels: map[string]string{
				opts.ManagerOptions.getDefaultNamespaceLabel(): opts.ManagerOptions.Namespace,
			},
		}
	}
	return &metav1.LabelSelector{MatchLabels: opts.MatchLabels}
}

// RegisterAdmissionWebHook registers the Mutating WebHook to the WebHook Server and returns the generated Admission Webhook
func (w *DefaultMutatingWebhook) RegisterAdmissionWebHook(opts WebhookOptions) (*admission.Webhook, error) {
	if opts.ManagerOptions.FailurePolicy == nil {
		return nil, errors.New("No failure policy set")
	}
	if opts.ManagerOptions.FilterEiriniApps != nil {
		w.FilterEiriniApps = *opts.ManagerOptions.FilterEiriniApps
	} else {
		w.FilterEiriniApps = true
	}

	MutatingWebhook, err := builder.NewWebhookBuilder().
		Path(fmt.Sprintf("/%s", opts.ID)).
		Mutating().
		NamespaceSelector(w.getNamespaceSelector(opts)).
		ForType(&corev1.Pod{}).
		Name(fmt.Sprintf("%s.%s.org", opts.ID, opts.ManagerOptions.OperatorFingerprint)).
		Handlers(w).
		WithManager(opts.Manager).
		FailurePolicy(*opts.ManagerOptions.FailurePolicy).
		Build()

	if err != nil {
		return nil, errors.Wrap(err, "couldn't build a new webhook")
	}
	err = opts.WebhookServer.Register(MutatingWebhook)
	if err != nil {
		return nil, errors.Wrap(err, "unable to register the hook with the admission server")
	}
	return MutatingWebhook, nil
}

// InjectClient injects the client.
func (w *DefaultMutatingWebhook) InjectClient(c client.Client) error {
	w.client = c
	return nil
}

// InjectDecoder injects the decoder.
func (w *DefaultMutatingWebhook) InjectDecoder(d types.Decoder) error {
	w.decoder = d
	return nil
}

// Handle delegates the Handle function to the Eirini Extension
func (w *DefaultMutatingWebhook) Handle(ctx context.Context, req types.Request) types.Response {

	pod, _ := w.GetPod(req)

	if !w.FilterEiriniApps {
		return w.EiriniExtension.Handle(ctx, w.EiriniExtensionManager, pod, req)
	}

	podCopy := pod.DeepCopy()

	// Patch only applications pod created by Eirini
	if v, ok := pod.GetLabels()["source_type"]; ok && v == "APP" {
		return w.EiriniExtension.Handle(ctx, w.EiriniExtensionManager, pod, req)
	}

	return admission.PatchResponse(pod, podCopy)

}
