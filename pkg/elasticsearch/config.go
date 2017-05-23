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
	"regexp"

	yaml "gopkg.in/yaml.v2"

	"github.com/galexrt/elasticsearch-operator/pkg/client/monitoring/v1alpha1"
)

var (
	invalidLabelCharRE = regexp.MustCompile(`[^a-zA-Z0-9_]`)
)

func sanitizeLabelName(name string) string {
	return invalidLabelCharRE.ReplaceAllString(name, "_")
}

func stringMapToMapSlice(m map[string]string) yaml.MapSlice {
	res := yaml.MapSlice{}

	for k, v := range m {
		res = append(res, yaml.MapItem{Key: k, Value: v})
	}

	return res
}

func generateConfig(p *v1alpha1.Elasticsearch) ([]byte, error) {
	if p.Spec.Config == "" {
		// TODO(galexrt) generate "good" default config from the info given
		// like number of masters is replicas-1, etc.
	}
	return []byte(p.Spec.Config), nil
}
