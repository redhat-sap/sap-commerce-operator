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
	BaseImageName string `json:"baseImageName,omitempty"`

	// Hybris base image tag
	// +kubebuilder:validation:Required
	BaseImageTag string `json:"baseImageTag,omitempty"`

	// Hybris app source repository URL
	// +kubebuilder:validation:Required
	SourceRepoURL string `json:"sourceRepoURL,omitempty"`

	// Hybris app source repository reference
	// +kubebuilder:validation:Required
	SourceRepoRef string `json:"sourceRepoRef,omitempty"`

	// Hybris app repository source location
	SourceRepoContext string `json:"sourceRepoContext,omitempty"`

	// Hybris app UDP port for multicast
	UDPPort *int32 `json:"UDPPort,omitempty"`
}

type DeploymentConfigStatusCondition struct {
	// Conditions of the deploymentConfig for the Hybris app
	Conditions []status.Condition `json:"conditions"`
}

type RouteStatusCondition struct {
	// Name of the route for the Hybris app
	RouteName string `json:"routeName"`

	// Host of the route for the Hybris app
	Host string `json:"host"`

	// Conditions of the route for the Hybris app
	Conditions []status.Condition `json:"conditions"`
}

// HybrisAppStatus defines the observed state of HybrisApp
type HybrisAppStatus struct {
	BuildConditions []BuildStatusCondition `json:"buildConditions"`

	DeploymentConfigConditions DeploymentConfigStatusCondition `json:"deploymentConfigConditions"`

	RouteConditions []RouteStatusCondition `json:"routeConditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// HybrisApp is the Schema for the hybrisapps API
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
