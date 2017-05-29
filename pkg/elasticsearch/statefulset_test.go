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

package elasticsearch

import (
	"fmt"
	"reflect"
	"testing"

	machineryv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/apps/v1beta1"

	"github.com/galexrt/elasticsearch-operator/pkg/client/monitoring/v1alpha1"
	"github.com/galexrt/elasticsearch-operator/pkg/config"
	"github.com/stretchr/testify/require"
)

var (
	defaultTestConfig = &config.Config{}
)

func TestStatefulSetLabelingAndAnnotations(t *testing.T) {
	labels := map[string]string{
		"testlabel": "testlabelvalue",
	}
	annotations := map[string]string{
		"testannotation": "testannotationvalue",
	}

	sset, err := makeStatefulSets(v1alpha1.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      labels,
			Annotations: annotations,
		},
	}, "master", &v1alpha1.ElasticsearchPartSpec{}, nil, defaultTestConfig)

	require.NoError(t, err)

	if !reflect.DeepEqual(labels, sset.Labels) || !reflect.DeepEqual(annotations, sset.Annotations) {
		t.Fatal("Labels or Annotations are not properly being propagated to the StatefulSet")
	}
}

func TestStatefulSetVolumeInitial(t *testing.T) {
	expected := &v1beta1.StatefulSet{
		Spec: v1beta1.StatefulSetSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "config",
									ReadOnly:  true,
									MountPath: "/config",
								},
								{
									Name:      "elasticsearch-example-master-data",
									ReadOnly:  false,
									MountPath: "/data",
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "config",
							VolumeSource: v1.VolumeSource{
								Secret: &v1.SecretVolumeSource{
									SecretName: configSecretName("example-master"),
									Items: []v1.KeyToPath{
										{
											Key:  configFilename,
											Path: configFilename,
										},
										{
											Key:  "log4j2.properties",
											Path: "log4j2.properties",
										},
										{
											Key:  "jvm.options",
											Path: "jvm.options",
										},
									},
								},
							},
						},
						{
							Name: "elasticsearch-example-master-data",
							VolumeSource: v1.VolumeSource{
								EmptyDir: &v1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}

	sset, err := makeStatefulSets(v1alpha1.Elasticsearch{
		ObjectMeta: machineryv1.ObjectMeta{
			Name: "example",
		},
	}, "master", &v1alpha1.ElasticsearchPartSpec{}, nil, defaultTestConfig)

	require.NoError(t, err)

	if !reflect.DeepEqual(expected.Spec.Template.Spec.Volumes, sset.Spec.Template.Spec.Volumes) || !reflect.DeepEqual(expected.Spec.Template.Spec.Containers[0].VolumeMounts, sset.Spec.Template.Spec.Containers[0].VolumeMounts) {
		fmt.Printf("%+v\n", expected.Spec.Template.Spec.Containers[0].VolumeMounts)
		fmt.Printf("%+v\n", sset.Spec.Template.Spec.Containers[0].VolumeMounts)
		t.Fatal("Volumes mounted in a Pod are not created correctly initially.")
	}
}

func TestStatefulSetVolumeSkip(t *testing.T) {
	old := &v1beta1.StatefulSet{
		ObjectMeta: machineryv1.ObjectMeta{
			Name: "example",
		},
		Spec: v1beta1.StatefulSetSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "config",
									ReadOnly:  true,
									MountPath: "/config",
								},
								{
									Name:      "elasticsearch-example-master-data",
									ReadOnly:  false,
									MountPath: "/data",
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "config",
							VolumeSource: v1.VolumeSource{
								Secret: &v1.SecretVolumeSource{
									SecretName: configSecretName("example-master"),
									Items: []v1.KeyToPath{
										{
											Key:  configFilename,
											Path: configFilename,
										},
										{
											Key:  "log4j2.properties",
											Path: "log4j2.properties",
										},
										{
											Key:  "jvm.options",
											Path: "jvm.options",
										},
									},
								},
							},
						},
						{
							Name: "elasticsearch-example-master-data",
							VolumeSource: v1.VolumeSource{
								EmptyDir: &v1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}

	sset, err := makeStatefulSets(v1alpha1.Elasticsearch{
		ObjectMeta: machineryv1.ObjectMeta{
			Name: "example",
		},
	}, "master", &v1alpha1.ElasticsearchPartSpec{}, old, defaultTestConfig)

	require.NoError(t, err)

	if !reflect.DeepEqual(old.Spec.Template.Spec.Volumes, sset.Spec.Template.Spec.Volumes) || !reflect.DeepEqual(old.Spec.Template.Spec.Containers[0].VolumeMounts, sset.Spec.Template.Spec.Containers[0].VolumeMounts) {
		t.Fatal("Volumes mounted in a Pod should not be reconciled.")
	}
}

func makeConfigMap() *v1.ConfigMap {
	res := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testcm",
			Namespace: "default",
		},
		Data: map[string]string{},
	}

	res.Data["test1"] = "value 1"
	res.Data["test2"] = "value 2"
	res.Data["test3"] = "value 3"

	return res
}
