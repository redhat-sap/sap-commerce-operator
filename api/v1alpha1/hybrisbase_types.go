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

// HybrisBaseSpec defines the desired state of HybrisBase
type HybrisBaseSpec struct {
	// Hybris package download URL used to download the package and build the base image
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Hybris package download URL"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=1
	URL string `json:"URL,omitempty"`

	// SAP account username used to download the Hybris package
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="User Name"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=2
	Username string `json:"username,omitempty"`

	// SAP account password used to download the Hybris package
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Format:=password
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:password"}
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=3
	Password string `json:"password,omitempty"`

	// SAP Jdk download URL used to build the base image
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	JdkURL string `json:"jdkURL,omitempty"`

	// Name of the Hybris base image to be built
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	ImageName string `json:"imageName,omitempty"`

	// Tag of the Hybris base image to be built
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	ImageTag string `json:"imageTag,omitempty"`

	// Source Repo stores the s2i Dockerfile
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
	BuildSourceRepo string `json:"buildSourceRepo,omitempty"`

	// Source Repo branch stores the s2i Dockerfile
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
	BuildSourceRepoBranch string `json:"buildSourceRepoBranch,omitempty"`
}

type BuildStatusCondition struct {
	// Name of the build for the Hybris base image
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	BuildName string `json:"buildName"`

	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:io.kubernetes.conditions"}
	Conditions []status.Condition `json:"conditions"`
}

// HybrisBaseStatus defines the observed state of HybrisBase
type HybrisBaseStatus struct {
	BuildConditions []BuildStatusCondition `json:"buildConditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// HybrisBase is the Schema for the Hybris Base API
// +operator-sdk:csv:customresourcedefinitions:displayName="Hybris Base"
type HybrisBase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HybrisBaseSpec   `json:"spec,omitempty"`
	Status HybrisBaseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// HybrisBaseList contains a list of HybrisBase
type HybrisBaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HybrisBase `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HybrisBase{}, &HybrisBaseList{})
}
