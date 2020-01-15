package canarydeploy

import (
	"context"
	"fmt"

	canaryv1beta1 "github.com/vntbbb/canary-operator/pkg/apis/canary/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_canarydeploy")

// CanaryDeployError pass error in canarydeploy_controller
type CanaryDeployError struct {
	msg string
}

func (e CanaryDeployError) Error() string {
	return e.msg
}

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new CanaryDeploy Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileCanaryDeploy{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("canarydeploy-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource CanaryDeploy
	err = c.Watch(&source.Kind{Type: &canaryv1beta1.CanaryDeploy{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner CanaryDeploy
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &canaryv1beta1.CanaryDeploy{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileCanaryDeploy implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileCanaryDeploy{}

// ReconcileCanaryDeploy reconciles a CanaryDeploy object
type ReconcileCanaryDeploy struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a CanaryDeploy object and makes changes based on the state read
// and what is in the CanaryDeploy.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileCanaryDeploy) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconcile: Reconciling CanaryDeploy")

	// Fetch the canaryDeploy
	canaryDeploy, err := r.getCanaryDeploy(request)
	if canaryDeploy == nil && err == nil {
		reqLogger.Error(err, "Reconcile: canaryDeploy: " + request.NamespacedName.String() + " not exists")
		return reconcile.Result{}, nil 
	} else if err != nil {
		reqLogger.Error(err, "Reconcile: Failed to get canaryDeploy: " + request.NamespacedName.String())
		return reconcile.Result{}, err
	}

	count, err := r.getCanaryPodCount(canaryDeploy) 
	if err != nil {
		reqLogger.Error(err, "Reconcile: Failed to get pods for canaryDeploy: " + request.NamespacedName.String())
		return reconcile.Result{}, err
	}
	canaryDeploy.Status.CanaryReplicas = count
	
	if err := r.client.Update(context.TODO(), canaryDeploy); err != nil {
		reqLogger.Error(err, "Reconcile: Failed to update canaryDeploy: " + request.NamespacedName.String())
		return reconcile.Result{}, err
	}

	if canaryDeploy.Status.CanaryReplicas < canaryDeploy.Spec.CanaryReplicas {
		if canaryDeploy.Status.CanaryReplicas == 0 {
			if err := r.startCanaryDeploy(canaryDeploy); err != nil {
				reqLogger.Error(err, "Reconcile: Failed to start canaryDeploy: " + request.NamespacedName.String())
				return reconcile.Result{}, err
			}
		} else {
			if canaryDeploy.Status.CanaryStatus == canaryv1beta1.CanaryPaused {
				if err := r.continueCanaryDeploy(canaryDeploy); err != nil {
					reqLogger.Error(err, "Reconcile: Failed to continue canaryDeploy: " + request.NamespacedName.String())
					return reconcile.Result{}, err
				}
			}
		}
	} else if canaryDeploy.Status.CanaryReplicas == canaryDeploy.Spec.CanaryReplicas {
		if canaryDeploy.Status.CanaryReplicas == canaryDeploy.Spec.DeployRef.Status.Replicas {
			canaryDeploy.Status.CanaryStatus = canaryv1beta1.CanaryComplete
			if err := r.client.Update(context.TODO(), canaryDeploy); err != nil {
				reqLogger.Error(err, "Reconcile: Failed to set complete to canaryDeploy: " + request.NamespacedName.String())
				return reconcile.Result{}, err
			}
		} else {
			if err := r.pauseCanaryDeploy(canaryDeploy); err != nil {
				reqLogger.Error(err, "Reconcile: Failed to pause canaryDeploy: " + request.NamespacedName.String())
				return reconcile.Result{}, err
			}
		}
	} else {
		reqLogger.Error(err, "Reconcile: canaryDeploy replicas is more than expected: " + request.NamespacedName.String())
		return reconcile.Result{}, nil
	}

	// canary is completed - don't requeue
	return reconcile.Result{}, nil
}

func (r *ReconcileCanaryDeploy) tidyDeployment(canaryDeploy *canaryv1beta1.CanaryDeploy, deployment *appsv1.Deployment) error {
	namespacedName := types.NamespacedName{
		Namespace: canaryDeploy.Spec.DeployRef.Namespace, 
		Name: canaryDeploy.Spec.DeployRef.Name,
	}
	
	AlreadyOwnedFlag := false
	if err := controllerutil.SetControllerReference(canaryDeploy, deployment, r.scheme); err != nil {
		if _, ok := err.(*controllerutil.AlreadyOwnedError); !ok {
			return CanaryDeployError{
				fmt.Sprint(
					"tidyDeployment: Failed to set controller for deployment: ", 
					namespacedName.String(), 
					", Error: ",
					err.Error(),
				),
			} 
		}
		
		AlreadyOwnedFlag = true
	}

	if AlreadyOwnedFlag == true && *deployment.Spec.Replicas == *canaryDeploy.Spec.DeployRef.Spec.Replicas {
		return nil
	}

	deployment.Spec.Replicas = canaryDeploy.Spec.DeployRef.Spec.Replicas
	if err := r.client.Update(context.TODO(), deployment); err != nil {
		return CanaryDeployError {
			fmt.Sprint(
				"tidyDeployment: Failed to update deployment: ", 
				namespacedName.String(),
				", Error: ", 
				err.Error(),
			),
		}
	}

	return nil
}

func (r *ReconcileCanaryDeploy) getCanaryDeploy(request reconcile.Request) (*canaryv1beta1.CanaryDeploy, error) {
	canaryDeploy := &canaryv1beta1.CanaryDeploy{}
	if err := r.client.Get(context.TODO(), request.NamespacedName, canaryDeploy); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return nil, nil
		}
		
		return nil, CanaryDeployError{fmt.Sprint("getCanaryDeploy: ", err.Error())}
	}

	return canaryDeploy, nil
}

func (r *ReconcileCanaryDeploy) startCanaryDeploy(canaryDeploy *canaryv1beta1.CanaryDeploy) error {
	deployment := &appsv1.Deployment{}
	namespacedName := types.NamespacedName{
		Namespace: canaryDeploy.Spec.DeployRef.Namespace, 
		Name: canaryDeploy.Spec.DeployRef.Name,
	}
	
	if err := r.client.Get(context.TODO(), namespacedName, deployment); err != nil {
		return CanaryDeployError{
			fmt.Sprint(
				"startCanaryDeploy: Failed to get deployment: ", 
				namespacedName.String(), 
				", Error: ",
				err.Error(),
			),
		}
	}

	// Tidy deployment
	// Set controller to canarydeploy
	// Set replicas to canarydeploy 
	if err := r.tidyDeployment(canaryDeploy, deployment); err != nil {
		return CanaryDeployError{
			fmt.Sprint(
				"startCanaryDeploy: Failed to tidy deployment: ", 
				namespacedName.String(), 
				", Error: ",
				err.Error(),
			),
		}
	}
	
	// start to do canary
	canaryLabels := *generateLabels(canaryDeploy)
	deploymentLabels := deployment.Spec.Template.Labels
	if deploymentLabels["canaryVersion"] == canaryLabels["canaryVersion"] {
		return nil
	}

	deployment = canaryDeploy.Spec.DeployRef.DeepCopy()
	deployment.Spec.Paused = false
	controllerutil.SetControllerReference(canaryDeploy, deployment, r.scheme)
	
	// update deployment
	if err := r.client.Update(context.TODO(), deployment); err != nil {
		return CanaryDeployError {
			fmt.Sprint(
				"doCanaryDeploy: Failed to update deployment: ", 
				namespacedName.String(),
				", Error: ", 
				err.Error(),
			),
		}
	}

	// update canaryDeploy
	canaryDeploy.Status.CanaryStatus = canaryv1beta1.CanaryActive
	if err := r.client.Update(context.TODO(), canaryDeploy); err != nil {
		return CanaryDeployError {
			fmt.Sprint(
				"doCanaryDeploy: Failed to update canaryDeploy: ", 
				namespacedName.String(),
				", Error: ", 
				err.Error(),
			),
		}
	}

	return nil
}

func (r *ReconcileCanaryDeploy) pauseCanaryDeploy(canaryDeploy *canaryv1beta1.CanaryDeploy) error {
	deployment := &appsv1.Deployment{}
	namespacedName := types.NamespacedName{
		Namespace: canaryDeploy.Spec.DeployRef.Namespace, 
		Name: canaryDeploy.Spec.DeployRef.Name,
	}
	if err := r.client.Get(context.TODO(), namespacedName, deployment); err != nil {
		return CanaryDeployError {
			fmt.Sprint(
				"pauseCanaryDeploy: Failed to get deployment: ", 
				namespacedName.String(),
				", Error: ", 
				err.Error(),
			),
		}
	}
	
	deployment.Spec.Paused = true
	err := r.client.Update(context.TODO(), deployment)
	if err != nil {
		return CanaryDeployError {
			fmt.Sprint(
				"pauseCanaryDeploy: Failed to update deployment: ", 
				namespacedName.String(),
				", Error: ", 
				err.Error(),
			),
		}
	}

	canaryDeploy.Status.CanaryStatus = canaryv1beta1.CanaryPaused
	if err := r.client.Update(context.TODO(), canaryDeploy); err != nil {
		return CanaryDeployError {
			fmt.Sprint(
				"pauseCanaryDeploy: Failed to update deployment: ", 
				namespacedName.String(),
				", Error: ", 
				err.Error(),
			),
		}
	}
	
	return nil
}

func (r *ReconcileCanaryDeploy) continueCanaryDeploy(canaryDeploy *canaryv1beta1.CanaryDeploy) error {
	deployment := &appsv1.Deployment{}
	namespacedName := types.NamespacedName{
		Namespace: canaryDeploy.Spec.DeployRef.Namespace, 
		Name: canaryDeploy.Spec.DeployRef.Name,
	}
	
	if err := r.client.Get(context.TODO(), namespacedName, deployment); err != nil {
		return CanaryDeployError {
			fmt.Sprint(
				"continueCanaryDeploy: Failed to get deployment: ",
				namespacedName.String(), 
				", error: ", 
				err.Error(),
			),
		}
	}
	
	deployment.Spec.Paused = false
	if err := r.client.Update(context.TODO(), deployment); err != nil {
		return CanaryDeployError {
			fmt.Sprint(
				"continueCanaryDeploy: Failed to update deployment: ", 
				namespacedName.String(),
				", error: ",
				err.Error(),
			),
		}
	}

	canaryDeploy.Status.CanaryStatus = canaryv1beta1.CanaryActive
	if err := r.client.Update(context.TODO(), canaryDeploy); err != nil {
		nsname := types.NamespacedName{Namespace: canaryDeploy.Namespace, Name: canaryDeploy.Name}
		return CanaryDeployError {
			fmt.Sprint(
				"continueCanaryDeploy: Failed to update canaryDeploy: ", 
				nsname.String(), 
				", error: ", 
				err.Error(),
			),
		}
	}
	
	return nil
}

func (r *ReconcileCanaryDeploy) getCanaryPodCount(canaryDeploy *canaryv1beta1.CanaryDeploy) (int32, error) {
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(canaryDeploy.Namespace),
		client.MatchingLabels(*generateLabels(canaryDeploy)),
	}
	
	if err := r.client.List(context.TODO(), podList, listOpts...); err != nil {
		if errors.IsNotFound(err) {
			fmt.Println("wugaojun: pod list is not found")
			return 0, nil
		}
		nsname := types.NamespacedName{canaryDeploy.Namespace, canaryDeploy.Name}
		return 0, CanaryDeployError{
			fmt.Sprint(
				"getCanaryPodList: Failed to list pods for canarydeploy: ", 
				nsname.String(),
				", Error: ", 
				err.Error(),
			),
		}
	}

	return int32(len(podList.Items)), nil
}


func generateLabels(canaryDeploy *canaryv1beta1.CanaryDeploy) *map[string]string {
	return &map[string]string{
		"canaryName": canaryDeploy.Spec.CanaryName, 
		"canaryVersion": canaryDeploy.Spec.CanaryVersion,
	}
}