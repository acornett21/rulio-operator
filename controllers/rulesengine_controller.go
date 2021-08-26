/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	rulesv1alpha1 "github.com/acornett21/rulio-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RulesEngineReconciler reconciles a RulesEngine object
type RulesEngineReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

//+kubebuilder:rbac:groups=rules.quay.io,resources=rulesengines,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rules.quay.io,resources=rulesengines/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=rules.quay.io,resources=rulesengines/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *RulesEngineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := r.Log.WithValues("rulesengine", req.NamespacedName)

	rulesEngine := &rulesv1alpha1.RulesEngine{}
	err := r.Get(ctx, req.NamespacedName, rulesEngine)

	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("RulesEngine resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}

		log.Error(err, "Failed to get RulesEngine")
	}

	foundDeployment := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: rulesEngine.Name, Namespace: rulesEngine.Namespace}, foundDeployment)
	if err != nil && errors.IsNotFound(err) {
		deployment := r.deploymentForRulesEngine(rulesEngine)
		log.Info("Creating a new Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)

		err = r.Create(ctx, deployment)
		if err != nil {
			log.Error(err, "Failed to create new Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
			return ctrl.Result{}, err
		}

		// Deployment created successfully - return and requeue
		log.Info("Deployment Created Successfully", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}

	size := rulesEngine.Spec.Size
	if *foundDeployment.Spec.Replicas != size {
		foundDeployment.Spec.Replicas = &size
		err = r.Update(ctx, foundDeployment)
		if err != nil {
			log.Error(err, "Failed to update Deployment", "Deployment.Namespace", foundDeployment.Namespace, "Deployment.Name", foundDeployment.Name)
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, err
	}

	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(rulesEngine.Namespace),
		client.MatchingLabels(labelsForRulesEngine(rulesEngine.Name)),
	}

	if err = r.List(ctx, podList, listOpts...); err != nil {
		log.Error(err, "Failed to list pods", "Memcached.Namespace", rulesEngine.Namespace, "Memcached.Name", rulesEngine.Name)
		return ctrl.Result{}, err
	}

	podNames := getPodNames(podList.Items)
	if !reflect.DeepEqual(podNames, rulesEngine.Status.Nodes) {
		rulesEngine.Status.Nodes = podNames
		err := r.Status().Update(ctx, rulesEngine)
		if err != nil {
			log.Error(err, "Failed to update RulesEngine status")
			return ctrl.Result{}, err
		}
	}

	//todo-adam need logic to write a service
	foundService := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: rulesEngine.Name, Namespace: rulesEngine.Namespace}, foundService)
	if err != nil && errors.IsNotFound(err) {
		service := r.serviceForRulesEngine(rulesEngine)
		log.Info("Creating a new Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)

		err = r.Create(ctx, service)
		if err != nil {
			log.Error(err, "Failed to create new Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
			return ctrl.Result{}, err
		}

		// Service created successfully - return and requeue
		log.Info("Service Created Successfully", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Service")
		return ctrl.Result{}, err
	}

	//todo-adam need logic to write a route

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RulesEngineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&rulesv1alpha1.RulesEngine{}).
		Owns(&appsv1.Deployment{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 2}).
		Complete(r)
}

func (r *RulesEngineReconciler) deploymentForRulesEngine(re *rulesv1alpha1.RulesEngine) *appsv1.Deployment {

	labels := labelsForRulesEngine(re.Name)
	replicas := re.Spec.Size

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      re.Name,
			Namespace: re.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "quay.io/acornett/rulio:v0.0.1",
						Name:  "rulesengine",
						//todo-adam leave this empty for now and see if it loads
						//Command: []string{},
						Ports: []corev1.ContainerPort{{
							ContainerPort: 8001,
							Name:          "rulesengine",
						}},
						ImagePullPolicy: "Always",
					}},
				},
			},
		},
	}

	ctrl.SetControllerReference(re, deployment, r.Scheme)

	return deployment
}

func (r *RulesEngineReconciler) serviceForRulesEngine(re *rulesv1alpha1.RulesEngine) *corev1.Service {

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      re.Name,
			Namespace: re.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Port:       8001,
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromInt(8001),
			}},
			Selector: labelsForRulesEngine(re.Name),
			Type:     corev1.ServiceTypeClusterIP,
		},
	}

	ctrl.SetControllerReference(re, service, r.Scheme)

	return service
}

func labelsForRulesEngine(name string) map[string]string {
	return map[string]string{"app": "rulesengine", "rulesengine_cr": name}
}

func getPodNames(pods []corev1.Pod) []string {

	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}
