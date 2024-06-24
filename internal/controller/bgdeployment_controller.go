package controller

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1 "k8s.io/api/apps/v1"
    corev1 "k8s.io/api/core/v1"
	bgdeploymentv1alpha1 "github.com/NikilLepcha/bluegreen-deployment/api/v1alpha1"
)

// BGDeploymentReconciler reconciles a BGDeployment object
type BGDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log logr.Logger
}

//+kubebuilder:rbac:groups=bgdeployment.example.com,resources=bgdeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=bgdeployment.example.com,resources=bgdeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=bgdeployment.example.com,resources=bgdeployments/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *BGDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("bluegreen-deployment", req.NamespacedName)

	// Fetch the BlueGreenDeployment instance
	bgd := &bgdeploymentv1alpha1.BGDeployment{}
	err := r.Get(ctx, req.NamespacedName, bgd)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Determine the color for the new deployment
	activeColor := green
	if bgd.Status.ActiveColor == activeColor {
		activeColor := blue
	}

	// Create or update the deployment
	newDeploymentName := fmt.Sprintf("%s-%s", strings.Split(bgd.Name, "-")[0], activeColor)
	newDeployment := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: newDeploymentName, Namespace: bgd.Namespace}, newDeployment)
	if err != nil && errors.IsNotFound(err) {
		newDeployment = r.constructDeployment(newDeploymentName, activeColor, bgd)
		err = r.Create(ctx, newDeployment)
		if err != nil {
			return ctrl.Result{}, err
		}
	} else if err != nil {
		return ctrl.Result{}, err
	} else {
		updated = r.updateDeployment(newDeployment, bgd)
		if updated {
			err = r.Update(ctx, newDeployment)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	// Create or update the service to point to the new deployment
	service := &corev1.Service{}
    err = r.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-%s", strings.Split(bgd.Name, "-")[0], "svc"), Namespace: bgd.Namespace}, service)
    if err != nil && errors.IsNotFound(err) {
        service = r.constructService(newDeployment, activeColor)
        err = r.Create(ctx, service)
        if err != nil {
            return ctrl.Result{}, err
        }
    } else if err != nil {
        return ctrl.Result{}, err
    } else {
        updated := r.updateService(service, activeColor)
        if updated {
            err = r.Update(ctx, service)
            if err != nil {
                return ctrl.Result{}, err
            }
        }
    }

	// Update the status to reflect the new active color
    bgd.Status.ActiveColor = activeColor
    err = r.Status().Update(ctx, bgd)
    if err != nil {
        return ctrl.Result{}, err
    }

	return ctrl.Result{}, nil
}

func (r *BGDeploymentReconciler) constructDeployment(newDeploymentName string, activeColor string, bgd *bgdeploymentv1alpha1.BGDeployment) *appsv1.Deployment {
	labels := map[string]string {
		app: strings.Split(newDeploymentName, "-")[0],
		color: activeColor,
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: newDeploymentName,
			Namespace: bgd.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &bgd.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name: "app",
						Image: bgd.Spec.Image,
						Ports: []corev1.ContainerPort{{
								ContainerPort: 80,
							}},
					}},
				},
			},
		},
	}
}

func (r *BGDeploymentReconciler) updateDeployment(deployment *appsv1.Deployment, bgd *bgdeploymentv1alpha1.BGDeployment) bool {
	updated := false
	if *deployment.Spec.Replicas != bgd.Spec.Replicas {
		deployment.Spec.Replicas = bgd.Spec.Replicas
		updated = true
	}
	if deployment.Spec.Template.Spec.Containers[0].Image != bgd.Spec.Image {
		deployment.Spec.Template.Spec.Containers[0].Image = bgd.Spec.Image
		updated = true
	}
	return updated
}

func (r *BGDeploymentReconciler) constructService(newDeployment *appsv1.Deployment, activeColor string) *corev1.Service {
	labels := map[string]string {
		app: strings.Split(newDeployment.Name, "-")[0]
		color: activeColor,
	}
	
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", strings.Split(newDeployment.Name, "-")[0], "svc"),
			Namespace: newDeployment.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name: "http",
					Protocol: corev1.ProtocolTCP,
					Port: 80,
					TargetPort: intstr.FromInt(80),
				},
			},
		},
	}
}

func (r *BGDeploymentReconciler) updateService(service *corev1.Service, activeColor string) bool {
	updated := false
    if service.Spec.Selector["color"] != activeColor {
        service.Spec.Selector["color"] = activeColor
        updated = true
    }
    return updated
}

// SetupWithManager sets up the controller with the Manager.
func (r *BGDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&bgdeploymentv1alpha1.BGDeployment{}).
		Owns(&appsv1.Deployment{}).
        Owns(&corev1.Service{}).
        Complete(r)
}
