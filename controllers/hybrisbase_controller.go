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
	"bytes"
	"context"
	"github.com/operator-framework/operator-lib/status"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	hybrisv1alpha1 "github.com/redhat-sap/sap-commerce-operator/api/v1alpha1"

	corev1 "k8s.io/api/core/v1"

	buildv1 "github.com/openshift/api/build/v1"
	imagev1 "github.com/openshift/api/image/v1"
)

const (
	DOCKER_FILE_REPO_URL = "https://github.com/xieshenzh/sap-commerce-operator"
	SAP_JDK_URL          = "https://github.com/SAP/SapMachine/releases/download/sapmachine-11.0.5/sapmachine-jdk-11.0.5-1.x86_64.rpm"
)

// HybrisBaseReconciler reconciles a HybrisBase object
type HybrisBaseReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=hybris.hybris.org,namespace="my-namespace",resources=hybrisbases;hybrisbases/finalizers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=hybris.hybris.org,namespace="my-namespace",resources=hybrisbases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=build.openshift.io,namespace="my-namespace",resources=buildconfigs;builds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=image.openshift.io,namespace="my-namespace",resources=imagestreams,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,namespace="my-namespace",resources=secrets,verbs=get;list;watch;create;update;patch;delete

func (r *HybrisBaseReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("hybrisbase", req.NamespacedName)

	// Fetch the HybrisBase instance
	hybrisBase := &hybrisv1alpha1.HybrisBase{}
	err := r.Get(ctx, req.NamespacedName, hybrisBase)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("HybrisBase resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get HybrisBase")
		return ctrl.Result{}, err
	}

	updateImageStream, err := r.ensureImageStream(hybrisBase, ctx, log)
	if err != nil {
		return ctrl.Result{}, err
	}
	updateSecret, err := r.ensureSecret(hybrisBase, ctx, log)
	if err != nil {
		return ctrl.Result{}, err
	}
	updateBuildConfig, err := r.ensureBuildConfig(hybrisBase, ctx, log)
	if err != nil {
		return ctrl.Result{}, err
	}
	if updateImageStream || updateSecret || updateBuildConfig {
		return ctrl.Result{Requeue: true}, nil
	}

	// List the builds for Hybirs base image
	buildList := &buildv1.BuildList{}
	listOpts := []client.ListOption{
		client.InNamespace(hybrisBase.Namespace),
		client.MatchingLabels(labelsForHybrisBase(hybrisBase.Name)),
	}
	if err = r.List(ctx, buildList, listOpts...); err != nil {
		log.Error(err, "Failed to list builds", "HybirsBase.Namespace", hybrisBase.Namespace, "HybirsBase.Name", hybrisBase.Name)
		return ctrl.Result{}, err
	}

	updateStatus := false
	for _, build := range buildList.Items {
		statusCondition := statusCondition(build.Status.Conditions)
		if conditions, ok := hybrisBase.Status.BuildConditions[build.Name]; ok {
			if !reflect.DeepEqual(conditions, statusCondition) {
				hybrisBase.Status.BuildConditions[build.Name] = statusCondition
				updateStatus = true
			}
		} else {
			hybrisBase.Status.BuildConditions[build.Name] = statusCondition
			updateStatus = true
		}
	}

	if updateStatus {
		err := r.Status().Update(ctx, hybrisBase)
		if err != nil {
			log.Error(err, "Failed to update HybrisBase status")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *HybrisBaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&hybrisv1alpha1.HybrisBase{}).
		Owns(&buildv1.BuildConfig{}).
		Owns(&imagev1.ImageStream{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func (r *HybrisBaseReconciler) ensureSecret(hybrisBase *hybrisv1alpha1.HybrisBase, ctx context.Context, log logr.Logger) (updated bool, err error) {
	// Check if the Secret already exists, if not create a new one
	found := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Name: hybrisBase.Name, Namespace: hybrisBase.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Define a new Secret
		secret := &corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Name:      hybrisBase.Name,
				Namespace: hybrisBase.Namespace,
				Labels:    labelsForHybrisBase(hybrisBase.Name),
			},
			StringData: map[string]string{
				"username": hybrisBase.Spec.Username,
				"password": hybrisBase.Spec.Password,
			},
		}
		// Set HybrisBase instance as the owner and controller
		ctrl.SetControllerReference(hybrisBase, secret, r.Scheme)

		log.Info("Creating a new Secret", "Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
		err = r.Create(ctx, secret)
		if err != nil {
			log.Error(err, "Failed to create new Secret", "Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
			return false, err
		}
		// Secret created successfully
		log.Info("Secret created", "Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
		return true, nil
	} else if err != nil {
		log.Error(err, "Failed to get Secret")
		return false, err
	}

	// Ensure the desired Secret data
	bytesUsername := []byte(hybrisBase.Spec.Username)
	bytesPassword := []byte(hybrisBase.Spec.Password)

	if !bytes.Equal(found.Data["username"], bytesUsername) || !bytes.Equal(found.Data["password"], bytesPassword) {
		found.StringData["username"] = hybrisBase.Spec.Username
		found.StringData["password"] = hybrisBase.Spec.Password
		err = r.Update(ctx, found)
		if err != nil {
			log.Error(err, "Failed to update Secret", "Secret.Namespace", found.Namespace, "Secret.Name", found.Name)
			return false, err
		}
		// Data updated - return and requeue
		log.Info("Secret updated", "Secret.Namespace", found.Namespace, "Secret.Name", found.Name)
		return true, nil
	}

	return false, nil
}

func (r *HybrisBaseReconciler) ensureImageStream(hybrisBase *hybrisv1alpha1.HybrisBase, ctx context.Context, log logr.Logger) (updated bool, err error) {
	// Check if the ImageStream already exists, if not create a new one
	name, tag := imageStreamNameTag(hybrisBase)

	found := &imagev1.ImageStream{}
	err = r.Get(ctx, types.NamespacedName{Name: name, Namespace: hybrisBase.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Define a new ImageStream
		is := &imagev1.ImageStream{
			ObjectMeta: v1.ObjectMeta{
				Name:      name,
				Namespace: hybrisBase.Namespace,
				Labels:    labelsForHybrisBase(hybrisBase.Name),
			},
			Spec: imagev1.ImageStreamSpec{
				Tags: []imagev1.TagReference{
					{
						Name: tag,
					},
				},
			},
		}
		// Set HybrisBase instance as the owner and controller
		ctrl.SetControllerReference(hybrisBase, is, r.Scheme)

		log.Info("Creating a new ImageStream", "ImageStream.Namespace", is.Namespace, "ImageStream.Name", is.Name)
		err = r.Create(ctx, is)
		if err != nil {
			log.Error(err, "Failed to create new ImageStream", "ImageStream.Namespace", is.Namespace, "ImageStream.Name", is.Name)
			return false, err
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

func (r *HybrisBaseReconciler) ensureBuildConfig(hybrisBase *hybrisv1alpha1.HybrisBase, ctx context.Context, log logr.Logger) (updated bool, err error) {
	// Check if the BuildConfig already exists, if not create a new one
	found := &buildv1.BuildConfig{}
	err = r.Get(ctx, types.NamespacedName{Name: hybrisBase.Name, Namespace: hybrisBase.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Define a new deployment
		bc := r.createBuildConfigForHybrisBase(hybrisBase)
		log.Info("Creating a new BuildConfig", "BuildConfig.Namespace", bc.Namespace, "BuildConfig.Name", bc.Name)
		err = r.Create(ctx, bc)
		if err != nil {
			log.Error(err, "Failed to create new BuildConfig", "BuildConfig.Namespace", bc.Namespace, "BuildConfig.Name", bc.Name)
			return false, err
		}
		// BuildConfig created successfully
		log.Info("BuildConfig created", "BuildConfig.Namespace", bc.Namespace, "BuildConfig.Name", bc.Name)
		return true, nil
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		return false, err
	}

	// Ensure the desired BuildConfig spec
	isName, isTag := imageStreamNameTag(hybrisBase)
	isNameTag := strings.Join([]string{isName, isTag}, ":")
	jdkURL := jdkURL(hybrisBase)

	foundJdkURL := envValue("SAP_JDK_URL", found.Spec.CommonSpec.Strategy.DockerStrategy.Env)
	foundURL := envValue("HYBRIS_URL", found.Spec.CommonSpec.Strategy.DockerStrategy.Env)

	if found.Spec.CommonSpec.Output.To.Name != isNameTag || foundJdkURL != jdkURL || foundURL != hybrisBase.Spec.URL {
		found.Spec.CommonSpec.Strategy.DockerStrategy.Env = replaceEnvValue("SAP_JDK_URL", jdkURL, found.Spec.CommonSpec.Strategy.DockerStrategy.Env)
		found.Spec.CommonSpec.Strategy.DockerStrategy.Env = replaceEnvValue("HYBRIS_URL", hybrisBase.Spec.URL, found.Spec.CommonSpec.Strategy.DockerStrategy.Env)
		found.Spec.CommonSpec.Output.To.Name = isNameTag
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

// createBuildConfigForHybrisBase returns a BuildConfig object for building the Hybris base image
func (r *HybrisBaseReconciler) createBuildConfigForHybrisBase(hybrisBase *hybrisv1alpha1.HybrisBase) *buildv1.BuildConfig {
	isName, isTag := imageStreamNameTag(hybrisBase)
	jdkURL := jdkURL(hybrisBase)

	bc := &buildv1.BuildConfig{
		ObjectMeta: v1.ObjectMeta{
			Name:      hybrisBase.Name,
			Namespace: hybrisBase.Namespace,
			Labels:    labelsForHybrisBase(hybrisBase.Name),
		},
		Spec: buildv1.BuildConfigSpec{
			CommonSpec: buildv1.CommonSpec{
				Output: buildv1.BuildOutput{
					To: &corev1.ObjectReference{
						Kind: "ImageStreamTag",
						Name: strings.Join([]string{isName, isTag}, ":"),
					},
				},
				Source: buildv1.BuildSource{
					Type: buildv1.BuildSourceGit,
					Git: &buildv1.GitBuildSource{
						URI: DOCKER_FILE_REPO_URL,
						Ref: "master",
					},
					ContextDir: "base",
				},
				Strategy: buildv1.BuildStrategy{
					Type: buildv1.DockerBuildStrategyType,
					DockerStrategy: &buildv1.DockerBuildStrategy{
						Env: []corev1.EnvVar{
							{
								Name:  "SAP_JDK_URL",
								Value: jdkURL,
							},
							{
								Name:  "HYBRIS_URL",
								Value: hybrisBase.Spec.URL,
							},
							{
								Name: "USERNAME",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: hybrisBase.Name,
										},
										Key: "username",
									},
								},
							},
							{
								Name: "PASSWORD",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: hybrisBase.Name,
										},
										Key: "password",
									},
								},
							},
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
	ctrl.SetControllerReference(hybrisBase, bc, r.Scheme)
	return bc
}

func imageStreamNameTag(hybrisBase *hybrisv1alpha1.HybrisBase) (string, string) {
	name := hybrisBase.Name
	if len(hybrisBase.Spec.ImageName) > 0 {
		name = hybrisBase.Spec.ImageName
	}
	tag := "latest"
	if len(hybrisBase.Spec.ImageTag) > 0 {
		tag = hybrisBase.Spec.ImageTag
	}
	return name, tag
}

func jdkURL(hybrisBase *hybrisv1alpha1.HybrisBase) string {
	jdkURL := SAP_JDK_URL
	if len(hybrisBase.Spec.JdkURL) > 0 {
		jdkURL = hybrisBase.Spec.JdkURL
	}
	return jdkURL
}

func envValue(name string, array []corev1.EnvVar) string {
	for _, env := range array {
		if env.Name == name {
			return env.Value
		}
	}
	return ""
}

func replaceEnvValue(name string, value string, array []corev1.EnvVar) []corev1.EnvVar {
	for _, env := range array {
		if env.Name == name {
			env.Value = value
			return array
		}
	}
	return append(array, corev1.EnvVar{
		Name:  name,
		Value: value,
	})
}

func statusCondition(buildConditions []buildv1.BuildCondition) status.Conditions {
	conditions := status.Conditions{}
	for _, buildCondition := range buildConditions {
		conditions = append(conditions, status.Condition{
			Type:               status.ConditionType(buildCondition.Type),
			Status:             buildCondition.Status,
			Reason:             status.ConditionReason(buildCondition.Reason),
			Message:            buildCondition.Message,
			LastTransitionTime: buildCondition.LastTransitionTime,
		})
	}
	return conditions
}

// labelsForHybrisBase returns the labels for selecting the resources
// belonging to the given HybrisBase CR name.
func labelsForHybrisBase(name string) map[string]string {
	return map[string]string{
		"app":           "hybrisBase",
		"hybrisBase_cr": name,
	}
}
