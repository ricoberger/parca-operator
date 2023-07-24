package controllers

import (
	"context"
	"os"
	"time"

	parcav1alpha1 "github.com/ricoberger/parca-operator/api/v1alpha1"
	"github.com/ricoberger/parca-operator/controllers/parcascrapeconfig"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	parcaScrapeConfigDeleteFailed = "ParcaScrapeConfigDeleteFailed"
	parcaScrapeConfigUpdateFailed = "ParcaScrapeConfigUpdateFailed"
	parcaScrapeConfigUpdated      = "ParcaScrapeConfigUpdated"

	parcaScrapeConfigFinalizer = "parca.ricoberger.de/finalizer"
)

// ParcaScrapeConfigReconciler reconciles a ParcaScrapeConfig object
type ParcaScrapeConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=parca.ricoberger.de,resources=parcascrapeconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=parca.ricoberger.de,resources=parcascrapeconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=parca.ricoberger.de,resources=parcascrapeconfigs/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to move the current state of the cluster
// closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *ParcaScrapeConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)
	reconciliationInterval := parcascrapeconfig.GetReconciliationInterval()

	parcaScrapeConfig := &parcav1alpha1.ParcaScrapeConfig{}
	err := r.Get(ctx, req.NamespacedName, parcaScrapeConfig)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request. Owned objects are
			// automatically garbage collected. For additional cleanup logic use finalizers. Return and don't requeue.
			reqLogger.Info("ParcaScrapeConfig resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Failed to get ParcaScrapeConfig.")
		return ctrl.Result{RequeueAfter: reconciliationInterval}, err
	}

	// When the ParcaScrapeConfig is marked for deletion, we remove it from the Parca configuration, before we remove
	// the finalizer and delete the object.
	isMemcachedMarkedToBeDeleted := parcaScrapeConfig.GetDeletionTimestamp() != nil
	if isMemcachedMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(parcaScrapeConfig, parcaScrapeConfigFinalizer) {
			// Remove the ParcaScrapeConfig from the Parca configuration and update the Parca configuration secret. So
			// that it doesn't contain the ParcaScrapeConfig anymore.
			err := parcascrapeconfig.DeleteScrapeConfig(*parcaScrapeConfig)
			if err != nil {
				reqLogger.Error(err, "Failed to delete ParcaScrapeConfig.")
				r.updateConditions(ctx, parcaScrapeConfig, parcaScrapeConfigDeleteFailed, err.Error(), metav1.ConditionFalse, nil)
				return ctrl.Result{RequeueAfter: reconciliationInterval}, err
			}

			finalConfig, err := parcascrapeconfig.GetConfig()
			if err != nil {
				reqLogger.Error(err, "Failed to get Parca configuration.")
				r.updateConditions(ctx, parcaScrapeConfig, parcaScrapeConfigUpdateFailed, err.Error(), metav1.ConditionFalse, nil)
				return ctrl.Result{RequeueAfter: reconciliationInterval}, err
			}

			err = r.Update(ctx, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      os.Getenv("PARCA_SCRAPECONFIG_FINAL_CONFIG_NAME"),
					Namespace: os.Getenv("PARCA_SCRAPECONFIG_FINAL_CONFIG_NAMESPACE"),
				},
				Data: map[string][]byte{
					"parca.yaml": finalConfig,
				},
			})
			if err != nil {
				reqLogger.Error(err, "Failed to update Parca configuration.")
				r.updateConditions(ctx, parcaScrapeConfig, parcaScrapeConfigUpdateFailed, err.Error(), metav1.ConditionFalse, nil)
				return ctrl.Result{RequeueAfter: reconciliationInterval}, err
			}

			// Remove the parcaScrapeConfigFinalizer. Once the finalizer is removed the object will be deleted.
			controllerutil.RemoveFinalizer(parcaScrapeConfig, parcaScrapeConfigFinalizer)
			err = r.Update(ctx, parcaScrapeConfig)
			if err != nil {
				reqLogger.Error(err, "Failed to remove finalizer.")
				r.updateConditions(ctx, parcaScrapeConfig, parcaScrapeConfigDeleteFailed, err.Error(), metav1.ConditionFalse, nil)
				return ctrl.Result{RequeueAfter: reconciliationInterval}, err
			}
		}

		return ctrl.Result{}, nil
	}

	// Get all Pods for the ParcaScrapeConfig based on the specified selector. The Pods are then used to generate the
	// Parca configuration via the SetScrapeConfig function.
	labelSelector, err := metav1.LabelSelectorAsSelector(&parcaScrapeConfig.Spec.Selector)
	if err != nil {
		reqLogger.Error(err, "Failed to build labelSelector for ParcaScrapeConfig.")
		r.updateConditions(ctx, parcaScrapeConfig, parcaScrapeConfigUpdateFailed, err.Error(), metav1.ConditionFalse, nil)
		return ctrl.Result{RequeueAfter: reconciliationInterval}, err
	}

	pods := &corev1.PodList{}
	err = r.List(ctx, pods, &client.ListOptions{
		Namespace:     parcaScrapeConfig.Namespace,
		LabelSelector: labelSelector,
	})
	if err != nil {
		reqLogger.Error(err, "Failed to get Pods for ParcaScrapeConfig.")
		r.updateConditions(ctx, parcaScrapeConfig, parcaScrapeConfigUpdateFailed, err.Error(), metav1.ConditionFalse, nil)
		return ctrl.Result{RequeueAfter: reconciliationInterval}, err
	}

	podIPs, err := parcascrapeconfig.SetScrapeConfig(*parcaScrapeConfig, pods.Items)
	if err != nil {
		reqLogger.Error(err, "Failed to update internal Parca configuration.")
		r.updateConditions(ctx, parcaScrapeConfig, parcaScrapeConfigUpdateFailed, err.Error(), metav1.ConditionFalse, nil)
		return ctrl.Result{RequeueAfter: reconciliationInterval}, err
	}

	// When the Parca configuration was updated, we can update the Parca configuration secret to include the new scrape
	// configuration.
	finalConfig, err := parcascrapeconfig.GetConfig()
	if err != nil {
		reqLogger.Error(err, "Failed to get Parca configuration.")
		r.updateConditions(ctx, parcaScrapeConfig, parcaScrapeConfigUpdateFailed, err.Error(), metav1.ConditionFalse, nil)
		return ctrl.Result{RequeueAfter: reconciliationInterval}, err
	}

	err = r.Update(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      os.Getenv("PARCA_SCRAPECONFIG_FINAL_CONFIG_NAME"),
			Namespace: os.Getenv("PARCA_SCRAPECONFIG_FINAL_CONFIG_NAMESPACE"),
		},
		Data: map[string][]byte{
			"parca.yaml": finalConfig,
		},
	})
	if err != nil {
		reqLogger.Error(err, "Failed to update Parca configuration.")
		r.updateConditions(ctx, parcaScrapeConfig, parcaScrapeConfigUpdateFailed, err.Error(), metav1.ConditionFalse, nil)
		return ctrl.Result{RequeueAfter: reconciliationInterval}, err
	}

	// Finally we add the parcaScrapeConfigFinalizer to the ParcaScrapeConfig. The finilizer is needed so that we can
	// remove the ParcaScrapeConfig from the Parca configuration secret when the ParcaScrapeConfig is deleted.
	if !controllerutil.ContainsFinalizer(parcaScrapeConfig, parcaScrapeConfigFinalizer) {
		controllerutil.AddFinalizer(parcaScrapeConfig, parcaScrapeConfigFinalizer)
		err := r.Update(ctx, parcaScrapeConfig)
		if err != nil {
			reqLogger.Error(err, "Failed to add finalizer.")
			r.updateConditions(ctx, parcaScrapeConfig, parcaScrapeConfigUpdateFailed, err.Error(), metav1.ConditionFalse, nil)
			return ctrl.Result{RequeueAfter: reconciliationInterval}, err
		}
	}

	reqLogger.Info("ParcaScrapeConfig updated.")
	r.updateConditions(ctx, parcaScrapeConfig, parcaScrapeConfigUpdated, "Parca Configuration was updated", metav1.ConditionTrue, podIPs)
	return ctrl.Result{RequeueAfter: reconciliationInterval}, nil
}

func (r *ParcaScrapeConfigReconciler) updateConditions(ctx context.Context, parcaScrapeConfig *parcav1alpha1.ParcaScrapeConfig, reason, message string, status metav1.ConditionStatus, podIPs []string) {
	reqLogger := log.FromContext(ctx)

	parcaScrapeConfig.Status.Conditions = []metav1.Condition{{
		Type:               "ParcaScrapeConfigReconciled",
		Status:             status,
		ObservedGeneration: parcaScrapeConfig.GetGeneration(),
		LastTransitionTime: metav1.NewTime(time.Now()),
		Reason:             reason,
		Message:            message,
	}}
	parcaScrapeConfig.Status.PodIPs = podIPs

	err := r.Status().Update(ctx, parcaScrapeConfig)
	if err != nil {
		reqLogger.Error(err, "Failed to update status.")
	}
}

// ignorePredicate is used to ignore updates to the CR status in which case metadata.Generation does not change. This is
// required to avoid infinite reconcile loops, when add the conditions and PodIPs to the status.
func ignorePredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ParcaScrapeConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&parcav1alpha1.ParcaScrapeConfig{}).
		WithEventFilter(ignorePredicate()).
		Complete(r)
}
