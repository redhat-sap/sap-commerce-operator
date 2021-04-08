/*


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
	"crypto/md5"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/go-logr/logr"

	"github.com/operator-framework/operator-lib/status"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	hybrisv1alpha1 "github.com/redhat-sap/sap-commerce-operator/api/v1alpha1"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	appsv1 "github.com/openshift/api/apps/v1"
	buildv1 "github.com/openshift/api/build/v1"
	imagev1 "github.com/openshift/api/image/v1"
	routev1 "github.com/openshift/api/route/v1"
)

const (
	licenceFile = "hybrislicence.jar"
	configFile  = "local.properties"

	licenceAnnotation = "licenceHash"
	configAnnotation  = "configHash"

	licenceVolume = "hybris-licence"
	configVolume  = "hybris-config"
)

// HybrisAppReconciler reconciles a HybrisApp object
type HybrisAppReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=hybris.hybris.org,namespace="my-namespace",resources=hybrisapps;hybrisapps/finalizers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=hybris.hybris.org,namespace="my-namespace",resources=hybrisapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=build.openshift.io,namespace="my-namespace",resources=buildconfigs;builds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=image.openshift.io,namespace="my-namespace",resources=imagestreams,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.openshift.io,namespace="my-namespace",resources=deploymentconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=route.openshift.io,namespace="my-namespace",resources=routes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,namespace="my-namespace",resources=services;configmaps;secrets,verbs=get;list;watch;create;update;patch;delete

func (r *HybrisAppReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("hybrisapp", req.NamespacedName)

	// Fetch the HybrisApp instance
	hybrisApp := &hybrisv1alpha1.HybrisApp{}
	err := r.Get(ctx, req.NamespacedName, hybrisApp)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("HybrisApp resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get HybrisApp")
		return ctrl.Result{}, err
	}

	updated, err := r.ensureImageStream(hybrisApp, ctx, log)
	if updated {
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}
	updated, err = r.ensureBuildConfig(hybrisApp, ctx, log)
	if updated {
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}

	if hybrisApp.Status.BuildConditions == nil {
		hybrisApp.Status.BuildConditions = []hybrisv1alpha1.BuildStatusCondition{}
	}
	if hybrisApp.Status.DeploymentConfigConditions.Conditions == nil {
		hybrisApp.Status.DeploymentConfigConditions.Conditions = []status.Condition{}
	}
	if hybrisApp.Status.RouteConditions == nil {
		hybrisApp.Status.RouteConditions = []hybrisv1alpha1.RouteStatusCondition{}
	}

	built, building, updated, err := r.updateBuildStatus(hybrisApp, ctx, log)
	if updated {
		err = r.Status().Update(ctx, hybrisApp)
		if err != nil {
			log.Error(err, "Failed to update HybrisApp status")
			return ctrl.Result{}, err
		}
	}

	if !built {
		log.Info("HybrisApp image is not available yet")
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	updated, err = r.ensureSecret(hybrisApp, ctx, log)
	if updated {
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}
	updated, err = r.ensureConfigMap(hybrisApp, ctx, log)
	if updated {
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}
	updated, err = r.ensureDeploymentConfig(hybrisApp, ctx, log)
	if updated {
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}
	updated, err = r.ensureService(hybrisApp, ctx, log)
	if updated {
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}
	updated, err = r.ensureRoute(hybrisApp, ctx, log)
	if updated {
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}

	updated, err = r.updateDeploymentConfigStatus(hybrisApp, ctx, log)
	if updated {
		err = r.Status().Update(ctx, hybrisApp)
		if err != nil {
			log.Error(err, "Failed to update HybrisApp status")
			return ctrl.Result{}, err
		}
	}
	updated, err = r.updateRouteStatus(hybrisApp, ctx, log)
	if updated {
		err = r.Status().Update(ctx, hybrisApp)
		if err != nil {
			log.Error(err, "Failed to update HybrisApp status")
			return ctrl.Result{}, err
		}
	}

	if updated {
		log.Info("HybrisApp status updated")
		return ctrl.Result{Requeue: true}, nil
	} else if building {
		log.Info("HybrisApp building in process")
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	return ctrl.Result{}, nil
}

func (r *HybrisAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&hybrisv1alpha1.HybrisApp{}).
		Owns(&buildv1.BuildConfig{}).
		Owns(&imagev1.ImageStream{}).
		Owns(&appsv1.DeploymentConfig{}).
		Owns(&corev1.Service{}).
		Owns(&routev1.Route{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func (r *HybrisAppReconciler) ensureImageStream(hybrisApp *hybrisv1alpha1.HybrisApp, ctx context.Context, log logr.Logger) (updated bool, err error) {
	// Check if the ImageStream already exists, if not create a new one
	found := &imagev1.ImageStream{}
	err = r.Get(ctx, types.NamespacedName{Name: hybrisApp.Name, Namespace: hybrisApp.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Define a new ImageStream
		is := &imagev1.ImageStream{
			ObjectMeta: v1.ObjectMeta{
				Name:      hybrisApp.Name,
				Namespace: hybrisApp.Namespace,
				Labels:    labelsForHybrisApp(hybrisApp.Name),
			},
			Spec: imagev1.ImageStreamSpec{
				Tags: []imagev1.TagReference{
					{
						Name: "latest",
					},
				},
			},
		}
		// Set HybrisApp instance as the owner and controller
		err := ctrl.SetControllerReference(hybrisApp, is, r.Scheme)
		if err != nil {
			log.Error(err, "Failed to set controller reference", "ImageStream.Namespace", is.Namespace, "ImageStream.Name", is.Name)
			return false, err
		}

		log.Info("Creating a new ImageStream", "ImageStream.Namespace", is.Namespace, "ImageStream.Name", is.Name)
		err = r.Create(ctx, is)
		if err != nil {
			if !errors.IsAlreadyExists(err) {
				log.Error(err, "Failed to create new ImageStream", "ImageStream.Namespace", is.Namespace, "ImageStream.Name", is.Name)
				return false, err
			} else {
				return true, nil
			}
		}
		// ImageStream created successfully
		log.Info("ImageStream created", "ImageStream.Namespace", is.Namespace, "ImageStream.Name", is.Name)
		return true, nil
	} else if err != nil {
		log.Error(err, "Failed to get ImageStream")
		return false, err
	}

	return false, nil
}

func (r *HybrisAppReconciler) ensureBuildConfig(hybrisApp *hybrisv1alpha1.HybrisApp, ctx context.Context, log logr.Logger) (updated bool, err error) {
	// Check if the BuildConfig already exists, if not create a new one
	found := &buildv1.BuildConfig{}
	err = r.Get(ctx, types.NamespacedName{Name: hybrisApp.Name, Namespace: hybrisApp.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Define a new deployment
		bc := r.createBuildConfigForHybrisApp(hybrisApp)
		//log.Info("----> BuildConfig", "bc.Spec.Strategy.SourceStrategy.Env", bc.Spec.Strategy.SourceStrategy.Env)
		// Set HybrisApp instance as the owner and controller
		err = ctrl.SetControllerReference(hybrisApp, bc, r.Scheme)
		if err != nil {
			log.Error(err, "Failed to set controller reference", "BuildConfig.Namespace", bc.Namespace, "BuildConfig.Name", bc.Name)
			return false, err
		}

		log.Info("Creating a new BuildConfig", "BuildConfig.Namespace", bc.Namespace, "BuildConfig.Name", bc.Name)
		err = r.Create(ctx, bc)
		if err != nil {
			if !errors.IsAlreadyExists(err) {
				log.Error(err, "Failed to create new BuildConfig", "BuildConfig.Namespace", bc.Namespace, "BuildConfig.Name", bc.Name)
				return false, err
			} else {
				return true, nil
			}
		}
		// BuildConfig created successfully
		log.Info("BuildConfig created", "BuildConfig.Namespace", bc.Namespace, "BuildConfig.Name", bc.Name)
		return true, nil
	} else if err != nil {
		log.Error(err, "Failed to get BuildConfig")
		return false, err
	}

	// Ensure the desired BuildConfig spec
	isNameTag := strings.Join([]string{hybrisApp.Spec.BaseImageName, hybrisApp.Spec.BaseImageTag}, ":")

	if found.Spec.CommonSpec.Source.ContextDir != hybrisApp.Spec.SourceRepoContext ||
		found.Spec.CommonSpec.Source.Git.URI != hybrisApp.Spec.SourceRepoURL ||
		found.Spec.CommonSpec.Source.Git.Ref != hybrisApp.Spec.SourceRepoRef ||
		found.Spec.CommonSpec.Strategy.SourceStrategy.From.Name != isNameTag {
		found.Spec.CommonSpec.Source.ContextDir = hybrisApp.Spec.SourceRepoContext
		found.Spec.CommonSpec.Source.Git.URI = hybrisApp.Spec.SourceRepoURL
		found.Spec.CommonSpec.Source.Git.Ref = hybrisApp.Spec.SourceRepoRef
		found.Spec.CommonSpec.Strategy.SourceStrategy.From.Name = isNameTag
		err = r.Update(ctx, found)
		if err != nil {
			log.Error(err, "Failed to update BuildConfig", "BuildConfig.Namespace", found.Namespace, "BuildConfig.Name", found.Name)
			return false, err
		}
		// Spec updated - return and requeue
		log.Info("BuildConfig updated", "BuildConfig.Namespace", found.Namespace, "BuildConfig.Name", found.Name)
		return true, nil
	}

	return false, nil
}

// createBuildConfigForHybrisApp returns a BuildConfig object for building the Hybris app image
func (r *HybrisAppReconciler) createBuildConfigForHybrisApp(hybrisApp *hybrisv1alpha1.HybrisApp) *buildv1.BuildConfig {
	bc := &buildv1.BuildConfig{
		ObjectMeta: v1.ObjectMeta{
			Name:      hybrisApp.Name,
			Namespace: hybrisApp.Namespace,
			Labels:    labelsForHybrisApp(hybrisApp.Name),
		},
		Spec: buildv1.BuildConfigSpec{
			CommonSpec: buildv1.CommonSpec{
				Output: buildv1.BuildOutput{
					To: &corev1.ObjectReference{
						Kind: "ImageStreamTag",
						Name: strings.Join([]string{hybrisApp.Name, "latest"}, ":"),
					},
				},
				Resources: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("2"),
						corev1.ResourceMemory: resource.MustParse("4Gi"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("1"),
						corev1.ResourceMemory: resource.MustParse("1Gi"),
					},
				},
				Source: buildv1.BuildSource{
					Type: buildv1.BuildSourceGit,
					Git: &buildv1.GitBuildSource{
						URI: hybrisApp.Spec.SourceRepoURL,
						Ref: hybrisApp.Spec.SourceRepoRef,
					},
				},
				Strategy: buildv1.BuildStrategy{
					Type: buildv1.SourceBuildStrategyType,
					SourceStrategy: &buildv1.SourceBuildStrategy{
						From: corev1.ObjectReference{Kind: "ImageStreamTag", Name: strings.Join([]string{hybrisApp.Spec.BaseImageName, hybrisApp.Spec.BaseImageTag}, ":")},
						Env: []corev1.EnvVar{
							{Name: "SOURCE_REPO_LOCAL_PROPERTIES_OVERRIDE", Value: hybrisApp.Spec.SourceRepoLocalPorpertiesOverride},
							{Name: "APACHE_JVM_ROUTE_NAME", Value: hybrisApp.Spec.ApachejvmRouteName},
							{Name: "HYBRIS_ANT_TASK_NAMES", Value: hybrisApp.Spec.HybrisANTTaskNames},
						},
					},
				},
			},
			Triggers: []buildv1.BuildTriggerPolicy{
				{
					Type: buildv1.ConfigChangeBuildTriggerType,
				},
				{
					Type: buildv1.ImageChangeBuildTriggerType,
				},
			},
		},
	}
	if len(hybrisApp.Spec.SourceRepoContext) > 0 {
		bc.Spec.CommonSpec.Source.ContextDir = hybrisApp.Spec.SourceRepoContext
	}

	return bc
}

func (r *HybrisAppReconciler) ensureSecret(hybrisApp *hybrisv1alpha1.HybrisApp, ctx context.Context, log logr.Logger) (updated bool, err error) {
	// Check if the Secret already exists, if not create a new one
	found := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Name: hybrisApp.Name, Namespace: hybrisApp.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Define a new Secret
		secret := &corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Name:      hybrisApp.Name,
				Namespace: hybrisApp.Namespace,
				Labels:    labelsForHybrisApp(hybrisApp.Name),
			},
		}
		// Set HybrisApp instance as the owner and controller
		err = ctrl.SetControllerReference(hybrisApp, secret, r.Scheme)
		if err != nil {
			log.Error(err, "Failed to set controller reference", "Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
			return false, err
		}

		log.Info("Creating a new Secret", "Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
		err = r.Create(ctx, secret)
		if err != nil {
			if !errors.IsAlreadyExists(err) {
				log.Error(err, "Failed to create new Secret", "Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
				return false, err
			} else {
				return true, nil
			}
		}
		// Secret created successfully
		log.Info("Secret created", "Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
		return true, nil
	} else if err != nil {
		log.Error(err, "Failed to get Secret")
		return false, err
	}

	return false, nil
}

func (r *HybrisAppReconciler) ensureConfigMap(hybrisApp *hybrisv1alpha1.HybrisApp, ctx context.Context, log logr.Logger) (updated bool, err error) {
	// Check if the ConfigMap already exists, if not create a new one
	found := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{Name: hybrisApp.Name, Namespace: hybrisApp.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Define a new ConfigMap
		cm := &corev1.ConfigMap{
			ObjectMeta: v1.ObjectMeta{
				Name:      hybrisApp.Name,
				Namespace: hybrisApp.Namespace,
				Labels:    labelsForHybrisApp(hybrisApp.Name),
			},
		}
		// Set HybrisApp instance as the owner and controller
		err = ctrl.SetControllerReference(hybrisApp, cm, r.Scheme)
		if err != nil {
			log.Error(err, "Failed to set controller reference", "ConfigMap.Namespace", cm.Namespace, "ConfigMap.Name", cm.Name)
			return false, err
		}

		log.Info("Creating a new ConfigMap", "ConfigMap.Namespace", cm.Namespace, "ConfigMap.Name", cm.Name)
		err = r.Create(ctx, cm)
		if err != nil {
			if !errors.IsAlreadyExists(err) {
				log.Error(err, "Failed to create new ConfigMap", "ConfigMap.Namespace", cm.Namespace, "ConfigMap.Name", cm.Name)
				return false, err
			} else {
				return true, nil
			}
		}
		// ConfigMap created successfully
		log.Info("ConfigMap created", "ConfigMap.Namespace", cm.Namespace, "ConfigMap.Name", cm.Name)
		return true, nil
	} else if err != nil {
		log.Error(err, "Failed to get ConfigMap")
		return false, err
	}

	return false, nil
}

func (r *HybrisAppReconciler) ensureService(hybrisApp *hybrisv1alpha1.HybrisApp, ctx context.Context, log logr.Logger) (updated bool, err error) {
	// Check if the Service already exists, if not create a new one
	found := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: hybrisApp.Name, Namespace: hybrisApp.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Define a new Service
		service := &corev1.Service{
			ObjectMeta: v1.ObjectMeta{
				Name:      hybrisApp.Name,
				Namespace: hybrisApp.Namespace,
				Labels:    labelsForHybrisApp(hybrisApp.Name),
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{
						Name:       "9001-tcp",
						Port:       9001,
						TargetPort: intstr.FromInt(9001),
						Protocol:   corev1.ProtocolTCP,
					},
					{
						Name:       "9002-tcp",
						Port:       9002,
						TargetPort: intstr.FromInt(9002),
						Protocol:   corev1.ProtocolTCP,
					},
				},
				Selector: labelsForHybrisApp(hybrisApp.Name),
				Type:     corev1.ServiceTypeClusterIP,
			},
		}
		// Set HybrisApp instance as the owner and controller
		err = ctrl.SetControllerReference(hybrisApp, service, r.Scheme)
		if err != nil {
			log.Error(err, "Failed to set controller reference", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
			return false, err
		}

		log.Info("Creating a new Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
		err = r.Create(ctx, service)
		if err != nil {
			if !errors.IsAlreadyExists(err) {
				log.Error(err, "Failed to create new Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
				return false, err
			} else {
				return true, nil
			}
		}
		// Service created successfully
		log.Info("Service created", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
		return true, nil
	} else if err != nil {
		log.Error(err, "Failed to get Service")
		return false, err
	}

	return false, nil
}

func (r *HybrisAppReconciler) ensureRoute(hybrisApp *hybrisv1alpha1.HybrisApp, ctx context.Context, log logr.Logger) (updated bool, err error) {
	// Check if the Route already exists, if not create a new one
	found := &routev1.Route{}
	err = r.Get(ctx, types.NamespacedName{Name: hybrisApp.Name, Namespace: hybrisApp.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Define a new Route
		route := &routev1.Route{
			ObjectMeta: v1.ObjectMeta{
				Name:      hybrisApp.Name,
				Namespace: hybrisApp.Namespace,
				Labels:    labelsForHybrisApp(hybrisApp.Name),
			},
			Spec: routev1.RouteSpec{
				Host: "",
				Port: &routev1.RoutePort{
					TargetPort: intstr.FromString("9002-tcp"),
				},
				TLS: &routev1.TLSConfig{
					Termination: routev1.TLSTerminationPassthrough,
				},
				To: routev1.RouteTargetReference{
					Kind: "Service",
					Name: hybrisApp.Name,
				},
			},
		}
		// Set HybrisApp instance as the owner and controller
		err = ctrl.SetControllerReference(hybrisApp, route, r.Scheme)
		if err != nil {
			log.Error(err, "Failed to set controller reference", "Route.Namespace", route.Namespace, "Route.Name", route.Name)
			return false, err
		}

		log.Info("Creating a new Route", "Route.Namespace", route.Namespace, "Route.Name", route.Name)
		err = r.Create(ctx, route)
		if err != nil {
			if !errors.IsAlreadyExists(err) {
				log.Error(err, "Failed to create new Route", "Route.Namespace", route.Namespace, "Route.Name", route.Name)
				return false, err
			} else {
				return true, nil
			}
		}
		// Route created successfully
		log.Info("Route created", "Route.Namespace", route.Namespace, "Route.Name", route.Name)
		return true, nil
	} else if err != nil {
		log.Error(err, "Failed to get Route")
		return false, err
	}

	return false, nil
}

func (r *HybrisAppReconciler) ensureDeploymentConfig(hybrisApp *hybrisv1alpha1.HybrisApp, ctx context.Context, log logr.Logger) (updated bool, err error) {
	foundSecret := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Name: hybrisApp.Name, Namespace: hybrisApp.Namespace}, foundSecret)
	if err != nil {
		log.Error(err, "Failed to get Secret")
		return false, err
	}

	foundCM := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{Name: hybrisApp.Name, Namespace: hybrisApp.Namespace}, foundCM)
	if err != nil {
		log.Error(err, "Failed to get ConfigMap")
		return false, err
	}

	// Check if the DeploymentConfig already exists, if not create a new one
	found := &appsv1.DeploymentConfig{}
	err = r.Get(ctx, types.NamespacedName{Name: hybrisApp.Name, Namespace: hybrisApp.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Define a new DeploymentConfig
		dc := r.createDeploymentConfigForHybrisApp(hybrisApp, foundSecret, foundCM)
		// Set HybrisApp instance as the owner and controller
		err = ctrl.SetControllerReference(hybrisApp, dc, r.Scheme)
		if err != nil {
			log.Error(err, "Failed to set controller reference", "DeploymentConfig.Namespace", dc.Namespace, "DeploymentConfig.Name", dc.Name)
			return false, err
		}

		log.Info("Creating a new DeploymentConfig", "DeploymentConfig.Namespace", dc.Namespace, "DeploymentConfig.Name", dc.Name)
		err = r.Create(ctx, dc)
		if err != nil {
			if !errors.IsAlreadyExists(err) {
				log.Error(err, "Failed to create new DeploymentConfig", "DeploymentConfig.Namespace", dc.Namespace, "DeploymentConfig.Name", dc.Name)
				return false, err
			} else {
				return true, nil
			}
		}
		// DeploymentConfig created successfully
		log.Info("DeploymentConfig created", "DeploymentConfig.Namespace", dc.Namespace, "DeploymentConfig.Name", dc.Name)
		return true, nil
	} else if err != nil {
		log.Error(err, "Failed to get DeploymentConfig")
		return false, err
	}

	if ensureDeploymentConfigVolumes(hybrisApp, found, foundSecret, foundCM) {
		err = r.Update(ctx, found)
		if err != nil {
			log.Error(err, "Failed to update DeploymentConfig", "DeploymentConfig.Namespace", found.Namespace, "DeploymentConfig.Name", found.Name)
			return false, err
		}
		// Spec updated - return and requeue
		log.Info("DeploymentConfig updated", "DeploymentConfig.Namespace", found.Namespace, "DeploymentConfig.Name", found.Name)
		return true, nil
	}

	return false, nil
}

// createDeploymentConfigForHybrisApp returns a BuildConfig object for building the Hybris app image
func (r *HybrisAppReconciler) createDeploymentConfigForHybrisApp(hybrisApp *hybrisv1alpha1.HybrisApp, secret *corev1.Secret, cm *corev1.ConfigMap) *appsv1.DeploymentConfig {
	activeDeadlineSeconds := int64(21600)
	intervalSeconds := int64(1)
	maxSurge := intstr.FromString("25%")
	maxUnavailable := intstr.FromString("25%")
	timeoutSeconds := int64(600)
	updatePeriodSeconds := int64(1)
	terminationGracePeriodSeconds := int64(30)

	dc := &appsv1.DeploymentConfig{
		ObjectMeta: v1.ObjectMeta{
			Name:      hybrisApp.Name,
			Namespace: hybrisApp.Namespace,
			Labels:    labelsForHybrisApp(hybrisApp.Name),
		},
		Spec: appsv1.DeploymentConfigSpec{
			Replicas: 1,
			Selector: labelsForHybrisApp(hybrisApp.Name),
			Strategy: appsv1.DeploymentStrategy{
				ActiveDeadlineSeconds: &activeDeadlineSeconds,
				Resources:             corev1.ResourceRequirements{},
				RollingParams: &appsv1.RollingDeploymentStrategyParams{
					IntervalSeconds:     &intervalSeconds,
					MaxSurge:            &maxSurge,
					MaxUnavailable:      &maxUnavailable,
					TimeoutSeconds:      &timeoutSeconds,
					UpdatePeriodSeconds: &updatePeriodSeconds,
				},
				Type: appsv1.DeploymentStrategyTypeRolling,
			},
			Template: &corev1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: labelsForHybrisApp(hybrisApp.Name),
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "",
							Name:  hybrisApp.Name,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 9001,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									ContainerPort: 9002,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("2"),
									corev1.ResourceMemory: resource.MustParse("6Gi"),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("1"),
									corev1.ResourceMemory: resource.MustParse("1Gi"),
								},
							},
							TerminationMessagePath:   "/dev/termination-log",
							TerminationMessagePolicy: corev1.TerminationMessageReadFile,
						},
					},
					SecurityContext:               &corev1.PodSecurityContext{},
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
				},
			},
			Triggers: appsv1.DeploymentTriggerPolicies{
				{
					Type: appsv1.DeploymentTriggerOnConfigChange,
				},
				{
					Type: appsv1.DeploymentTriggerOnImageChange,
					ImageChangeParams: &appsv1.DeploymentTriggerImageChangeParams{
						Automatic: true,
						ContainerNames: []string{
							hybrisApp.Name,
						},
						From: corev1.ObjectReference{
							Kind: "ImageStreamTag",
							Name: strings.Join([]string{hybrisApp.Name, "latest"}, ":"),
						},
					},
				},
			},
		},
	}
	_ = ensureDeploymentConfigVolumes(hybrisApp, dc, secret, cm)
	return dc
}

func ensureDeploymentConfigVolumes(hybrisApp *hybrisv1alpha1.HybrisApp, dc *appsv1.DeploymentConfig, secret *corev1.Secret, cm *corev1.ConfigMap) bool {
	if dc.Spec.Template.Annotations == nil {
		dc.Spec.Template.Annotations = map[string]string{}
	}

	var licenceHash string
	var licenceExists bool
	var updateLicence bool
	if secret != nil && secret.Data != nil {
		licence, ok := secret.Data[licenceFile]
		if ok {
			licenceExists = true
			licenceHash = fmt.Sprintf("%x", md5.Sum(licence))
			if anno, ok := dc.Spec.Template.Annotations[licenceAnnotation]; !ok || anno != licenceHash {
				updateLicence = true
			}
		}
	}

	var configHash string
	var configExists bool
	var updateConfig bool
	if cm != nil && cm.Data != nil {
		config, ok := cm.Data[configFile]
		if ok {
			configExists = true
			configHash = fmt.Sprintf("%x", md5.Sum([]byte(config)))
			if anno, ok := dc.Spec.Template.Annotations[configAnnotation]; !ok || anno != configHash {
				updateConfig = true
			}
		}
	}

	if updateLicence || !licenceExists {
		for i, v := range dc.Spec.Template.Spec.Volumes {
			if v.Name == licenceVolume {
				dc.Spec.Template.Spec.Volumes[i] = dc.Spec.Template.Spec.Volumes[len(dc.Spec.Template.Spec.Volumes)-1]
				dc.Spec.Template.Spec.Volumes = dc.Spec.Template.Spec.Volumes[:len(dc.Spec.Template.Spec.Volumes)-1]
				break
			}
		}
		for i, v := range dc.Spec.Template.Spec.Containers[0].VolumeMounts {
			if v.Name == licenceVolume {
				dc.Spec.Template.Spec.Containers[0].VolumeMounts[i] = dc.Spec.Template.Spec.Containers[0].VolumeMounts[len(dc.Spec.Template.Spec.Containers[0].VolumeMounts)-1]
				dc.Spec.Template.Spec.Containers[0].VolumeMounts = dc.Spec.Template.Spec.Containers[0].VolumeMounts[:len(dc.Spec.Template.Spec.Containers[0].VolumeMounts)-1]
				break
			}
		}
	}

	if updateConfig || !configExists {
		for i, v := range dc.Spec.Template.Spec.Volumes {
			if v.Name == configVolume {
				dc.Spec.Template.Spec.Volumes[i] = dc.Spec.Template.Spec.Volumes[len(dc.Spec.Template.Spec.Volumes)-1]
				dc.Spec.Template.Spec.Volumes = dc.Spec.Template.Spec.Volumes[:len(dc.Spec.Template.Spec.Volumes)-1]
				break
			}
		}
		for i, v := range dc.Spec.Template.Spec.Containers[0].VolumeMounts {
			if v.Name == configVolume {
				dc.Spec.Template.Spec.Containers[0].VolumeMounts[i] = dc.Spec.Template.Spec.Containers[0].VolumeMounts[len(dc.Spec.Template.Spec.Containers[0].VolumeMounts)-1]
				dc.Spec.Template.Spec.Containers[0].VolumeMounts = dc.Spec.Template.Spec.Containers[0].VolumeMounts[:len(dc.Spec.Template.Spec.Containers[0].VolumeMounts)-1]
				break
			}
		}
	}

	defaultMode := int32(420)

	if updateLicence {
		dc.Spec.Template.Annotations[licenceAnnotation] = licenceHash

		dc.Spec.Template.Spec.Volumes = append(dc.Spec.Template.Spec.Volumes,
			corev1.Volume{
				Name: licenceVolume,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName:  hybrisApp.Name,
						DefaultMode: &defaultMode,
						Items: []corev1.KeyToPath{
							{
								Key:  licenceFile,
								Path: licenceFile,
							},
						},
					},
				},
			})

		dc.Spec.Template.Spec.Containers[0].VolumeMounts = append(dc.Spec.Template.Spec.Containers[0].VolumeMounts,
			corev1.VolumeMount{
				Name:      licenceVolume,
				MountPath: "/etc/hybris/licence",
				ReadOnly:  true,
			})
	} else if !licenceExists {
		updateLicence = len(dc.Spec.Template.Annotations[licenceAnnotation]) > 0
		if updateLicence {
			dc.Spec.Template.Annotations[licenceAnnotation] = ""
		}
	}

	if updateConfig {
		dc.Spec.Template.Annotations[configAnnotation] = configHash

		dc.Spec.Template.Spec.Volumes = append(dc.Spec.Template.Spec.Volumes,
			corev1.Volume{
				Name: configVolume,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: hybrisApp.Name,
						},
						DefaultMode: &defaultMode,
						Items: []corev1.KeyToPath{
							{
								Key:  configFile,
								Path: configFile,
							},
						},
					},
				},
			})

		dc.Spec.Template.Spec.Containers[0].VolumeMounts = append(dc.Spec.Template.Spec.Containers[0].VolumeMounts,
			corev1.VolumeMount{
				Name:      configVolume,
				MountPath: "/etc/hybris/config",
			})
	} else if !configExists {
		updateConfig = len(dc.Spec.Template.Annotations[configAnnotation]) > 0
		if updateConfig {
			dc.Spec.Template.Annotations[configAnnotation] = ""
		}
	}

	return updateLicence || updateConfig
}

func (r *HybrisAppReconciler) updateBuildStatus(hybrisApp *hybrisv1alpha1.HybrisApp, ctx context.Context, log logr.Logger) (built bool, building bool, updated bool, err error) {
	// List the builds for Hybirs app image
	buildList := &buildv1.BuildList{}
	listOpts := []client.ListOption{
		client.InNamespace(hybrisApp.Namespace),
		client.MatchingLabels(labelsForHybrisApp(hybrisApp.Name)),
	}
	if err := r.List(ctx, buildList, listOpts...); err != nil {
		log.Error(err, "Failed to list builds", "HybirsApp.Namespace", hybrisApp.Namespace, "HybirsApp.Name", hybrisApp.Name)
		return false, false, false, err
	}

	statusConditions := []hybrisv1alpha1.BuildStatusCondition{}
	if len(buildList.Items) > 0 {
		for _, build := range buildList.Items {
			statusCondition := buildStatusCondition(build.Name, build.Status.Conditions)
			sort.SliceStable(statusCondition.Conditions, func(i, j int) bool {
				return statusCondition.Conditions[i].LastTransitionTime.Before(&statusCondition.Conditions[j].LastTransitionTime)
			})

			if build.Status.Phase == buildv1.BuildPhaseComplete {
				built = true
			}
			if build.Status.Phase == buildv1.BuildPhaseNew ||
				build.Status.Phase == buildv1.BuildPhasePending ||
				build.Status.Phase == buildv1.BuildPhaseRunning {
				building = true
			}

			statusConditions = append(statusConditions, *statusCondition)
		}
	}

	if !reflect.DeepEqual(hybrisApp.Status.BuildConditions, statusConditions) {
		hybrisApp.Status.BuildConditions = statusConditions
		return built, building, true, nil
	}
	return built, building, false, nil
}

func (r *HybrisAppReconciler) updateDeploymentConfigStatus(hybrisApp *hybrisv1alpha1.HybrisApp, ctx context.Context, log logr.Logger) (updated bool, err error) {
	// List the builds for Hybirs app image
	dc := &appsv1.DeploymentConfig{}
	key := client.ObjectKey{
		Name:      hybrisApp.Name,
		Namespace: hybrisApp.Namespace,
	}
	if err := r.Get(ctx, key, dc); err != nil {
		log.Error(err, "Failed to get deploymentConfig", "HybirsApp.Namespace", hybrisApp.Namespace, "HybirsApp.Name", hybrisApp.Name)
		return false, err
	}

	statusCondition := deploymentConfigStatusCondition(dc.Status.Conditions)
	sort.SliceStable(statusCondition.Conditions, func(i, j int) bool {
		return statusCondition.Conditions[i].LastTransitionTime.Before(&statusCondition.Conditions[j].LastTransitionTime)
	})

	if !reflect.DeepEqual(hybrisApp.Status.DeploymentConfigConditions, *statusCondition) {
		hybrisApp.Status.DeploymentConfigConditions = *statusCondition
		return true, nil
	}
	return false, nil
}

func deploymentConfigStatusCondition(dcConditions []appsv1.DeploymentCondition) *hybrisv1alpha1.DeploymentConfigStatusCondition {
	conditions := []status.Condition{}
	for _, dcCondition := range dcConditions {
		conditions = append(conditions, status.Condition{
			Type:               status.ConditionType(dcCondition.Type),
			Status:             dcCondition.Status,
			Reason:             status.ConditionReason(dcCondition.Reason),
			Message:            dcCondition.Message,
			LastTransitionTime: dcCondition.LastTransitionTime,
		})
	}
	return &hybrisv1alpha1.DeploymentConfigStatusCondition{
		Conditions: conditions,
	}
}

func (r *HybrisAppReconciler) updateRouteStatus(hybrisApp *hybrisv1alpha1.HybrisApp, ctx context.Context, log logr.Logger) (updated bool, err error) {
	// List the builds for Hybirs app image
	route := &routev1.Route{}
	key := client.ObjectKey{
		Name:      hybrisApp.Name,
		Namespace: hybrisApp.Namespace,
	}
	if err := r.Get(ctx, key, route); err != nil {
		log.Error(err, "Failed to get route", "HybirsApp.Namespace", hybrisApp.Namespace, "HybirsApp.Name", hybrisApp.Name)
		return false, err
	}

	statusConditions := []hybrisv1alpha1.RouteStatusCondition{}
	if len(route.Status.Ingress) > 0 {
		for _, routeStatus := range route.Status.Ingress {
			statusCondition := routeStatusCondition(routeStatus.RouterName, routeStatus.Host, routeStatus.Conditions)
			sort.SliceStable(statusCondition.Conditions, func(i, j int) bool {
				return statusCondition.Conditions[i].LastTransitionTime.Before(&statusCondition.Conditions[j].LastTransitionTime)
			})

			statusConditions = append(statusConditions, *statusCondition)
		}
	}

	if !reflect.DeepEqual(hybrisApp.Status.RouteConditions, statusConditions) {
		hybrisApp.Status.RouteConditions = statusConditions
		return true, nil
	}
	return false, nil
}

func routeStatusCondition(name string, host string, routeConditions []routev1.RouteIngressCondition) *hybrisv1alpha1.RouteStatusCondition {
	var conditions []status.Condition
	for _, routeCondition := range routeConditions {
		condition := status.Condition{
			Type:    status.ConditionType(routeCondition.Type),
			Status:  routeCondition.Status,
			Reason:  status.ConditionReason(routeCondition.Reason),
			Message: routeCondition.Message,
		}
		if routeCondition.LastTransitionTime != nil {
			condition.LastTransitionTime = *routeCondition.LastTransitionTime
		}
		conditions = append(conditions, condition)
	}
	return &hybrisv1alpha1.RouteStatusCondition{
		RouteName:  name,
		Host:       host,
		Conditions: conditions,
	}
}

// labelsForHybrisApp returns the labels for selecting the resources
// belonging to the given HybrisApp CR name.
func labelsForHybrisApp(name string) map[string]string {
	return map[string]string{
		"app":          "hybrisApp",
		"hybrisApp_cr": name,
	}
}
