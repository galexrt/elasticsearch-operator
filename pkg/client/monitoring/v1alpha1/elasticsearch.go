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
	TPRElasticsearchKind = "Elasticsearch"
	TPRElasticsearchName = "elasticsearchs"
)

type ElastichearchsGetter interface {
	Elastichearchs(namespace string) ElasticsearchInterface
}

type ElasticsearchInterface interface {
	Create(*Elasticsearch) (*Elasticsearch, error)
	Get(name string) (*Elasticsearch, error)
	Update(*Elasticsearch) (*Elasticsearch, error)
	Delete(name string, options *metav1.DeleteOptions) error
	List(opts metav1.ListOptions) (runtime.Object, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
}

type elasticseachs struct {
	restClient rest.Interface
	client     *dynamic.ResourceClient
	ns         string
}

func newElastichearchs(r rest.Interface, c *dynamic.Client, namespace string) *elasticseachs {
	return &elasticseachs{
		r,
		c.Resource(
			&metav1.APIResource{
				Kind:       TPRElasticsearchKind,
				Name:       TPRElasticsearchName,
				Namespaced: true,
			},
			namespace,
		),
		namespace,
	}
}

func (p *elasticseachs) Create(o *Elasticsearch) (*Elasticsearch, error) {
	up, err := UnstructuredFromElasticsearch(o)
	if err != nil {
		return nil, err
	}

	up, err = p.client.Create(up)
	if err != nil {
		return nil, err
	}

	return ElasticsearchFromUnstructured(up)
}

func (p *elasticseachs) Get(name string) (*Elasticsearch, error) {
	obj, err := p.client.Get(name)
	if err != nil {
		return nil, err
	}
	return ElasticsearchFromUnstructured(obj)
}

func (p *elasticseachs) Update(o *Elasticsearch) (*Elasticsearch, error) {
	up, err := UnstructuredFromElasticsearch(o)
	if err != nil {
		return nil, err
	}

	up, err = p.client.Update(up)
	if err != nil {
		return nil, err
	}

	return ElasticsearchFromUnstructured(up)
}

func (p *elasticseachs) Delete(name string, options *metav1.DeleteOptions) error {
	return p.client.Delete(name, options)
}

func (p *elasticseachs) List(opts metav1.ListOptions) (runtime.Object, error) {
	req := p.restClient.Get().
		Namespace(p.ns).
		Resource(TPRElasticsearchName).
		// VersionedParams(&options, v1.ParameterCodec)
		FieldsSelectorParam(nil)

	b, err := req.DoRaw()
	if err != nil {
		return nil, err
	}
	var prom ElasticsearchList
	return &prom, json.Unmarshal(b, &prom)
}

func (p *elasticseachs) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	r, err := p.restClient.Get().
		Prefix("watch").
		Namespace(p.ns).
		Resource(TPRElasticsearchName).
		// VersionedParams(&options, v1.ParameterCodec).
		FieldsSelectorParam(nil).
		Stream()
	if err != nil {
		return nil, err
	}
	return watch.NewStreamWatcher(&elasticsearchDecoder{
		dec:   json.NewDecoder(r),
		close: r.Close,
	}), nil
}

// ElasticsearchFromUnstructured unmarshals a Elasticsearch object from dynamic client's unstructured
func ElasticsearchFromUnstructured(r *unstructured.Unstructured) (*Elasticsearch, error) {
	b, err := json.Marshal(r.Object)
	if err != nil {
		return nil, err
	}
	var p Elasticsearch
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, err
	}
	p.TypeMeta.Kind = TPRElasticsearchKind
	p.TypeMeta.APIVersion = TPRGroup + "/" + TPRVersion
	return &p, nil
}

// UnstructuredFromElasticsearch marshals a Elasticsearch object into dynamic client's unstructured
func UnstructuredFromElasticsearch(p *Elasticsearch) (*unstructured.Unstructured, error) {
	p.TypeMeta.Kind = TPRElasticsearchKind
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

type elasticsearchDecoder struct {
	dec   *json.Decoder
	close func() error
}

func (d *elasticsearchDecoder) Close() {
	d.close()
}

func (d *elasticsearchDecoder) Decode() (action watch.EventType, object runtime.Object, err error) {
	var e struct {
		Type   watch.EventType
		Object Elasticsearch
	}
	if err := d.dec.Decode(&e); err != nil {
		return watch.Error, nil, err
	}
	return e.Type, &e.Object, nil
}
