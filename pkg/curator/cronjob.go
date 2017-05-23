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
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
	batchv1 "k8s.io/client-go/pkg/apis/batch/v1"
	"k8s.io/client-go/pkg/apis/batch/v2alpha1"

	"github.com/galexrt/elasticsearch-operator/pkg/client/monitoring/v1alpha1"
	"github.com/pkg/errors"
)

const (
	defaultBaseImage = "quay.io/galexrt/curator"
	defaultVersion   = "5.4.0"
)

var (
	minReplicas                 int32 = 1
	managedByOperatorLabel            = "managed-by"
	managedByOperatorLabelValue       = "curator-operator"
	managedByOperatorLabels           = map[string]string{
		managedByOperatorLabel: managedByOperatorLabelValue,
	}
	probeTimeoutSeconds int32 = 3
)

func makeCronjob(p v1alpha1.Curator, old *v2alpha1.CronJob, config *Config) (*v2alpha1.CronJob, error) {
	// TODO(fabxc): is this the right point to inject defaults?
	// Ideally we would do it before storing but that's currently not possible.
	// Potentially an update handler on first insertion.

	if p.Spec.BaseImage == "" {
		p.Spec.BaseImage = defaultBaseImage
	}
	if p.Spec.Version == "" {
		p.Spec.Version = defaultVersion
	}

	if p.Spec.Resources.Requests == nil {
		p.Spec.Resources.Requests = v1.ResourceList{}
	}
	if _, ok := p.Spec.Resources.Requests[v1.ResourceMemory]; !ok {
		p.Spec.Resources.Requests[v1.ResourceMemory] = resource.MustParse("100Mi")
	}

	spec, err := makeCronJobSpec(p, config)
	if err != nil {
		return nil, errors.Wrap(err, "make StatefulSet spec")
	}

	cronjob := &v2alpha1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:        prefixedName(p.Name),
			Labels:      p.ObjectMeta.Labels,
			Annotations: p.ObjectMeta.Annotations,
		},
		Spec: *spec,
	}

	if old != nil {
		cronjob.Annotations = old.Annotations

		// mounted volumes are not reconciled as StatefulSets do not allow
		// modification of the PodTemplate.
		// TODO(brancz): remove this once StatefulSets allow modification of the
		// PodTemplate.
		cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].VolumeMounts = old.Spec.JobTemplate.Spec.Template.Spec.Containers[0].VolumeMounts
		cronjob.Spec.JobTemplate.Spec.Template.Spec.Volumes = old.Spec.JobTemplate.Spec.Template.Spec.Volumes
	}
	return cronjob, nil
}

func makeEmptyConfig(name string) (*v1.Secret, error) {
	s, err := makeConfigSecret(name)
	if err != nil {
		return nil, err
	}

	s.ObjectMeta.Annotations = map[string]string{
		"empty": "true",
	}

	return s, nil
}

type ConfigMapReference struct {
	Key      string `json:"key"`
	Checksum string `json:"checksum"`
}

type ConfigMapReferenceList struct {
	Items []*ConfigMapReference `json:"items"`
}

func (l *ConfigMapReferenceList) Len() int {
	return len(l.Items)
}

func (l *ConfigMapReferenceList) Less(i, j int) bool {
	return l.Items[i].Key < l.Items[j].Key
}

func (l *ConfigMapReferenceList) Swap(i, j int) {
	l.Items[i], l.Items[j] = l.Items[j], l.Items[i]
}

func makeConfigSecret(name string) (*v1.Secret, error) {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:   configSecretName(name),
			Labels: managedByOperatorLabels,
		},
		Data: map[string][]byte{
			configFilename: []byte{},
		},
	}, nil
}

func makeCronJobSpec(p v1alpha1.Curator, c *Config) (*v2alpha1.CronJobSpec, error) {
	// Curator may take quite long to shut down to save existing data.
	// Allow up to 10 minutes for clean termination.
	terminationGracePeriod := int64(600)

	volumes := []v1.Volume{
		{
			Name: "config",
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: configSecretName(p.Name),
				},
			},
		},
	}

	cronVolumeMounts := []v1.VolumeMount{
		{
			Name:      "config",
			ReadOnly:  true,
			MountPath: "/etc/curator/config",
		},
	}

	return &v2alpha1.CronJobSpec{
		Schedule: p.Spec.Schedule,
		JobTemplate: v2alpha1.JobTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"app":     "curator",
					"curator": p.Name,
				},
			},
			Spec: batchv1.JobSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name:         "curator",
								Image:        fmt.Sprintf("%s:%s", p.Spec.BaseImage, p.Spec.Version),
								VolumeMounts: cronVolumeMounts,
								Resources:    p.Spec.Resources,
							},
						},
						ServiceAccountName:            p.Spec.ServiceAccountName,
						NodeSelector:                  p.Spec.NodeSelector,
						TerminationGracePeriodSeconds: &terminationGracePeriod,
						Volumes: volumes,
					},
				},
			},
		},
	}, nil
}

func configSecretName(name string) string {
	return prefixedName(name)
}

func volumeName(name string) string {
	return fmt.Sprintf("%s-data", prefixedName(name))
}

func prefixedName(name string) string {
	return fmt.Sprintf("curator-%s", name)
}

func subPathForStorage(s *v1alpha1.StorageSpec) string {
	if s == nil {
		return ""
	}

	return "curator-data"
}
