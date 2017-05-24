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

package curator

import (
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/batch/v2alpha1"

	"github.com/galexrt/elasticsearch-operator/pkg/client/monitoring/v1alpha1"
	"github.com/galexrt/elasticsearch-operator/pkg/config"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/client-go/pkg/apis/batch/v1"
)

var (
	defaultTestConfig = &config.Config{}
)

func TestCronJobLabelingAndAnnotations(t *testing.T) {
	labels := map[string]string{
		"testlabel": "testlabelvalue",
	}
	annotations := map[string]string{
		"testannotation": "testannotationvalue",
	}

	cj, err := makeCronJob(v1alpha1.Curator{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      labels,
			Annotations: annotations,
		},
	}, nil, defaultTestConfig)

	require.NoError(t, err)

	labels["app"] = "curator"
	labels["curator"] = ""

	if !reflect.DeepEqual(labels, cj.Labels) || !reflect.DeepEqual(annotations, cj.Annotations) {
		t.Fatal("Labels or Annotations are not properly being propagated to the CronJob")
	}
}

func TestCronJobVolumeInitial(t *testing.T) {
	expected := &v2alpha1.CronJob{
		Spec: v2alpha1.CronJobSpec{
			JobTemplate: v2alpha1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									VolumeMounts: []v1.VolumeMount{
										{
											Name:      "config",
											ReadOnly:  true,
											MountPath: "/etc/curator/config",
											SubPath:   "",
										},
									},
								},
							},
							Volumes: []v1.Volume{
								{
									Name: "config",
									VolumeSource: v1.VolumeSource{
										Secret: &v1.SecretVolumeSource{
											SecretName: configSecretName(""),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	cj, err := makeCronJob(v1alpha1.Curator{}, nil, defaultTestConfig)

	require.NoError(t, err)

	if !reflect.DeepEqual(expected.Spec.JobTemplate.Spec.Template.Spec.Volumes, cj.Spec.JobTemplate.Spec.Template.Spec.Volumes) || !reflect.DeepEqual(expected.Spec.JobTemplate.Spec.Template.Spec.Containers[0].VolumeMounts, cj.Spec.JobTemplate.Spec.Template.Spec.Containers[0].VolumeMounts) {
		t.Fatal("Volumes mounted in a Pod are not created correctly initially.")
	}
}

func TestCronJobVolumeSkip(t *testing.T) {
	old := &v2alpha1.CronJob{
		Spec: v2alpha1.CronJobSpec{
			JobTemplate: v2alpha1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									VolumeMounts: []v1.VolumeMount{
										{
											Name:      "config",
											ReadOnly:  true,
											MountPath: "/etc/elasticsearch/config",
											SubPath:   "",
										},
									},
								},
							},
							Volumes: []v1.Volume{
								{
									Name: "config",
									VolumeSource: v1.VolumeSource{
										Secret: &v1.SecretVolumeSource{
											SecretName: configSecretName(""),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	cj, err := makeCronJob(v1alpha1.Curator{}, old, defaultTestConfig)

	require.NoError(t, err)

	if !reflect.DeepEqual(old.Spec.JobTemplate.Spec.Template.Spec.Volumes, cj.Spec.JobTemplate.Spec.Template.Spec.Volumes) || !reflect.DeepEqual(old.Spec.JobTemplate.Spec.Template.Spec.Containers[0].VolumeMounts, cj.Spec.JobTemplate.Spec.Template.Spec.Containers[0].VolumeMounts) {
		t.Fatal("Volumes mounted in a CronJob should not be reconciled.")
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
