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
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

const (
	TPRElasticsearchClusterKind = "Elasticsearch"
	TPRElasticsearchClusterName = "elasticsearchs"
)

type ElasticsearchClustersGetter interface {
	ElasticsearchClusters(namespace string) ElasticsearchClusterInterface
}

type ElasticsearchClusterInterface interface {
	Create(*ElasticsearchCluster) (*ElasticsearchCluster, error)
	Get(name string) (*ElasticsearchCluster, error)
	Update(*ElasticsearchCluster) (*ElasticsearchCluster, error)
	Delete(name string, options *metav1.DeleteOptions) error
	List(opts metav1.ListOptions) (runtime.Object, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
}

type elasticsearchClusters struct {
	restClient rest.Interface
	client     *dynamic.ResourceClient
	ns         string
}

func newElasticsearchClusters(r rest.Interface, c *dynamic.Client, namespace string) *elasticsearchClusters {
	return &elasticsearchClusters{
		r,
		c.Resource(
			&metav1.APIResource{
				Kind:       TPRElasticsearchClusterKind,
				Name:       TPRElasticsearchClusterName,
				Namespaced: true,
			},
			namespace,
		),
		namespace,
	}
}

func (p *elasticsearchClusters) Create(o *ElasticsearchCluster) (*ElasticsearchCluster, error) {
	up, err := UnstructuredFromElasticsearchCluster(o)
	if err != nil {
		return nil, err
	}

	up, err = p.client.Create(up)
	if err != nil {
		return nil, err
	}

	return ElasticsearchClusterFromUnstructured(up)
}

func (p *elasticsearchClusters) Get(name string) (*ElasticsearchCluster, error) {
	obj, err := p.client.Get(name)
	if err != nil {
		return nil, err
	}
	return ElasticsearchClusterFromUnstructured(obj)
}

func (p *elasticsearchClusters) Update(o *ElasticsearchCluster) (*ElasticsearchCluster, error) {
	up, err := UnstructuredFromElasticsearchCluster(o)
	if err != nil {
		return nil, err
	}

	up, err = p.client.Update(up)
	if err != nil {
		return nil, err
	}

	return ElasticsearchClusterFromUnstructured(up)
}

func (p *elasticsearchClusters) Delete(name string, options *metav1.DeleteOptions) error {
	return p.client.Delete(name, options)
}

func (p *elasticsearchClusters) List(opts metav1.ListOptions) (runtime.Object, error) {
	req := p.restClient.Get().
		Namespace(p.ns).
		Resource(TPRElasticsearchClusterName).
		// VersionedParams(&options, v1.ParameterCodec)
		FieldsSelectorParam(nil)

	b, err := req.DoRaw()
	if err != nil {
		return nil, err
	}
	var prom ElasticsearchList
	return &prom, json.Unmarshal(b, &prom)
}

func (p *elasticsearchClusters) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	r, err := p.restClient.Get().
		Prefix("watch").
		Namespace(p.ns).
		Resource(TPRElasticsearchClusterName).
		// VersionedParams(&options, v1.ParameterCodec).
		FieldsSelectorParam(nil).
		Stream()
	if err != nil {
		return nil, err
	}
	return watch.NewStreamWatcher(&elasticsearchClustersDecoder{
		dec:   json.NewDecoder(r),
		close: r.Close,
	}), nil
}

// ElasticsearchClusterFromUnstructured unmarshals a ElasticsearchCluster object from dynamic client's unstructured
func ElasticsearchClusterFromUnstructured(r *unstructured.Unstructured) (*ElasticsearchCluster, error) {
	b, err := json.Marshal(r.Object)
	if err != nil {
		return nil, err
	}
	var p ElasticsearchCluster
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, err
	}
	p.TypeMeta.Kind = TPRElasticsearchClusterKind
	p.TypeMeta.APIVersion = TPRGroup + "/" + TPRVersion
	return &p, nil
}

// UnstructuredFromElasticsearchCluster marshals a Elasticsearch object into dynamic client's unstructured
func UnstructuredFromElasticsearchCluster(p *ElasticsearchCluster) (*unstructured.Unstructured, error) {
	p.TypeMeta.Kind = TPRElasticsearchClusterKind
	p.TypeMeta.APIVersion = TPRGroup + "/" + TPRVersion
	b, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	var r unstructured.Unstructured
	if err := json.Unmarshal(b, &r.Object); err != nil {
		return nil, err
	}
	return &r, nil
}

type elasticsearchClustersDecoder struct {
	dec   *json.Decoder
	close func() error
}

func (d *elasticsearchClustersDecoder) Close() {
	d.close()
}

func (d *elasticsearchClustersDecoder) Decode() (action watch.EventType, object runtime.Object, err error) {
	var e struct {
		Type   watch.EventType
		Object Elasticsearch
	}
	if err := d.dec.Decode(&e); err != nil {
		return watch.Error, nil, err
	}
	return e.Type, &e.Object, nil
}
