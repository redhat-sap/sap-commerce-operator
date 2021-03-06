// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"github.com/operator-framework/operator-lib/status"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BuildStatusCondition) DeepCopyInto(out *BuildStatusCondition) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]status.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BuildStatusCondition.
func (in *BuildStatusCondition) DeepCopy() *BuildStatusCondition {
	if in == nil {
		return nil
	}
	out := new(BuildStatusCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DeploymentConfigStatusCondition) DeepCopyInto(out *DeploymentConfigStatusCondition) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]status.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DeploymentConfigStatusCondition.
func (in *DeploymentConfigStatusCondition) DeepCopy() *DeploymentConfigStatusCondition {
	if in == nil {
		return nil
	}
	out := new(DeploymentConfigStatusCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HybrisApp) DeepCopyInto(out *HybrisApp) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HybrisApp.
func (in *HybrisApp) DeepCopy() *HybrisApp {
	if in == nil {
		return nil
	}
	out := new(HybrisApp)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *HybrisApp) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HybrisAppList) DeepCopyInto(out *HybrisAppList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]HybrisApp, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HybrisAppList.
func (in *HybrisAppList) DeepCopy() *HybrisAppList {
	if in == nil {
		return nil
	}
	out := new(HybrisAppList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *HybrisAppList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HybrisAppSpec) DeepCopyInto(out *HybrisAppSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HybrisAppSpec.
func (in *HybrisAppSpec) DeepCopy() *HybrisAppSpec {
	if in == nil {
		return nil
	}
	out := new(HybrisAppSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HybrisAppStatus) DeepCopyInto(out *HybrisAppStatus) {
	*out = *in
	if in.BuildConditions != nil {
		in, out := &in.BuildConditions, &out.BuildConditions
		*out = make([]BuildStatusCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.DeploymentConfigConditions.DeepCopyInto(&out.DeploymentConfigConditions)
	if in.RouteConditions != nil {
		in, out := &in.RouteConditions, &out.RouteConditions
		*out = make([]RouteStatusCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HybrisAppStatus.
func (in *HybrisAppStatus) DeepCopy() *HybrisAppStatus {
	if in == nil {
		return nil
	}
	out := new(HybrisAppStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HybrisBase) DeepCopyInto(out *HybrisBase) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HybrisBase.
func (in *HybrisBase) DeepCopy() *HybrisBase {
	if in == nil {
		return nil
	}
	out := new(HybrisBase)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *HybrisBase) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HybrisBaseList) DeepCopyInto(out *HybrisBaseList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]HybrisBase, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HybrisBaseList.
func (in *HybrisBaseList) DeepCopy() *HybrisBaseList {
	if in == nil {
		return nil
	}
	out := new(HybrisBaseList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *HybrisBaseList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HybrisBaseSpec) DeepCopyInto(out *HybrisBaseSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HybrisBaseSpec.
func (in *HybrisBaseSpec) DeepCopy() *HybrisBaseSpec {
	if in == nil {
		return nil
	}
	out := new(HybrisBaseSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HybrisBaseStatus) DeepCopyInto(out *HybrisBaseStatus) {
	*out = *in
	if in.BuildConditions != nil {
		in, out := &in.BuildConditions, &out.BuildConditions
		*out = make([]BuildStatusCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HybrisBaseStatus.
func (in *HybrisBaseStatus) DeepCopy() *HybrisBaseStatus {
	if in == nil {
		return nil
	}
	out := new(HybrisBaseStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RouteStatusCondition) DeepCopyInto(out *RouteStatusCondition) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]status.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RouteStatusCondition.
func (in *RouteStatusCondition) DeepCopy() *RouteStatusCondition {
	if in == nil {
		return nil
	}
	out := new(RouteStatusCondition)
	in.DeepCopyInto(out)
	return out
}
