// Copyright 2016 The prometheus-operator Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/batch/v2alpha1"
)

// Elasticsearch defines a Elasticsearch deployment.
type Elasticsearch struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object’s metadata. More info:
	// http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Specification of the desired behavior of the Elasticsearch cluster. More info:
	// http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#spec-and-status
	Spec ElasticsearchSpec `json:"spec"`
	// Most recent observed status of the Elasticsearch cluster. Read-only. Not
	// included when requesting from the apiserver, only from the Elasticsearch
	// Operator API itself. More info:
	// http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#spec-and-status
	Status *ElasticsearchStatus `json:"status,omitempty"`
}

// ElasticsearchList is a list of Elasticsearches.
type ElasticsearchList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata
	metav1.ListMeta `json:"metadata,omitempty"`
	// List of Elasticsearches
	Items []*Elasticsearch `json:"items"`
}

// ElasticsearchSpec Specification of the desired behavior of the Elasticsearch cluster. More info:
// http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#spec-and-status
type ElasticsearchSpec struct {
	// Version of Elasticsearch to be deployed.
	Version string `json:"version,omitempty"`
	// When a Elasticsearch deployment is paused, no actions except for deletion
	// will be performed on the underlying objects.
	Paused bool `json:"paused,omitempty"`
	// Base image to use for a Elasticsearch deployment.
	BaseImage string `json:"baseImage,omitempty"`
	// An optional list of references to secrets in the same namespace
	// to use for pulling prometheus and alertmanager images from registries
	// see http://kubernetes.io/docs/user-guide/images#specifying-imagepullsecrets-on-a-pod
	ImagePullSecrets []v1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	// Number of instances to deploy for a Elasticsearch deployment.
	Replicas *int32 `json:"replicas,omitempty"`
	// The external URL the Elasticsearch instances will be available under. This is
	// necessary to generate correct URLs. This is necessary if Elasticsearch is not
	// served from root of a DNS name.
	Config string `json:"config,omitempty"`
	// Storage spec to specify how storage shall be used.
	Storage *StorageSpec `json:"storage,omitempty"`
	// Define resources requests and limits for single Pods.
	Resources v1.ResourceRequirements `json:"resources,omitempty"`
	// Define which Nodes the Pods are scheduled on.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// ServiceAccountName is the name of the ServiceAccount to use to run the
	// Elasticsearch Pods.
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
	// EvaluationInterval string                    `json:"evaluationInterval"`
	// Remote          RemoteSpec                 `json:"remote"`
	// Sharding...
}

// ElasticsearchStatus Most recent observed status of the Elasticsearch cluster. Read-only. Not
// included when requesting from the apiserver, only from the Elasticsearch
// Operator API itself. More info:
// http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#spec-and-status
type ElasticsearchStatus struct {
	// Represents whether any actions on the underlaying managed objects are
	// being performed. Only delete actions will be performed.
	Paused bool `json:"paused"`
	// Total number of non-terminated pods targeted by this Elasticsearch deployment
	// (their labels match the selector).
	Replicas int32 `json:"replicas"`
	// Total number of non-terminated pods targeted by this Elasticsearch deployment
	// that have the desired version spec.
	UpdatedReplicas int32 `json:"updatedReplicas"`
	// Total number of available pods (ready for at least minReadySeconds)
	// targeted by this Elasticsearch deployment.
	AvailableReplicas int32 `json:"availableReplicas"`
	// Total number of unavailable pods targeted by this Elasticsearch deployment.
	UnavailableReplicas int32 `json:"unavailableReplicas"`
}

// StorageSpec defines the configured storage for a group Elasticsearch servers.
type StorageSpec struct {
	// Name of the StorageClass to use when requesting storage provisioning. More
	// info: https://kubernetes.io/docs/user-guide/persistent-volumes/#storageclasses
	Class string `json:"class"`
	// A label query over volumes to consider for binding.
	Selector *metav1.LabelSelector `json:"selector"`
	// Resources represents the minimum resources the volume should have. More
	// info: http://kubernetes.io/docs/user-guide/persistent-volumes#resources
	Resources v1.ResourceRequirements `json:"resources"`
}

// Curator defines a elasticsearch curator job.
type Curator struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object’s metadata. More info:
	// http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Specification of the desired behavior of the Elasticsearch cluster. More info:
	// http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#spec-and-status
	Spec CuratorSpec `json:"spec"`
	// Most recent observed status of the Elasticsearch cluster. Read-only. Not
	// included when requesting from the apiserver, only from the Elasticsearch
	// Operator API itself. More info:
	// http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#spec-and-status
	Status *v2alpha1.CronJobStatus `json:"status,omitempty"`
}

// CuratorList is a list of Curators.
type CuratorList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata
	metav1.ListMeta `json:"metadata,omitempty"`
	// List of Elasticsearches
	Items []*Curator `json:"items"`
}

// CuratorSpec Specification of the desired behavior of the Curator job. More info:
// http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#spec-and-status
type CuratorSpec struct {
	// Version of Elasticsearch to be deployed.
	Version string `json:"version,omitempty"`
	// When a Elasticsearch deployment is paused, no actions except for deletion
	// will be performed on the underlying objects.
	Paused bool `json:"paused,omitempty"`
	// Base image to use for a Elasticsearch deployment.
	BaseImage string `json:"baseImage,omitempty"`
	// An optional list of references to secrets in the same namespace
	// to use for pulling prometheus and alertmanager images from registries
	// see http://kubernetes.io/docs/user-guide/images#specifying-imagepullsecrets-on-a-pod
	//ImagePullSecrets []v1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	// Number of instances to deploy for a Elasticsearch deployment.
	Schedule string `json:"schedule"`
	// The external URL the Elasticsearch instances will be available under. This is
	// necessary to generate correct URLs. This is necessary if Elasticsearch is not
	// served from root of a DNS name.
	Config string `json:"config,omitempty"`
	// The actions that should be built as a config.
	Actions string `json:"actions,omitempty"`
	// Define resources requests and limits for single Pods.
	Resources v1.ResourceRequirements `json:"resources,omitempty"`
	// Define which Nodes the Pods are scheduled on.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// ServiceAccountName is the name of the ServiceAccount to use to run the
	// Elasticsearch Pods.
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
}

// CuratorStatus Most recent observed status of the Curator job. Read-only. Not
// included when requesting from the apiserver, only from the Elasticsearch
// Operator API itself. More info:
// http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#spec-and-status
type CuratorStatus struct {
	// Represents whether any actions on the underlaying managed objects are
	// being performed. Only delete actions will be performed.
	Paused bool `json:"paused"`
	// LastScheduleTime keeps information of when was the last time the job was successfully scheduled.
	// +optional
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty" protobuf:"bytes,4,opt,name=lastScheduleTime"`
}

// TLSConfig specifies TLS configuration parameters.
type TLSConfig struct {
	// The CA cert to use for the targets.
	CAFile string `json:"caFile,omitempty"`
	// The client cert file for the targets.
	CertFile string `json:"certFile,omitempty"`
	// The client key file for the targets.
	KeyFile string `json:"keyFile,omitempty"`
	// Used to verify the hostname for the targets.
	ServerName string `json:"serverName,omitempty"`
	// Disable target certificate validation.
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`
}
