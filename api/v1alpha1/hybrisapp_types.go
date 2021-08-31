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

package v1alpha1

import (
	"github.com/operator-framework/operator-lib/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HybrisAppSpec defines the desired state of HybrisApp
type HybrisAppSpec struct {
	// Hybris base image name
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	BaseImageName string `json:"baseImageName,omitempty"`

	// Hybris base image tag
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	BaseImageTag string `json:"baseImageTag,omitempty"`

	// Hybris app source repository URL
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	SourceRepoURL string `json:"sourceRepoURL,omitempty"`

	// Hybris app source repository reference
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Source Repo Ref or Branch Name"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	SourceRepoRef string `json:"sourceRepoRef,omitempty"`

	// Hybris app repository source location
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	SourceRepoContext string `json:"sourceRepoContext,omitempty"`

	// Hybris app repository local.properties override location
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	SourceRepoLocalPropertiesOverride string `json:"sourceRepoLocalPropertiesOverride,omitempty"`

	// Hybris app Apache Tomcat server.xml jvmRoute name
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Apache Tomcat jvmRoute name"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	ApachejvmRouteName string `json:"apachejvmRouteName,omitempty"`

	// Hybris app ANT tasks
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Specify ANT tasks, e.g. clean,compile,deploy"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	HybrisANTTaskNames string `json:"hybrisANTTaskNames,omitempty"`

	// Service Port for AJP End Point, range: 30000-32768
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Service Port for Apache Jserv Protocol End Point"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	AJPServicePort int32 `json:"aJPServicePort,omitempty"`

	// Pod Healthy Probe path for startup and readiness probe
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
	PodHealthyProbePath string `json:"podHealthyProbePath,omitempty"`

	// Period Second for Startup Probe
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
	StartupProbePeriodSecond int32 `json:"startupProbePeriodSecond,omitempty"`

	// Failure Threshold second for Startup Probe
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
	StartupProbeFailureThreshold int32 `json:"startupProbeFailureThreshold,omitempty"`
}

type DeploymentConfigStatusCondition struct {
	// Conditions of the deploymentConfig for the Hybris app
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:io.kubernetes.conditions"}
	Conditions []status.Condition `json:"conditions"`
}

type RouteStatusCondition struct {
	// Name of the route for the Hybris app
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	RouteName string `json:"routeName"`

	// Host of the route for the Hybris app
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	Host string `json:"host"`

	// Conditions of the route for the Hybris app
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:io.kubernetes.conditions"}
	Conditions []status.Condition `json:"conditions"`
}

// HybrisAppStatus defines the observed state of HybrisApp
type HybrisAppStatus struct {
	BuildConditions []BuildStatusCondition `json:"buildConditions"`

	DeploymentConfigConditions DeploymentConfigStatusCondition `json:"deploymentConfigConditions"`
	RouteConditions            []RouteStatusCondition          `json:"routeConditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// HybrisApp is the Schema for the hybrisapps API
//+operator-sdk:csv:customresourcedefinitions:displayName="Hybris Application"
type HybrisApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HybrisAppSpec   `json:"spec,omitempty"`
	Status HybrisAppStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// HybrisAppList contains a list of HybrisApp
type HybrisAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HybrisApp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HybrisApp{}, &HybrisAppList{})
}
