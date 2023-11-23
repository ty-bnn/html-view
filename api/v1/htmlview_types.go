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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// HtmlViewSpec defines the desired state of HtmlView
type HtmlViewSpec struct {
	// Html contains HTML file
	//+kubebuilder:validation:Required
	Html map[string]string `json:"html,omitempty"`

	// Replicas is the number of pods.
	// +kubebuilder:default=1
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// Port is the ingress port.
	// +kubebuilder:default=80
	// +optional
	Port int32 `json:"port,omitempty"`
}

// TODO: statusを構造体で扱う

// HtmlViewStatus defines the observed state of HtmlView
// +kubebuilder:validation:Enum=NotReady;Running
type HtmlViewStatus string

const (
	HtmlViewNotReady = HtmlViewStatus("NotReady")
	HtmlViewRunning  = HtmlViewStatus("Running")
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="REPLICAS",type="integer",JSONPath=".spec.replicas"
//+kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status"

// HtmlView is the Schema for the htmlviews API
type HtmlView struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HtmlViewSpec   `json:"spec,omitempty"`
	Status HtmlViewStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// HtmlViewList contains a list of HtmlView
type HtmlViewList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HtmlView `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HtmlView{}, &HtmlViewList{})
}
