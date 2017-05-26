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

package api

import (
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/go-kit/kit/log"
	"k8s.io/client-go/kubernetes"

	"github.com/galexrt/elasticsearch-operator/pkg/client/monitoring/v1alpha1"
	"github.com/galexrt/elasticsearch-operator/pkg/config"
	"github.com/galexrt/elasticsearch-operator/pkg/curator"
	"github.com/galexrt/elasticsearch-operator/pkg/elasticsearch"
	"github.com/galexrt/elasticsearch-operator/pkg/elasticsearchcluster"
	"github.com/galexrt/elasticsearch-operator/pkg/k8sutil"
)

type API struct {
	kclient *kubernetes.Clientset
	mclient *v1alpha1.MonitoringV1alpha1Client
	logger  log.Logger
}

func New(conf config.Config, l log.Logger) (*API, error) {
	cfg, err := k8sutil.NewClusterConfig(conf.Host, conf.TLSInsecure, &conf.TLSConfig)
	if err != nil {
		return nil, err
	}

	kclient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	mclient, err := v1alpha1.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &API{
		kclient: kclient,
		mclient: mclient,
		logger:  l,
	}, nil
}

var (
	elasticsearchRoute = regexp.MustCompile("/apis/elasticsearch.zerbytes.net/v1alpha1/namespaces/(.*)/elasticsearchs/(.*)/status")
	curatorRoute       = regexp.MustCompile("/apis/elasticsearch.zerbytes.net/v1alpha1/namespaces/(.*)/curators/(.*)/status")
)

func (api *API) Register(mux *http.ServeMux) {
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if elasticsearchRoute.MatchString(req.URL.Path) {
			api.elasticsearchStatus(w, req)
		} else if curatorRoute.MatchString(req.URL.Path) {
			api.curatorStatus(w, req)
		} else {
			w.WriteHeader(404)
		}
	})
}

type objectReference struct {
	name      string
	namespace string
}

func parseStatusURL(path string) objectReference {
	matches := elasticsearchRoute.FindAllStringSubmatch(path, -1)
	ns := ""
	name := ""
	if len(matches) == 1 {
		if len(matches[0]) == 3 {
			ns = matches[0][1]
			name = matches[0][2]
		}
	}

	return objectReference{
		name:      name,
		namespace: ns,
	}
}

func (api *API) elasticsearchStatus(w http.ResponseWriter, req *http.Request) {
	or := parseStatusURL(req.URL.Path)

	p, err := api.mclient.Elastichearchs(or.namespace).Get(or.name)
	if err != nil {
		if k8sutil.IsResourceNotFoundError(err) {
			w.WriteHeader(404)
		}
		api.logger.Log("error", err)
		return
	}

	p.Status, _, err = elasticsearch.ElasticsearchStatus(api.kclient, p)
	if err != nil {
		api.logger.Log("error", err)
	}

	b, err := json.Marshal(p)
	if err != nil {
		api.logger.Log("error", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(b)
}

func (api *API) curatorStatus(w http.ResponseWriter, req *http.Request) {
	or := parseStatusURL(req.URL.Path)

	p, err := api.mclient.Curators(or.namespace).Get(or.name)
	if err != nil {
		if k8sutil.IsResourceNotFoundError(err) {
			w.WriteHeader(404)
		}
		api.logger.Log("error", err)
		return
	}

	p.Status, _, err = curator.CuratorStatus(api.kclient, p)
	if err != nil {
		api.logger.Log("error", err)
	}

	b, err := json.Marshal(p)
	if err != nil {
		api.logger.Log("error", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(b)
}

func (api *API) elasticsearchClusterStatus(w http.ResponseWriter, req *http.Request) {
	or := parseStatusURL(req.URL.Path)

	p, err := api.mclient.ElasticsearchClusters(or.namespace).Get(or.name)
	if err != nil {
		if k8sutil.IsResourceNotFoundError(err) {
			w.WriteHeader(404)
		}
		api.logger.Log("error", err)
		return
	}

	p.Status, _, err = elasticsearchcluster.ElasticsearchClusterStatus(api.kclient, p)
	if err != nil {
		api.logger.Log("error", err)
	}

	b, err := json.Marshal(p)
	if err != nil {
		api.logger.Log("error", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(b)
}
