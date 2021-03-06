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
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/apps/v1beta1"

	"github.com/galexrt/elasticsearch-operator/pkg/client/monitoring/v1alpha1"
	"github.com/galexrt/elasticsearch-operator/pkg/config"
	"github.com/pkg/errors"
)

const (
	governingServiceName = "elasticsearch-operated"
	defaultBaseImage     = "quay.io/galexrt/elasticsearch-kubernetes"
	defaultVersion       = "5.4.0"
)

var (
	minReplicas                 int32 = 1
	managedByOperatorLabel            = "managed-by"
	managedByOperatorLabelValue       = "elasticsearch-operator"
	managedByOperatorLabels           = map[string]string{
		managedByOperatorLabel: managedByOperatorLabelValue,
	}
	probeTimeoutSeconds int32 = 3
)

func makeStatefulSets(el v1alpha1.Elasticsearch, tkey string, p *v1alpha1.ElasticsearchPartSpec, old *v1beta1.StatefulSet, config *config.Config) (*v1beta1.StatefulSet, error) {
	// TODO(fabxc): is this the right point to inject defaults?
	// Ideally we would do it before storing but that's currently not possible.
	// Potentially an update handler on first insertion.

	version := el.Spec.Version
	name := el.ObjectMeta.Name
	nameWithKey := el.ObjectMeta.Name + "-" + tkey

	if p.BaseImage == "" {
		p.BaseImage = defaultBaseImage
	}
	if p.Version == "" && version == "" {
		p.Version = defaultVersion
	}
	if p.Replicas != nil && *p.Replicas < minReplicas {
		p.Replicas = &minReplicas
	}

	if p.Resources.Requests == nil {
		p.Resources.Requests = v1.ResourceList{}
	}
	if _, ok := p.Resources.Requests[v1.ResourceMemory]; !ok {
		p.Resources.Requests[v1.ResourceMemory] = resource.MustParse("2Gi")
	}

	spec, err := makeStatefulSetSpec(name, tkey, &el, p, config)
	if err != nil {
		return nil, errors.Wrap(err, "make StatefulSet spec")
	}

	statefulset := &v1beta1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        prefixedName(nameWithKey),
			Labels:      el.ObjectMeta.Labels,
			Annotations: el.ObjectMeta.Annotations,
		},
		Spec: *spec,
	}

	// TODO(galexrt) check why k8s.io/client-go hasn't this implemented yet..
	//if el.Spec.ImagePullSecrets != nil && len(el.Spec.ImagePullSecrets) > 0 {
	//	statefulset.Template.ImagePullSecrets = el.Spec.ImagePullSecrets
	//}

	if vc := p.Storage; vc == nil {
		statefulset.Spec.Template.Spec.Volumes = append(statefulset.Spec.Template.Spec.Volumes, v1.Volume{
			Name: volumeName(nameWithKey),
			VolumeSource: v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{},
			},
		})
	} else {
		pvc := v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: volumeName(nameWithKey),
			},
			Spec: v1.PersistentVolumeClaimSpec{
				AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
				Resources:   vc.Resources,
				Selector:    vc.Selector,
			},
		}
		if len(vc.Class) > 0 {
			pvc.ObjectMeta.Annotations = map[string]string{
				"volume.beta.kubernetes.io/storage-class": vc.Class,
			}
		}
		statefulset.Spec.VolumeClaimTemplates = append(statefulset.Spec.VolumeClaimTemplates, pvc)
	}

	if old != nil {
		statefulset.Annotations = old.Annotations

		// mounted volumes are not reconciled as StatefulSets do not allow
		// modification of the PodTemplate.
		// TODO(brancz): remove this once StatefulSets allow modification of the
		// PodTemplate.
		statefulset.Spec.Template.Spec.Containers[0].VolumeMounts = old.Spec.Template.Spec.Containers[0].VolumeMounts
		statefulset.Spec.Template.Spec.Volumes = old.Spec.Template.Spec.Volumes
	}

	return statefulset, nil
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
	data := map[string][]byte{
		configFilename: []byte{},
		jvmOpts:        []byte{},
		log4jFilename:  []byte{},
	}
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:   configSecretName(name),
			Labels: managedByOperatorLabels,
		},
		Data: data,
	}, nil
}

func makeGovenorningService(p *v1alpha1.Elasticsearch) *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: governingServiceName,
			Labels: map[string]string{
				"operated-elasticsearch": "true",
			},
		},
		Spec: v1.ServiceSpec{
			ClusterIP: "None",
			Ports: []v1.ServicePort{
				{
					Name:       "http",
					Port:       9200,
					TargetPort: intstr.FromString("http"),
				},
			},
			Selector: map[string]string{
				"app": "elasticsearch",
			},
		},
	}
}

func makeStatefulSetService(name, tkey string) *v1.Service {
	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"operated-elasticsearch": "true",
				"elasticsearch":          name,
			},
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name:       "transport",
					Port:       9300,
					TargetPort: intstr.FromString("transport"),
					Protocol:   v1.ProtocolTCP,
				},
			},
			Selector: map[string]string{
				"app":  "elasticsearch",
				"role": tkey,
			},
		},
	}
	if tkey != "discovery" && tkey != "ingest" {
		svc.Spec.ClusterIP = "None"
	} else {
		svc.Spec.Selector["role"] = "master"
	}
	if tkey == "ingest" {
		svc.Spec.Ports = []v1.ServicePort{
			{
				Name:       "http",
				Port:       9200,
				TargetPort: intstr.FromString("http"),
				Protocol:   v1.ProtocolTCP,
			},
		}
	} else {

	}
	return svc
}

func makeStatefulSetSpec(name, tkey string, el *v1alpha1.Elasticsearch, p *v1alpha1.ElasticsearchPartSpec, c *config.Config) (*v1beta1.StatefulSetSpec, error) {
	// Elasticsearch may take quite long to shut down to checkpoint existing data.
	// Allow up to 5 minutes for clean termination.
	// TODO change back to 300 or so
	terminationGracePeriod := int64(10)

	volumes := []v1.Volume{
		{
			Name: "config",
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: configSecretName(name + "-" + tkey),
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
	}

	volumeMounts := []v1.VolumeMount{
		{
			Name:      "config",
			ReadOnly:  true,
			MountPath: "/config",
		},
		{
			Name:      volumeName(name + "-" + tkey),
			MountPath: "/data",
		},
	}

	ports := []v1.ContainerPort{
		{
			Name:          "http",
			ContainerPort: 9200,
			Protocol:      v1.ProtocolTCP,
		},
		{
			Name:          "transport",
			ContainerPort: 9300,
			Protocol:      v1.ProtocolTCP,
		},
	}

	probeHandler := v1.Handler{
		TCPSocket: &v1.TCPSocketAction{
			Port: intstr.FromString("transport"),
		},
	}

	securityContext := &v1.SecurityContext{
		Capabilities: &v1.Capabilities{
			Add: []v1.Capability{
				"IPC_LOCK",
				"SYS_RESOURCE",
			},
		},
	}

	statefulSetSpec := &v1beta1.StatefulSetSpec{
		ServiceName: governingServiceName,
		Replicas:    p.Replicas,
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"app":           "elasticsearch",
					"elasticsearch": name,
					"role":          tkey,
				},
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:         "elasticsearch",
						Image:        fmt.Sprintf("%s:%s", p.BaseImage, el.Spec.Version),
						Ports:        ports,
						VolumeMounts: volumeMounts,
						Args: []string{
							"/run.sh",
						},
						// TODO(galexrt) Allow env vars per part from user
						Env: []v1.EnvVar{
							{
								Name: "NODE_NAME",
								ValueFrom: &v1.EnvVarSource{
									FieldRef: &v1.ObjectFieldSelector{
										FieldPath: "metadata.name",
									},
								},
							},
							{
								Name:  "CONFIG_DIR",
								Value: "/config",
							},
						},
						LivenessProbe: &v1.Probe{
							Handler: probeHandler,
							// For larger servers, restoring a checkpoint on startup may take quite a bit of time.
							// Wait up to 5 minutes.
							InitialDelaySeconds: 120,
							PeriodSeconds:       5,
							TimeoutSeconds:      probeTimeoutSeconds,
							FailureThreshold:    10,
						},
						/*
							TODO(galexrt) check back later if we can enable this again
							as the discovery service isn't "activating" with this..
							ReadinessProbe: &v1.Probe{
								Handler:          probeHandler,
								TimeoutSeconds:   probeTimeoutSeconds,
								PeriodSeconds:    5,
								FailureThreshold: 6,
							},
						*/
						Resources:       p.Resources,
						SecurityContext: securityContext,
					},
				},
				ServiceAccountName:            el.Spec.ServiceAccountName,
				NodeSelector:                  p.NodeSelector,
				TerminationGracePeriodSeconds: &terminationGracePeriod,
				Volumes: volumes,
			},
		},
	}

	for _, key := range []string{
		"master",
		"data",
		"ingest",
	} {
		value := "false"
		if key == tkey {
			value = "true"
		}

		statefulSetSpec.Template.Spec.Containers[0].Env = append(statefulSetSpec.Template.Spec.Containers[0].Env, v1.EnvVar{
			Name:  "NODE_" + strings.ToUpper(key),
			Value: value,
		})
	}

	for _, env := range p.Env {
		statefulSetSpec.Template.Spec.Containers[0].Env = append(statefulSetSpec.Template.Spec.Containers[0].Env, env)
	}

	return statefulSetSpec, nil
}

func configSecretName(name string) string {
	return prefixedName(name)
}

func volumeName(name string) string {
	return fmt.Sprintf("%s-data", prefixedName(name))
}

func prefixedName(name string) string {
	return fmt.Sprintf("elasticsearch-%s", name)
}
