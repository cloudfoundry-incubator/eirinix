package extension

import (
	"context"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"

	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

// Extension is the Eirini Extension interface
//
// An Eirini Extension must implement it by providing only an Handle method which
// will be used as a response to the kube api server.
//
// The Extension typically returns a set of patches defining the difference between the pod received in the request
// and the wanted state from the Extension.
type Extension interface {
	// Handle handles a kubernetes request.
	// It is the main entry point of the Eirini extensions and the arguments are the
	// decoded payloads from the kubeapi server.
	//
	// The manager will attempt to decode a pod from the request if possible and passes it to the Manager.
	Handle(context.Context, Manager, *corev1.Pod, types.Request) types.Response
}

// Watcher is the Eirini Watcher Extension interface.
//
// An Eirini Watcher must implement a Handle method, which is called with the event that occurred in the
// namespace.
type Watcher interface {
	Handle(Manager, watch.Event)
}

// MutatingWebhook is the interface of the generated webhook
// from the Extension
//
// It represent the minimal set of methods that the libraries used behind the scenes expect from a structure
// that implements a Mutating Webhook
type MutatingWebhook interface {
	Handle(context.Context, types.Request) types.Response
	InjectClient(c client.Client) error
	InjectDecoder(d types.Decoder) error
	RegisterAdmissionWebHook(WebhookOptions) (*admission.Webhook, error)
}

// Manager is the interface of the manager for registering Eirini extensions
//
// It will generate webhooks that will satisfy the MutatingWebhook interface from the defined Extensions.
type Manager interface {

	// AddExtension adds an Extension to the manager
	//
	// The manager later on, will register the Extension when Start() is being called.
	AddExtension(e Extension)

	// Start starts the manager infinite loop.
	//
	// Registers all the Extensions and generates
	// the respective mutating webhooks.
	//
	// Returns error in case of failure.
	Start() error

	// ListExtensions returns a list of the current loaded Extension
	ListExtensions() []Extension

	// GetKubeConnection sets up a kube connection if not already present
	//
	// Returns the rest config used to establish a connection to the kubernetes cluster.
	GetKubeConnection() (*rest.Config, error)

	// GetKubeClient sets up a kube client if not already present
	//
	// Returns the kubernetes interface.
	GetKubeClient() (corev1client.CoreV1Interface, error)

	// GetLogger returns the logger of the application. It can be passed an already existing one
	// by using NewManager()
	GetLogger() *zap.SugaredLogger

	// Watch starts the main loop for the registered watchers
	Watch() error

	// AddWatcher register a watcher to EiriniX
	AddWatcher(w Watcher)
}
