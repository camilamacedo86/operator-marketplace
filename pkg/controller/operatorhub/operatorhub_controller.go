package operatorhub

import (
	"context"
	"sigs.k8s.io/controller-runtime/pkg/builder"

	configv1 "github.com/openshift/api/config/v1"
	mktconfig "github.com/operator-framework/operator-marketplace/pkg/apis/config/v1"
	"github.com/operator-framework/operator-marketplace/pkg/controller/options"
	"github.com/operator-framework/operator-marketplace/pkg/operatorhub"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Add creates a new OperatorHub Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, _ options.ControllerOptions) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	client := mgr.GetClient()
	return &ReconcileOperatorHub{
		client:  client,
		handler: operatorhub.NewHandler(client),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	if !mktconfig.IsAPIAvailable() {
		return nil
	}

	// We only care if the event came from the cluster config.
	pred := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			if e.Object.GetName() == operatorhub.DefaultName {
				return true
			}
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			if e.ObjectOld.GetName() == operatorhub.DefaultName {
				return true
			}
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			if e.Object.GetName() == operatorhub.DefaultName {
				// If DeleteStateUnknown is true it implies that the Delete event was missed
				// and we can ignore it.
				if e.DeleteStateUnknown {
					return false
				}
				return true
			}
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			if e.Object.GetName() == operatorhub.DefaultName {
				return true
			}
			return false
		},
	}

	return builder.ControllerManagedBy(mgr).
		Named("operatorhub-controller").
		For(&configv1.OperatorHub{}).
		WithEventFilter(pred).
		Complete(r)

}

// blank assignment to verify that ReconcileOperatorHub implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileOperatorHub{}

// ReconcileOperatorHub reconciles a OperatorHub object
type ReconcileOperatorHub struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client  client.Client
	handler operatorhub.Handler
}

// Reconcile reads that state of the cluster for a OperatorHub object and makes changes based on the state read
// and what is in the OperatorHub.Spec
func (r *ReconcileOperatorHub) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log.Infof("Reconciling OperatorHub %s", request.Name)

	// Fetch the OperatorHub instance
	instance := &configv1.OperatorHub{}
	if err := r.client.Get(ctx, request.NamespacedName, instance); err != nil {
		// Request object not found, could have been deleted after reconcile request.
		// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}
	if err := r.handler.Handle(ctx, instance); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
