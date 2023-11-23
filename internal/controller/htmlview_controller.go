/*
Copyright 2023.

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

package controller

import (
	"context"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	applymetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
	applynetworkingv1 "k8s.io/client-go/applyconfigurations/networking/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	viewv1 "github.com/ty-bnn/html-view/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	applyappsv1 "k8s.io/client-go/applyconfigurations/apps/v1"
)

// HtmlViewReconciler reconciles a HtmlView object
type HtmlViewReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=view.ty-bnn.github.io,resources=htmlviews,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=view.ty-bnn.github.io,resources=htmlviews/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=view.ty-bnn.github.io,resources=htmlviews/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the HtmlView object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.0/pkg/reconcile
func (r *HtmlViewReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var htmlView viewv1.HtmlView
	err := r.Get(ctx, req.NamespacedName, &htmlView)
	if errors.IsNotFound(err) {
		return ctrl.Result{}, nil
	}
	if err != nil {
		logger.Error(err, "unable to get MarkdownView", "name", req.NamespacedName)
		return ctrl.Result{}, err
	}

	if !htmlView.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	err = r.reconcileConfigMap(ctx, htmlView)
	if err != nil {
		return ctrl.Result{}, err
	}
	err = r.reconcileDeployment(ctx, htmlView)
	if err != nil {
		return ctrl.Result{}, err
	}
	err = r.reconcileService(ctx, htmlView)
	if err != nil {
		return ctrl.Result{}, err
	}
	err = r.reconcileIngress(ctx, htmlView)
	if err != nil {
		return ctrl.Result{}, err
	}

	return r.updateStatus(ctx, htmlView)
}

// SetupWithManager sets up the controller with the Manager.
func (r *HtmlViewReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&viewv1.HtmlView{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&networkingv1.Ingress{}).
		Complete(r)
}

func (r *HtmlViewReconciler) reconcileConfigMap(ctx context.Context, htmlView viewv1.HtmlView) error {
	logger := log.FromContext(ctx)

	cm := &corev1.ConfigMap{}
	cm.SetNamespace(htmlView.Namespace)
	cm.SetName("html-cm")

	op, err := ctrl.CreateOrUpdate(ctx, r.Client, cm, func() error {
		if cm.Data == nil {
			cm.Data = make(map[string]string)
		}
		for name, content := range htmlView.Spec.Html {
			cm.Data[name] = content
		}
		return nil
	})

	if err != nil {
		logger.Error(err, "unable to create or update ConfigMap")
		return err
	}
	if op != controllerutil.OperationResultNone {
		logger.Info("reconcile ConfigMap successfully", "op", op)
	}
	return nil
}

func (r *HtmlViewReconciler) reconcileDeployment(ctx context.Context, htmlView viewv1.HtmlView) error {
	logger := log.FromContext(ctx)

	depName := "nginx-server"
	viewerImage := "nginx:latest"

	dep := applyappsv1.Deployment(depName, htmlView.Namespace).
		WithName(depName).
		WithSpec(applyappsv1.DeploymentSpec().
			WithReplicas(htmlView.Spec.Replicas).
			WithSelector(applymetav1.LabelSelector().WithMatchLabels(map[string]string{
				"app": depName,
			})).
			WithTemplate(applycorev1.PodTemplateSpec().
				WithLabels(map[string]string{
					"app": depName,
				}).
				WithSpec(applycorev1.PodSpec().
					WithContainers(applycorev1.Container().
						WithName("nginx").
						WithImage(viewerImage).
						WithVolumeMounts(applycorev1.VolumeMount().
							WithName("html-volume").
							WithMountPath("/usr/share/nginx/html"),
						),
					).
					WithVolumes(applycorev1.Volume().
						WithName("html-volume").
						WithConfigMap(applycorev1.ConfigMapVolumeSource().
							WithName("html-cm"),
						),
					),
				),
			),
		)

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(dep)
	if err != nil {
		return err
	}
	patch := &unstructured.Unstructured{
		Object: obj,
	}

	var current appsv1.Deployment
	err = r.Get(ctx, client.ObjectKey{Namespace: htmlView.Namespace, Name: depName}, &current)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	currApplyConfig, err := applyappsv1.ExtractDeployment(&current, "html-view-controller")
	if err != nil {
		return err
	}

	if equality.Semantic.DeepEqual(dep, currApplyConfig) {
		return nil
	}

	err = r.Patch(ctx, patch, client.Apply, &client.PatchOptions{
		FieldManager: "html-view-controller",
		Force:        pointer.Bool(true),
	})

	if err != nil {
		logger.Error(err, "unable to create or update Deployment")
		return err
	}
	logger.Info("reconcile Deployment successfully", "name", depName)
	return nil
}

func (r *HtmlViewReconciler) reconcileService(ctx context.Context, htmlView viewv1.HtmlView) error {
	logger := log.FromContext(ctx)

	svcName := "nginx-service"

	svc := applycorev1.Service(svcName, htmlView.Namespace).
		WithName(svcName).
		WithSpec(applycorev1.ServiceSpec().
			WithPorts(applycorev1.ServicePort().
				WithPort(80),
			).
			WithSelector(map[string]string{
				"app": svcName,
			}),
		)

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(svc)
	if err != nil {
		return err
	}
	patch := &unstructured.Unstructured{
		Object: obj,
	}

	var current corev1.Service
	err = r.Get(ctx, client.ObjectKey{Namespace: htmlView.Namespace, Name: svcName}, &current)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	currApplyConfig, err := applycorev1.ExtractService(&current, "html-view-controller")
	if err != nil {
		return err
	}

	if equality.Semantic.DeepEqual(svc, currApplyConfig) {
		return nil
	}

	err = r.Patch(ctx, patch, client.Apply, &client.PatchOptions{
		FieldManager: "html-view-controller",
		Force:        pointer.Bool(true),
	})

	if err != nil {
		logger.Error(err, "unable to create or update Service")
		return err
	}
	logger.Info("reconcile Service successfully", "name", svcName)
	return nil
}

func (r *HtmlViewReconciler) reconcileIngress(ctx context.Context, htmlView viewv1.HtmlView) error {
	logger := log.FromContext(ctx)

	ingName := "nginx-ingress"

	portNum := htmlView.Spec.Port
	if portNum == 0 {
		portNum = 80
	}

	ing := applynetworkingv1.Ingress(ingName, htmlView.Namespace).
		WithName(ingName).
		WithSpec(applynetworkingv1.IngressSpec().
			WithRules(applynetworkingv1.IngressRule().
				WithHTTP(applynetworkingv1.HTTPIngressRuleValue().
					WithPaths(applynetworkingv1.HTTPIngressPath().
						WithBackend(applynetworkingv1.IngressBackend().
							WithService(applynetworkingv1.IngressServiceBackend().
								WithName("nginx-service").
								WithPort(applynetworkingv1.ServiceBackendPort().
									WithNumber(portNum),
								),
							),
						).
						WithPath("/").
						WithPathType("Prefix"),
					),
				),
			),
		)

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(ing)
	if err != nil {
		return err
	}
	patch := &unstructured.Unstructured{
		Object: obj,
	}

	var current networkingv1.Ingress
	err = r.Get(ctx, client.ObjectKey{Namespace: htmlView.Namespace, Name: ingName}, &current)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	currApplyConfig, err := applynetworkingv1.ExtractIngress(&current, "html-view-controller")
	if err != nil {
		return err
	}

	if equality.Semantic.DeepEqual(ing, currApplyConfig) {
		return nil
	}

	err = r.Patch(ctx, patch, client.Apply, &client.PatchOptions{
		FieldManager: "html-view-controller",
		Force:        pointer.Bool(true),
	})

	if err != nil {
		logger.Error(err, "unable to create or update Ingress")
		return err
	}
	logger.Info("reconcile Ingress successfully", "name", ingName)
	return nil
}

func (r *HtmlViewReconciler) updateStatus(ctx context.Context, htmlView viewv1.HtmlView) (ctrl.Result, error) {
	var dep appsv1.Deployment
	err := r.Get(ctx, client.ObjectKey{Namespace: htmlView.Namespace, Name: "nginx-server"}, &dep)
	if err != nil {
		return ctrl.Result{}, err
	}

	var status viewv1.HtmlViewStatus
	if dep.Status.AvailableReplicas < htmlView.Spec.Replicas {
		status = viewv1.HtmlViewNotReady
	} else if dep.Status.AvailableReplicas == htmlView.Spec.Replicas {
		status = viewv1.HtmlViewRunning
	}

	if htmlView.Status != status {
		htmlView.Status = status
		err = r.Status().Update(ctx, &htmlView)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if htmlView.Status != viewv1.HtmlViewRunning {
		return ctrl.Result{Requeue: true}, nil
	}
	return ctrl.Result{}, nil
}
