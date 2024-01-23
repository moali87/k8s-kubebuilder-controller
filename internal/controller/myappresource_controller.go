/*
Copyright 2024.

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
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	kapps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"angi.myrepo.io/homework/api/v1alpha1"
	appv1alpha1 "angi.myrepo.io/homework/api/v1alpha1"
)

// MyAppResourceReconciler reconciles a MyAppResource object
type MyAppResourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme

    Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=app.interviews.myrepo.io,resources=myappresources,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.interviews.myrepo.io,resources=myappresources/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=app.interviews.myrepo.io,resources=myappresources/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MyAppResource object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *MyAppResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := log.FromContext(ctx)

	// TODO(user): your logic here
    log.Info("Get MyAppResource kind")
    myKind := appv1alpha1.MyAppResource{}
    if err := r.Client.Get(ctx, req.NamespacedName, &myKind); err != nil {
        log.Error(err, "failed to get custom kind") 

        // Although not sure why, it seems we ignore the error and continue.
        // Perhaps the error is automatically removed if the resource is eventually created
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // TODO: Create method to clean up existing resources
    if err := r.cleanUpResources(ctx, log, &myKind); err != nil {
        log.Error(err, "unable to clean up resources")
        return ctrl.Result{}, err
    }
    
    log.Info("checking for existing deployment resources")
    deployment := kapps.Deployment{}
    err := r.Client.Get(ctx, client.ObjectKey{Namespace: myKind.Namespace, Name: myKind.Name}, &deployment)
    if apierrors.IsNotFound(err) {
        log.Info("did not find any existing deployment, creating one")
        deployment = *buildDeploymentTemplate(myKind)
        if err := r.Client.Create(ctx, &deployment); err !=  nil {
            log.Error(err, "unable to create deployment")
            return ctrl.Result{}, err
        }

        r.Recorder.Eventf(&myKind, core.EventTypeNormal, "Created", "Created deployment %s", deployment.Name)
        return ctrl.Result{}, nil
    }
    if err != nil {
        log.Error(err, "failed to get existind deployment")
        return ctrl.Result{}, err
    }

    setReplicas := int32(1)
    if myKind.Spec.ReplicaCount != nil {
        setReplicas = *myKind.Spec.ReplicaCount
    }

    if *&deployment.Spec.Replicas != &setReplicas {
        log.Info("updating replica count to", setReplicas)
        deployment.Spec.Replicas = &setReplicas
        if err := r.Client.Update(ctx, &deployment); err != nil {
            log.Error(err, "unable to update replica count")
            return ctrl.Result{}, err
        }

        r.Recorder.Eventf(&myKind, core.EventTypeNormal, "Scaled", "Scaled deployment %q to %d replicas", deployment.Name, setReplicas)
        return ctrl.Result{}, nil
    }

    myKind.Status.ReplicaCount = *&deployment.Status.ReadyReplicas
    if r.Client.Status().Update(ctx, &myKind); err != nil {
		log.Error(err, "failed to update status")
		return ctrl.Result{}, err
	}


	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MyAppResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1alpha1.MyAppResource{}).
		Complete(r)
}

var (
	controllerDeploymentOwnerKey = ".metadata.controller"
)

// cleanUpResources deletes any existing deployments and their resources for the given kind if they do not match current spec
func (r *MyAppResourceReconciler) cleanUpResources(ctx context.Context, log logr.Logger, myKind *appv1alpha1.MyAppResource) error {
    log.Info("searching and deleting existing deployments")
    serviceList := core.ServiceList{}
    if err := r.List(ctx, &serviceList, client.InNamespace(myKind.Namespace), client.MatchingFields{controllerDeploymentOwnerKey: myKind.Name}); err != nil {
        return err
    }

    deploymentList := kapps.DeploymentList{}
    if err := r.List(ctx, &deploymentList, client.InNamespace(myKind.Namespace), client.MatchingFields{controllerDeploymentOwnerKey: myKind.Name}); err != nil {
        return err
    }

    for _, deployment := range deploymentList.Items {
        if deployment.GetName() == myKind.GetName() {
            continue
        }

        if err := r.Client.Delete(ctx, &deployment); err != nil {
            log.Error(err, "failed to delete deployment")
            return err
        }
    }

    // Look for existing deployment, if redis is to be removed, remove the service.
    if strings.ToLower(myKind.Spec.Redis.Enabled) == "false" {
        for _, service := range serviceList.Items {
            if service.GetName() == myKind.GetName() {
                if err := r.Client.Delete(ctx, &service); err != nil { return err }
            }
        }

    }

    return nil
}

func buildDeploymentTemplate(myKind v1alpha1.MyAppResource) *kapps.Deployment {
    var redisContainer core.Container = core.Container{
        Name: fmt.Sprintf("%s-redis", myKind.GetName()),
        Image: "redis:latest",
        Ports: []core.ContainerPort{
            {
                ContainerPort: int32(6379),
                Name: "RedisPort",
            },
        },
    }

    deployment := kapps.Deployment{
        ObjectMeta: metav1.ObjectMeta{
            Name: myKind.GetName(),
            Namespace: myKind.GetNamespace(),
            OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(&myKind, appv1alpha1.GroupVersion.WithKind("MyAppResource"))},
        },
        Spec: kapps.DeploymentSpec{
            Replicas: myKind.Spec.ReplicaCount,
            Selector: &metav1.LabelSelector{
                MatchLabels: map[string]string{
                    "app.interviews.myrepo.io/deployment-name": myKind.GetName(),
                },
            },
            Template: core.PodTemplateSpec{
                ObjectMeta: metav1.ObjectMeta{
                    Labels: map[string]string{
                        "app.interviews.myrepo.io": myKind.GetName(),
                    },
                },
                Spec: core.PodSpec{
                    Containers: []core.Container{
                        {
                            Name: fmt.Sprintf("%s-podinfo", myKind.GetName()),
                            Image: fmt.Sprintf("%s:%s", myKind.Spec.Image.Repository, myKind.Spec.Image.Tag),
                            Env: []core.EnvVar{
                                {
                                    Name: "PODINFO_UI_COLOR",
                                    Value: myKind.Spec.UI.Color,
                                },
                                {
                                    Name: "PODINFO_UI_MESSAGE",
                                    Value: myKind.Spec.UI.Message,
                                },
                                {
                                    Name: "PODINFO_CACHE_SERVER",
                                    Value: "tcp://redis:6379",
                                },
                            },
                        },
                    },
                },
            },
        },
    }

    if strings.ToLower(myKind.Spec.Redis.Enabled) == "true" {
        deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers, redisContainer)
    }

    return &deployment
}

func buildRedisServiceTemplate(myKind v1alpha1.MyAppResource) *core.Service {
    service := core.Service{
        ObjectMeta: metav1.ObjectMeta{
            Name: myKind.GetName(),
        },
        Spec: core.ServiceSpec{
            Selector: map[string]string{
                "app.interviews.myrepo.io/deployment-name": myKind.GetName(),
            },
            Ports: []core.ServicePort{
                {
                    Protocol: "TCP",
                    Port: int32(6379),
                    TargetPort: intstr.FromString("RedisPort"),

                },
            },
        },
    }

    return &service
}
