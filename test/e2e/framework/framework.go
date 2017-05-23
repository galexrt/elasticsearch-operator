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
	"net/http"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/galexrt/elasticsearch-operator/pkg/client/monitoring/v1alpha1"
	"github.com/galexrt/elasticsearch-operator/pkg/k8sutil"
	"github.com/pkg/errors"
)

type Framework struct {
	KubeClient     kubernetes.Interface
	MonClient      *v1alpha1.MonitoringV1alpha1Client
	HTTPClient     *http.Client
	MasterHost     string
	Namespace      *v1.Namespace
	OperatorPod    *v1.Pod
	ClusterIP      string
	DefaultTimeout time.Duration
}

// Setup setups a test framework and returns it.
func New(ns, kubeconfig, opImage, ip string) (*Framework, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	cli, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	httpc := cli.CoreV1().RESTClient().(*rest.RESTClient).Client
	if err != nil {
		return nil, err
	}

	mclient, err := v1alpha1.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	namespace, err := CreateNamespace(cli, ns)
	if err != nil {
		return nil, err
	}

	f := &Framework{
		MasterHost:     config.Host,
		KubeClient:     cli,
		MonClient:      mclient,
		HTTPClient:     httpc,
		Namespace:      namespace,
		ClusterIP:      ip,
		DefaultTimeout: time.Minute,
	}

	err = f.setup(opImage)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func (f *Framework) setup(opImage string) error {
	if err := f.setupElasticsearchOperator(opImage); err != nil {
		return err
	}
	return nil
}

func (f *Framework) setupElasticsearchOperator(opImage string) error {
	deploy, err := MakeDeployment("../../example/non-rbac/elasticsearch-operator.yaml")
	if err != nil {
		return err
	}

	if opImage != "" {
		// Override operator image used, if specified when running tests.
		deploy.Spec.Template.Spec.Containers[0].Image = opImage
	}

	err = CreateDeployment(f.KubeClient, f.Namespace.Name, deploy)
	if err != nil {
		return err
	}

	opts := metav1.ListOptions{LabelSelector: fields.SelectorFromSet(fields.Set(deploy.Spec.Template.ObjectMeta.Labels)).String()}
	err = WaitForPodsReady(f.KubeClient, f.Namespace.Name, f.DefaultTimeout, 1, opts)
	if err != nil {
		return errors.Wrap(err, "failed to wait for elasticsearch operator to become ready")
	}

	pl, err := f.KubeClient.Core().Pods(f.Namespace.Name).List(opts)
	if err != nil {
		return err
	}
	f.OperatorPod = &pl.Items[0]

	return k8sutil.WaitForTPRReady(f.KubeClient.Core().RESTClient(), v1alpha1.TPRGroup, v1alpha1.TPRVersion, v1alpha1.TPRElasticsearchName)
}

// Teardown tears down a previously initialized test environment.
func (f *Framework) Teardown() error {
	if err := f.KubeClient.Core().Services(f.Namespace.Name).Delete("elasticsearch-operated", nil); err != nil && !k8sutil.IsResourceNotFoundError(err) {
		return err
	}

	if err := f.KubeClient.Extensions().Deployments(f.Namespace.Name).Delete("elasticsearch-operator", nil); err != nil {
		return err
	}
	if err := DeleteNamespace(f.KubeClient, f.Namespace.Name); err != nil {
		return err
	}

	return nil
}
