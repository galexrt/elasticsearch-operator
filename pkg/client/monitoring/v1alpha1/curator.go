// Copyright 2017 The elasticsearch-operator Authors
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
	TPRCuratorKind = "Curator"
	TPRCuratorName = "curators"
)

type CuratoresGetter interface {
	Curatores(namespace string) CuratorInterface
}

type CuratorInterface interface {
	Create(*Curator) (*Curator, error)
	Get(name string) (*Curator, error)
	Update(*Curator) (*Curator, error)
	Delete(name string, options *metav1.DeleteOptions) error
	List(opts metav1.ListOptions) (runtime.Object, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
}

type curators struct {
	restClient rest.Interface
	client     *dynamic.ResourceClient
	ns         string
}

func newCurators(r rest.Interface, c *dynamic.Client, namespace string) *curators {
	return &curators{
		r,
		c.Resource(
			&metav1.APIResource{
				Kind:       TPRCuratorKind,
				Name:       TPRCuratorName,
				Namespaced: true,
			},
			namespace,
		),
		namespace,
	}
}

func (p *curators) Create(o *Curator) (*Curator, error) {
	up, err := UnstructuredFromCurator(o)
	if err != nil {
		return nil, err
	}

	up, err = p.client.Create(up)
	if err != nil {
		return nil, err
	}

	return CuratorFromUnstructured(up)
}

func (p *curators) Get(name string) (*Curator, error) {
	obj, err := p.client.Get(name)
	if err != nil {
		return nil, err
	}
	return CuratorFromUnstructured(obj)
}

func (p *curators) Update(o *Curator) (*Curator, error) {
	up, err := UnstructuredFromCurator(o)
	if err != nil {
		return nil, err
	}

	up, err = p.client.Update(up)
	if err != nil {
		return nil, err
	}

	return CuratorFromUnstructured(up)
}

func (p *curators) Delete(name string, options *metav1.DeleteOptions) error {
	return p.client.Delete(name, options)
}

func (p *curators) List(opts metav1.ListOptions) (runtime.Object, error) {
	req := p.restClient.Get().
		Namespace(p.ns).
		Resource(TPRCuratorName).
		// VersionedParams(&options, v1.ParameterCodec)
		FieldsSelectorParam(nil)

	b, err := req.DoRaw()
	if err != nil {
		return nil, err
	}
	var prom CuratorList
	return &prom, json.Unmarshal(b, &prom)
}

func (p *curators) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	r, err := p.restClient.Get().
		Prefix("watch").
		Namespace(p.ns).
		Resource(TPRCuratorName).
		// VersionedParams(&options, v1.ParameterCodec).
		FieldsSelectorParam(nil).
		Stream()
	if err != nil {
		return nil, err
	}
	return watch.NewStreamWatcher(&curatorDecoder{
		dec:   json.NewDecoder(r),
		close: r.Close,
	}), nil
}

// CuratorFromUnstructured unmarshals a Curator object from dynamic client's unstructured
func CuratorFromUnstructured(r *unstructured.Unstructured) (*Curator, error) {
	b, err := json.Marshal(r.Object)
	if err != nil {
		return nil, err
	}
	var p Curator
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, err
	}
	p.TypeMeta.Kind = TPRCuratorKind
	p.TypeMeta.APIVersion = TPRGroup + "/" + TPRVersion
	return &p, nil
}

// UnstructuredFromCurator marshals a Curator object into dynamic client's unstructured
func UnstructuredFromCurator(p *Curator) (*unstructured.Unstructured, error) {
	p.TypeMeta.Kind = TPRCuratorKind
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

type curatorDecoder struct {
	dec   *json.Decoder
	close func() error
}

func (d *curatorDecoder) Close() {
	d.close()
}

func (d *curatorDecoder) Decode() (action watch.EventType, object runtime.Object, err error) {
	var e struct {
		Type   watch.EventType
		Object Curator
	}
	if err := d.dec.Decode(&e); err != nil {
		return watch.Error, nil, err
	}
	return e.Type, &e.Object, nil
}
