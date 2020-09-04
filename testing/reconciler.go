package testing

import (
	"context"
	"fmt"
	"time"

	eirinix "code.cloudfoundry.org/eirinix"
	log "code.cloudfoundry.org/quarks-utils/pkg/ctxlog"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type testReconciler struct {
	mgr eirinix.Manager
}

func (r *testReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx := log.NewContextWithRecorder(r.mgr.GetContext(), "test-reconciler", r.mgr.GetKubeManager().GetEventRecorderFor("test-recorder"))
	pod := &corev1.Pod{}

	// Set the ctx to be Background, as the top-level context for incoming requests.
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	log.Info(ctx, "Reconciling pod ", request.NamespacedName)
	if err := r.mgr.GetKubeManager().GetClient().Get(ctx, request.NamespacedName, pod); err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	// Simply make sure our annotation is there!
	pod.ObjectMeta.Annotations["touched"] = "yes"
	err := r.mgr.GetKubeManager().GetClient().Update(ctx, pod)
	if err != nil {
		log.WithEvent(pod, "UpdateError").Errorf(ctx, "Failed to update pod annotation '%s/%s' (%v): %s", pod.Namespace, pod.Name, pod.ResourceVersion, err)
		return reconcile.Result{Requeue: true}, nil
	}
	return reconcile.Result{}, nil
}

func (r *testReconciler) Register(m eirinix.Manager) error {
	r.mgr = m

	c, err := controller.New("test-controller", m.GetKubeManager(), controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return errors.Wrap(err, "Adding test controller to manager failed.")
	}

	// watch pods, trigger if one pod is created
	p := predicate.Funcs{
		CreateFunc:  func(e event.CreateEvent) bool { return true },
		DeleteFunc:  func(e event.DeleteEvent) bool { return true },
		GenericFunc: func(e event.GenericEvent) bool { return true },
		UpdateFunc:  func(e event.UpdateEvent) bool { return true },
	}
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
			pod := a.Object.(*corev1.Pod)

			result := []reconcile.Request{}
			result = append(result, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      pod.Name,
					Namespace: pod.Namespace,
				}})
			return result
		}),
	}, p)
	if err != nil {
		return errors.Wrapf(err, "Watching secrets failed in Restart controller failed.")
	}

	return nil
}

type EditImageReconciler struct {
	mgr eirinix.Manager
}

func (r *EditImageReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx := log.NewContextWithRecorder(r.mgr.GetContext(), "test-reconciler", r.mgr.GetKubeManager().GetEventRecorderFor("test-recorder"))
	pod := &corev1.Pod{}

	// Set the ctx to be Background, as the top-level context for incoming requests.
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	log.Info(ctx, "Reconciling pod ", request.NamespacedName)
	if err := r.mgr.GetKubeManager().GetClient().Get(ctx, request.NamespacedName, pod); err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	pod.Spec.Containers[0].Image = "opensuse/leap"
	err := r.mgr.GetKubeManager().GetClient().Update(ctx, pod)
	if err != nil {
		fmt.Println("Error during pod update", err)
		log.WithEvent(pod, "UpdateError").Errorf(ctx, "Failed to update pod annotation '%s/%s' (%v): %s", pod.Namespace, pod.Name, pod.ResourceVersion, err)
		return reconcile.Result{Requeue: true}, nil
	}
	return reconcile.Result{}, nil
}

func (r *EditImageReconciler) Register(m eirinix.Manager) error {
	r.mgr = m

	c, err := controller.New("test-controller", m.GetKubeManager(), controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return errors.Wrap(err, "Adding test controller to manager failed.")
	}

	// watch pods, trigger if one pod is created
	p := predicate.Funcs{
		CreateFunc:  func(e event.CreateEvent) bool { return false },
		DeleteFunc:  func(e event.DeleteEvent) bool { return false },
		GenericFunc: func(e event.GenericEvent) bool { return false },
		UpdateFunc:  func(e event.UpdateEvent) bool { return true },
	}
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
			pod := a.Object.(*corev1.Pod)

			result := []reconcile.Request{}
			result = append(result, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      pod.Name,
					Namespace: pod.Namespace,
				}})
			return result
		}),
	}, p)
	if err != nil {
		return errors.Wrapf(err, "Watching secrets failed in Restart controller failed.")
	}

	return nil
}
