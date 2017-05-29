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
	"regexp"

	"github.com/galexrt/elasticsearch-operator/pkg/client/monitoring/v1alpha1"
)

var (
	invalidLabelCharRE = regexp.MustCompile(`[^a-zA-Z0-9_]`)
	defaultJavaOpts    = `status = error
appender.console.type = Console
appender.console.name = console
appender.console.layout.type = PatternLayout
appender.console.layout.pattern = [%d{ISO8601}][%-5p][%-25c{1.}] %marker%m%n
rootLogger.level = info
rootLogger.appenderRef.console.ref = console`
)

func generateConfig(p *v1alpha1.Elasticsearch, tkey string) (map[string][]byte, error) {
	configs := map[string][]byte{}

	fmt.Printf("=>\n")
	fmt.Printf("=>\n")
	fmt.Printf("=> GENERATING CONFIG FOR %+v\n", tkey)
	fmt.Printf("=>\n")
	fmt.Printf("=>\n")

	var part *v1alpha1.ElasticsearchPartSpec

	if tkey == "data" {
		part = p.Spec.Data
	} else if tkey == "master" {
		part = p.Spec.Master
	} else if tkey == "ingest" {
		part = p.Spec.Ingest
	}
	configs[configFilename] = generateElasticsearchConfig(p, part)
	configs[jvmOpts] = generateJvmOptsConfig(p, part)
	configs[log4jFilename] = generateLog4JConfig(p, part)

	return configs, nil
}

func generateElasticsearchConfig(p *v1alpha1.Elasticsearch, part *v1alpha1.ElasticsearchPartSpec) []byte {
	config := []byte{}
	if part != nil {
		if len(part.AdditionalConfig) > 0 {
			config = append(config, "\n"+p.Spec.AdditionalConfig...)
		}
	}

	if len(p.Spec.AdditionalConfig) > 0 {
		config = append(config, "\n"+p.Spec.AdditionalConfig...)
	}
	return config
}

func generateJvmOptsConfig(p *v1alpha1.Elasticsearch, part *v1alpha1.ElasticsearchPartSpec) []byte {
	// TODO(galexrt)
	return []byte{}
}

func generateLog4JConfig(p *v1alpha1.Elasticsearch, part *v1alpha1.ElasticsearchPartSpec) []byte {
	// TODO(galexrt)
	return []byte{}
}
