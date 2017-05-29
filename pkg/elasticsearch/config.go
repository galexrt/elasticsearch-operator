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
	"github.com/galexrt/elasticsearch-operator/pkg/utils"
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

	var part *v1alpha1.ElasticsearchPartSpec

	if tkey == "data" {
		part = p.Spec.Data
	} else if tkey == "master" {
		part = p.Spec.Master
	} else if tkey == "ingest" {
		part = p.Spec.Ingest
	}
	configs[configFilename] = generateElasticsearchConfig(p, tkey, part)
	configs[jvmOpts] = generateJvmOptsConfig(p, part)
	configs[log4jFilename] = generateLog4JConfig(p, part)

	return configs, nil
}

func generateElasticsearchConfig(p *v1alpha1.Elasticsearch, tkey string, part *v1alpha1.ElasticsearchPartSpec) []byte {
	minimumMasterNodes := *p.Spec.Master.Replicas - int32(1)
	if minimumMasterNodes <= 0 {
		minimumMasterNodes = 1
	}

	config := []byte(`cluster:
  name: ` + p.Name + `

node:
  name: ${NODE_NAME}
  master: ${NODE_MASTER}
  data: ${NODE_DATA}
  ingest: ${NODE_INGEST}
  # was ${MAX_LOCAL_STORAGE_NODES}
  max_local_storage_nodes: 1
network.host: 0.0.0.0
path:
  data: /data/data
  logs: /data/log
bootstrap:
  memory_lock: true
http:
  enabled: true
  compression: true
  cors:
    enabled: true
    allow-origin: "*"
discovery:
  zen:
    ping.unicast.hosts: elasticsearch-discovery
    minimum_master_nodes: ` + utils.String(minimumMasterNodes) + "\n")

	if part != nil {
		if len(part.AdditionalConfig) > 0 {
			config = append(config, "\n"+part.AdditionalConfig...)
		}
	}

	if len(p.Spec.AdditionalConfig) > 0 {
		config = append(config, "\n"+p.Spec.AdditionalConfig...)
	}

	return config
}

func generateJvmOptsConfig(p *v1alpha1.Elasticsearch, part *v1alpha1.ElasticsearchPartSpec) []byte {
	// TODO(galexrt)
	config := []byte(`-XX:+UseConcMarkSweepGC
-XX:CMSInitiatingOccupancyFraction=75
-XX:+UseCMSInitiatingOccupancyOnly
-XX:+DisableExplicitGC
-XX:+AlwaysPreTouch
-server
-Xss1m
-Djava.awt.headless=true
-Dfile.encoding=UTF-8
-Djna.nosys=true
-Djdk.io.permissionsUseCanonicalPath=true
-Dio.netty.noUnsafe=true
-Dio.netty.noKeySetOptimization=true
-Dlog4j.shutdownHookEnabled=false
-Dlog4j2.disable.jmx=true
-Dlog4j.skipJansi=true
-XX:+HeapDumpOnOutOfMemoryError
`)

	if len(part.JavaOpts) > 0 {
		config = append(config, "\n"+part.JavaOpts...)
	}

	if p.Spec.JavaMemoryControl {
		// TODO(galexrt) add xmx thingy args to config
	}

	return config
}

func generateLog4JConfig(p *v1alpha1.Elasticsearch, part *v1alpha1.ElasticsearchPartSpec) []byte {
	// TODO(galexrt)
	config := []byte(`status = error
appender.console.type = Console
appender.console.name = console
appender.console.layout.type = PatternLayout
appender.console.layout.pattern = [%d{ISO8601}][%-5p][%-25c{1.}] %marker%m%n
rootLogger.level = info
rootLogger.appenderRef.console.ref = console
`)
	return config
}
