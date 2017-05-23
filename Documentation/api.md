# API Docs

This Document documents the types introduced by the Prometheus Operator to be consumed by users.

> Note this document is generated from code comments. When contributing a change to this document please do so by changing the code comments.

## Curator

Curator defines a elasticsearch curator job.

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata | Standard object’s metadata. More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata | [metav1.ObjectMeta](https://kubernetes.io/docs/api-reference/v1.6/#objectmeta-v1-meta) | false |
| spec | Specification of the desired behavior of the Elasticsearch cluster. More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#spec-and-status | [CuratorSpec](#curatorspec) | true |
| status | Most recent observed status of the Elasticsearch cluster. Read-only. Not included when requesting from the apiserver, only from the Elasticsearch Operator API itself. More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#spec-and-status | *[CuratorStatus](#curatorstatus) | false |

## CuratorList

CuratorList is a list of Curators.

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata | Standard list metadata More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata | [metav1.ListMeta](https://kubernetes.io/docs/api-reference/v1.6/#listmeta-v1-meta) | false |
| items | List of Elasticsearches | []*[Curator](#curator) | true |

## CuratorSpec

CuratorSpec Specification of the desired behavior of the Curator job. More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#spec-and-status

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| version | Version of Elasticsearch to be deployed. | string | false |
| paused | When a Elasticsearch deployment is paused, no actions except for deletion will be performed on the underlying objects. | bool | false |
| baseImage | Base image to use for a Elasticsearch deployment. | string | false |
| schedule | An optional list of references to secrets in the same namespace to use for pulling prometheus and alertmanager images from registries see http://kubernetes.io/docs/user-guide/images#specifying-imagepullsecrets-on-a-pod ImagePullSecrets []v1.LocalObjectReference `json:\"imagePullSecrets,omitempty\"` Number of instances to deploy for a Elasticsearch deployment. | string | false |
| config | The external URL the Elasticsearch instances will be available under. This is necessary to generate correct URLs. This is necessary if Elasticsearch is not served from root of a DNS name. | string | false |
| actions | The actions that should be built as a config. | string | false |
| resources | Define resources requests and limits for single Pods. | [v1.ResourceRequirements](https://kubernetes.io/docs/api-reference/v1.6/#resourcerequirements-v1-core) | false |
| nodeSelector | Define which Nodes the Pods are scheduled on. | map[string]string | false |
| serviceAccountName | ServiceAccountName is the name of the ServiceAccount to use to run the Elasticsearch Pods. | string | false |

## CuratorStatus

CuratorStatus Most recent observed status of the Curator job. Read-only. Not included when requesting from the apiserver, only from the Elasticsearch Operator API itself. More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#spec-and-status

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| paused | Represents whether any actions on the underlaying managed objects are being performed. Only delete actions will be performed. | bool | true |
| replicas | Total number of non-terminated pods targeted by this Elasticsearch deployment (their labels match the selector). | int32 | true |
| updatedReplicas | Total number of non-terminated pods targeted by this Elasticsearch deployment that have the desired version spec. | int32 | true |
| availableReplicas | Total number of available pods (ready for at least minReadySeconds) targeted by this Elasticsearch deployment. | int32 | true |
| unavailableReplicas | Total number of unavailable pods targeted by this Elasticsearch deployment. | int32 | true |

## Elasticsearch

Elasticsearch defines a Elasticsearch deployment.

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata | Standard object’s metadata. More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata | [metav1.ObjectMeta](https://kubernetes.io/docs/api-reference/v1.6/#objectmeta-v1-meta) | false |
| spec | Specification of the desired behavior of the Elasticsearch cluster. More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#spec-and-status | [ElasticsearchSpec](#elasticsearchspec) | true |
| status | Most recent observed status of the Elasticsearch cluster. Read-only. Not included when requesting from the apiserver, only from the Elasticsearch Operator API itself. More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#spec-and-status | *[ElasticsearchStatus](#elasticsearchstatus) | false |

## ElasticsearchList

ElasticsearchList is a list of Elasticsearches.

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata | Standard list metadata More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata | [metav1.ListMeta](https://kubernetes.io/docs/api-reference/v1.6/#listmeta-v1-meta) | false |
| items | List of Elasticsearches | []*[Elasticsearch](#elasticsearch) | true |

## ElasticsearchSpec

ElasticsearchSpec Specification of the desired behavior of the Elasticsearch cluster. More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#spec-and-status

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| version | Version of Elasticsearch to be deployed. | string | false |
| paused | When a Elasticsearch deployment is paused, no actions except for deletion will be performed on the underlying objects. | bool | false |
| baseImage | Base image to use for a Elasticsearch deployment. | string | false |
| imagePullSecrets | An optional list of references to secrets in the same namespace to use for pulling prometheus and alertmanager images from registries see http://kubernetes.io/docs/user-guide/images#specifying-imagepullsecrets-on-a-pod | [][v1.LocalObjectReference](https://kubernetes.io/docs/api-reference/v1.6/#localobjectreference-v1-core) | false |
| replicas | Number of instances to deploy for a Elasticsearch deployment. | *int32 | false |
| config | The external URL the Elasticsearch instances will be available under. This is necessary to generate correct URLs. This is necessary if Elasticsearch is not served from root of a DNS name. | string | false |
| storage | Storage spec to specify how storage shall be used. | *[StorageSpec](#storagespec) | false |
| resources | Define resources requests and limits for single Pods. | [v1.ResourceRequirements](https://kubernetes.io/docs/api-reference/v1.6/#resourcerequirements-v1-core) | false |
| nodeSelector | Define which Nodes the Pods are scheduled on. | map[string]string | false |
| serviceAccountName | ServiceAccountName is the name of the ServiceAccount to use to run the Elasticsearch Pods. | string | false |

## ElasticsearchStatus

ElasticsearchStatus Most recent observed status of the Elasticsearch cluster. Read-only. Not included when requesting from the apiserver, only from the Elasticsearch Operator API itself. More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#spec-and-status

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| paused | Represents whether any actions on the underlaying managed objects are being performed. Only delete actions will be performed. | bool | true |
| replicas | Total number of non-terminated pods targeted by this Elasticsearch deployment (their labels match the selector). | int32 | true |
| updatedReplicas | Total number of non-terminated pods targeted by this Elasticsearch deployment that have the desired version spec. | int32 | true |
| availableReplicas | Total number of available pods (ready for at least minReadySeconds) targeted by this Elasticsearch deployment. | int32 | true |
| unavailableReplicas | Total number of unavailable pods targeted by this Elasticsearch deployment. | int32 | true |

## StorageSpec

StorageSpec defines the configured storage for a group Elasticsearch servers.

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| class | Name of the StorageClass to use when requesting storage provisioning. More info: https://kubernetes.io/docs/user-guide/persistent-volumes/#storageclasses | string | true |
| selector | A label query over volumes to consider for binding. | *[metav1.LabelSelector](https://kubernetes.io/docs/api-reference/v1.6/#labelselector-v1-meta) | true |
| resources | Resources represents the minimum resources the volume should have. More info: http://kubernetes.io/docs/user-guide/persistent-volumes#resources | [v1.ResourceRequirements](https://kubernetes.io/docs/api-reference/v1.6/#resourcerequirements-v1-core) | true |

## TLSConfig

TLSConfig specifies TLS configuration parameters.

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| caFile | The CA cert to use for the targets. | string | false |
| certFile | The client cert file for the targets. | string | false |
| keyFile | The client key file for the targets. | string | false |
| serverName | Used to verify the hostname for the targets. | string | false |
| insecureSkipVerify | Disable target certificate validation. | bool | false |
