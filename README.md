# Elasticsearch Operator

The code in this repo is based of the [coreos/prometheus-operator](https://github.com/coreos/prometheus-operator).

**Project status: *alpha*** Not all planned features are completed. The API, spec, status
and other user facing objects are subject to change. We do not support backward-compatibility
for the alpha releases.

The Elasticsearch Operator for Kubernetes provides easy monitoring definitions for Kubernetes
services and deployment and management of Elasticsearch instances.

Once installed, the Elasticsearch Operator provides the following features:

* **Create/Destroy**: Easily launch a Elasticsearch instance for your Kubernetes namespace,
  a specific application or team easily using the Operator.

* **Simple Configuration**: Configure the fundamentals of Elasticsearch like versions, persistence,
  retention policies, and replicas from a native Kubernetes resource.

* **Target Services via Labels**: Automatically generate monitoring target configurations based
  on familiar Kubernetes label queries; no need to learn a Elasticsearch specific configuration language.

For an introduction to the Elasticsearch Operator, see the initial [blog
post](https://coreos.com/blog/the-prometheus-operator.html).

**Documentation is hosted on [coreos.com](https://coreos.com/operators/prometheus/docs/latest/)**

The current project roadmap [can be found here](./ROADMAP.md).

## Prerequisites

Version `>=0.0.1` of the Elasticsearch Operator requires a Kubernetes
cluster of version `>=1.5.0`. If you are just starting out with the
Elasticsearch Operator, it is highly recommended to use the latest version.

## Third party resources

The Operator acts on the following [third party resources (TPRs)](http://kubernetes.io/docs/user-guide/thirdpartyresources/):

* **`Elasticsearch`**, which defines a desired Elasticsearch deployment.
  The Operator ensures at all times that a deployment matching the resource definition is running.

To learn more about the TPRs introduced by the Elasticsearch Operator have a look
at the [design doc](Documentation/design.md).

## Installation

Install the Operator inside a cluster by running the following command:

```
kubectl apply -f bundle.yaml
```

> Note: make sure to adapt the namespace in the ClusterRoleBinding if deploying in another namespace than the default namespace.

To run the Operator outside of a cluster:

```
make
hack/run-external.sh <kubectl cluster name>
```

## Removal

To remove the operator and Elasticsearch, first delete any third party resources you created in each namespace. The
operator will automatically shut down and remove Elasticsearch and Alertmanager pods, and associated configmaps.

```
for n in $(kubectl get namespaces -o jsonpath={..metadata.name}); do
  kubectl delete --all --namespace=$n elasticsearch
done
```

After a couple of minutes you can go ahead and remove the operator itself.

```
kubectl delete -f bundle.yaml
```

The operator automatically creates services in each namespace where you created a Elasticsearch resources,
and defines three third party resources. You can clean these up now.

```
for n in $(kubectl get namespaces -o jsonpath={..metadata.name}); do
  kubectl delete --ignore-not-found --namespace=$n service elasticsearch-operated
done

kubectl delete --ignore-not-found thirdpartyresource \
  elasticsearch.elasticsearch.zerbytes.net
```

**The Elasticsearch Operator collects anonymous usage statistics to help us learning how the software is being used and how we can improve it. To disable collection, run the Operator with the flag `-analytics=false`**
