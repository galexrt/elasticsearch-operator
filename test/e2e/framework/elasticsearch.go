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

package framework

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/pkg/api/v1"

	"github.com/galexrt/elasticsearch-operator/pkg/client/monitoring/v1alpha1"
	"github.com/galexrt/elasticsearch-operator/pkg/elasticsearch"
	"github.com/pkg/errors"
)

func (f *Framework) MakeBasicElasticsearch(ns, name, group string, replicas int32) *v1alpha1.Elasticsearch {
	return &v1alpha1.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: v1alpha1.ElasticsearchSpec{
			Replicas: &replicas,
			RuleSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"role": "rulefile",
				},
			},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceMemory: resource.MustParse("400Mi"),
				},
			},
		},
	}
}

func (f *Framework) AddAlertingToElasticsearch(p *v1alpha1.Elasticsearch, ns, name string) {
	p.Spec.Alerting = v1alpha1.AlertingSpec{
		Alertmanagers: []v1alpha1.AlertmanagerEndpoints{
			v1alpha1.AlertmanagerEndpoints{
				Namespace: ns,
				Name:      fmt.Sprintf("alertmanager-%s", name),
				Port:      intstr.FromString("web"),
			},
		},
	}
}

func (f *Framework) MakeBasicElasticsearchNodePortService(name, group string, nodePort int32) *v1.Service {
	pService := f.MakeElasticsearchService(name, group, v1.ServiceTypeNodePort)
	pService.Spec.Ports[0].NodePort = nodePort
	return pService
}

func (f *Framework) MakeElasticsearchService(name, group string, serviceType v1.ServiceType) *v1.Service {
	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("elasticsearch-%s", name),
			Labels: map[string]string{
				"group": group,
			},
		},
		Spec: v1.ServiceSpec{
			Type: serviceType,
			Ports: []v1.ServicePort{
				v1.ServicePort{
					Name:       "web",
					Port:       9200,
					TargetPort: intstr.FromString("web"),
				},
			},
			Selector: map[string]string{
				"elasticsearch": name,
			},
		},
	}
	return service
}

func (f *Framework) CreateElasticsearchAndWaitUntilReady(ns string, p *v1alpha1.Elasticsearch) error {
	_, err := f.MonClient.Elastichearchs(ns).Create(p)
	if err != nil {
		return err
	}

	if err := f.WaitForElasticsearchReady(p, 5*time.Minute); err != nil {
		return fmt.Errorf("failed to create %d Elasticsearch instances (%v): %v", p.Spec.Replicas, p.Name, err)
	}

	return nil
}

func (f *Framework) UpdateElasticsearchAndWaitUntilReady(ns string, p *v1alpha1.Elasticsearch) error {
	_, err := f.MonClient.Elastichearchs(ns).Update(p)
	if err != nil {
		return err
	}
	if err := f.WaitForElasticsearchReady(p, 5*time.Minute); err != nil {
		return fmt.Errorf("failed to update %d Elasticsearch instances (%v): %v", p.Spec.Replicas, p.Name, err)
	}

	return nil
}

func (f *Framework) WaitForElasticsearchReady(p *v1alpha1.Elasticsearch, timeout time.Duration) error {
	return wait.Poll(2*time.Second, timeout, func() (bool, error) {
		st, _, err := elasticsearch.ElasticsearchStatus(f.KubeClient, p)
		if err != nil {
			log.Print(err)
			return false, nil
		}
		return st.UpdatedReplicas == *p.Spec.Replicas, nil
	})
}

func (f *Framework) DeleteElasticsearchAndWaitUntilGone(ns, name string) error {
	_, err := f.MonClient.Elastichearchs(ns).Get(name)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("requesting Elasticsearch tpr %v failed", name))
	}

	if err := f.MonClient.Elastichearchs(ns).Delete(name, nil); err != nil {
		return errors.Wrap(err, fmt.Sprintf("deleting Elasticsearch tpr %v failed", name))
	}

	if err := WaitForPodsReady(
		f.KubeClient,
		ns,
		f.DefaultTimeout,
		0,
		elasticsearch.ListOptions(name),
	); err != nil {
		return errors.Wrap(err, fmt.Sprintf("waiting for Elasticsearch tpr (%s) to vanish timed out", name))
	}

	return nil
}

func (f *Framework) WaitForElasticsearchRunImageAndReady(ns string, p *v1alpha1.Elasticsearch) error {
	if err := WaitForPodsRunImage(f.KubeClient, ns, int(*p.Spec.Replicas), promImage(p.Spec.Version), elasticsearch.ListOptions(p.Name)); err != nil {
		return err
	}
	return WaitForPodsReady(
		f.KubeClient,
		ns,
		f.DefaultTimeout,
		int(*p.Spec.Replicas),
		elasticsearch.ListOptions(p.Name),
	)
}

func promImage(version string) string {
	return fmt.Sprintf("quay.io/galexrt/elasticsearch-kubernetes:%s", version)
}

func (f *Framework) WaitForTargets(amount int) error {
	var targets []*Target

	if err := wait.Poll(time.Second, time.Minute*10, func() (bool, error) {
		var err error
		targets, err = f.GetActiveTargets()
		if err != nil {
			return false, err
		}

		if len(targets) == amount {
			return true, nil
		}

		return false, nil
	}); err != nil {
		return fmt.Errorf("waiting for targets timed out. %v of %v targets found. %v", len(targets), amount, err)
	}

	return nil
}

func (f *Framework) GetActiveTargets() ([]*Target, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s:30900/api/v1/targets", f.ClusterIP))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	rt := elasticsearchTargetAPIResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&rt); err != nil {
		return nil, err
	}

	return rt.Data.ActiveTargets, nil
}

type Target struct {
	ScrapeURL string `json:"scrapeUrl"`
}

type targetDiscovery struct {
	ActiveTargets []*Target `json:"activeTargets"`
}

type elasticsearchTargetAPIResponse struct {
	Status string           `json:"status"`
	Data   *targetDiscovery `json:"data"`
}
