# Elasticsearch Operator

[![CircleCI branch](https://img.shields.io/circleci/project/github/RedSparr0w/node-csgo-parser/master.svg)]() [![Docker Repository on Quay](https://quay.io/repository/galexrt/elasticsearch-operator/status "Docker Repository on Quay")](https://quay.io/repository/galexrt/elasticsearch-operator) [![Go Report Card](https://goreportcard.com/badge/github.com/galexrt/srcds_exporter)](https://goreportcard.com/report/github.com/galexrt/srcds_exporter) [![license](https://img.shields.io/github/license/mashape/apistatus.svg)]()

> **!!WARNING!!** This code is currently not maintained anymore, please use another Elasticsearch like [upmc-enterprises/elasticsearch-operator](https://github.com/upmc-enterprises/elasticsearch-operator) instead!
> If you want to pick up the repo/code, let me know by mail. Thanks!

The code in this repo is based of the [coreos/prometheus-operator](https://github.com/coreos/prometheus-operator).

**Project status: *alpha*** Not all planned features are completed. The API, spec, status
and other user facing objects are subject to change. We do not support backward-compatibility
for the alpha releases.

The Elasticsearch Operator for Kubernetes provides easy monitoring definitions for Kubernetes
services and deployment and management of Elasticsearch instances.

Once installed, the Elasticsearch Operator provides the following features:

* **Create/Destroy**: Easily launch a Elasticsearch cluster for your Kubernetes namespace,
  a specific application or team easily using the Operator.

* **Simple Configuration**: Configure the fundamentals of Elasticsearch like versions, persistence,
  and replicas from a native Kubernetes resource.

For an introduction to the Elasticsearch Operator, see the initial [blog
post](https://edenmal.net/2017/05/30/Kubernetes-Elasticsearch-Operator/).

The current project roadmap [can be found here](./ROADMAP.md).

## Prerequisites

Version `>=0.0.1` of the Elasticsearch Operator requires a Kubernetes
cluster of version `>=1.5.0`. If you are just starting out with the
Elasticsearch Operator, it is highly recommended to use the latest version.

## Third party resources

The Operator acts on the following [third party resources (TPRs)](http://kubernetes.io/docs/user-guide/thirdpartyresources/):

* **`Elasticsearch`**, which defines a desired Elasticsearch cluster.
  The Operator ensures at all times that the specific resource definitions are running.

* **`Curator`**, which defines a desired curator cronjob.
  The Operator ensures at all times that a cronjob matching the config and resource definition is running.

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
operator will automatically shut down and remove Elasticsearch and ElasticsearchCluster pods and Curator cronjobs, and associated configmaps.

```
for n in $(kubectl get namespaces -o jsonpath={..metadata.name}); do
  kubectl delete --all --namespace=$n elasticsearch
  kubectl delete --all --namespace=$n curator
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
  curator.elasticsearch.zerbytes.net \
  elasticsearch.elasticsearch.zerbytes.net \
```

**The Elasticsearch Operator collects anonymous usage statistics to help us learning how the software is being used and how we can improve it. To disable collection, run the Operator with the flag `-analytics=false`**
