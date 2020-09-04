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
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	hybrisv1alpha1 "github.com/redhat-sap/sap-commerce-operator/api/v1alpha1"

	corev1 "k8s.io/api/core/v1"

	appsv1 "github.com/openshift/api/apps/v1"
	buildv1 "github.com/openshift/api/build/v1"
	imagev1 "github.com/openshift/api/image/v1"
	routev1 "github.com/openshift/api/route/v1"
)

// HybrisAppReconciler reconciles a HybrisApp object
type HybrisAppReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=hybris.hybris.org,namespace="my-namespace",resources=hybrisapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=hybris.hybris.org,namespace="my-namespace",resources=hybrisapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=build.openshift.io,namespace="my-namespace",resources=buildconfigs;builds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=image.openshift.io,namespace="my-namespace",resources=imagestreams,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.openshift.io,namespace="my-namespace",resources=deploymentconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=route.openshift.io,namespace="my-namespace",resources=routes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,namespace="my-namespace",resources=services;configmaps,verbs=get;list;watch;create;update;patch;delete

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
		// Set HybrisBase instance as the owner and controller
		ctrl.SetControllerReference(hybrisApp, is, r.Scheme)

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
				Source: buildv1.BuildSource{
					Type: buildv1.BuildSourceGit,
					Git: &buildv1.GitBuildSource{
						URI: hybrisApp.Spec.SourceRepoURL,
						Ref: hybrisApp.Spec.SourceRepoRef,
					},
					ContextDir: hybrisApp.Spec.SourceRepoContext,
				},
				Strategy: buildv1.BuildStrategy{
					Type: buildv1.SourceBuildStrategyType,
					SourceStrategy: &buildv1.SourceBuildStrategy{
						From: corev1.ObjectReference{
							Kind: "ImageStreamTag",
							Name: strings.Join([]string{hybrisApp.Spec.BaseImageName, hybrisApp.Spec.BaseImageTag}, ":"),
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
	// Set HybrisBase instance as the owner and controller
	ctrl.SetControllerReference(hybrisApp, bc, r.Scheme)
	return bc
}

// labelsForHybrisApp returns the labels for selecting the resources
// belonging to the given HybrisApp CR name.
func labelsForHybrisApp(name string) map[string]string {
	return map[string]string{
		"app":           "hybrisApp",
		"hybrisBase_cr": name,
	}
}
