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

	"github.com/galexrt/elasticsearch-operator/pkg/client/monitoring/v1alpha1"
)

var (
	invalidLabelCharRE = regexp.MustCompile(`[^a-zA-Z0-9_]`)
)

func generateConfig(p *v1alpha1.Elasticsearch) (map[string][]byte, error) {
	configs := map[string][]byte{
		"master": []byte(""),
		"data":   []byte(""),
		"ingest": []byte(""),
	}

	if p.Spec.Master != nil && len(p.Spec.Master.AdditionalConfig) > 0 {
		configs["master"] = append(configs["master"], "\n"+p.Spec.Master.AdditionalConfig...)
	}
	if p.Spec.Data != nil && len(p.Spec.Data.AdditionalConfig) > 0 {
		configs["data"] = append(configs["data"], "\n"+p.Spec.Data.AdditionalConfig...)
	}
	if p.Spec.Ingest != nil && len(p.Spec.Ingest.AdditionalConfig) > 0 {
		configs["ingest"] = append(configs["ingest"], "\n"+p.Spec.Ingest.AdditionalConfig...)
	}

	return configs, nil
}
