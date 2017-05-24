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

func generateConfig(p *v1alpha1.Elasticsearch) ([]byte, error) {
	if p.Spec.Config == "" {
		// TODO(galexrt) generate "good" default config from the info given
		// like number of masters is replicas-1, etc.
		p.Spec.Config = `cluster:
  name: ${CLUSTER_NAME}

node:
  master: ${NODE_MASTER}
  data: ${NODE_DATA}
  name: ${NODE_NAME}
  ingest: ${NODE_INGEST}
  max_local_storage_nodes: ${MAX_LOCAL_STORAGE_NODES}

network.host: ${NETWORK_HOST}

path:
  data: /data/data
  logs: /data/log

bootstrap:
  memory_lock: true

http:
  enabled: ${HTTP_ENABLE}
  compression: true
  cors:
    enabled: ${HTTP_CORS_ENABLE}
    allow-origin: ${HTTP_CORS_ALLOW_ORIGIN}

discovery:
  zen:
    ping.unicast.hosts: ${DISCOVERY_SERVICE}
    minimum_master_nodes: ${NUMBER_OF_MASTERS}

# x-pack related settings
action.auto_create_index: .security,.monitoring*,.watches,.triggered_watches,.watcher-history*,filebeat-*,metricbeat-*,packetbeat-*,winlogbeat-*,heartbeat-*`
	}
	return []byte(p.Spec.Config), nil
}
