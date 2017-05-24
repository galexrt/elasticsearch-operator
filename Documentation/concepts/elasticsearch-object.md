# elasticsearch-object
```
apiVersion: "elasticsearch.zerbytes.net/v1alpha1"
kind: "Elasticsearch"
metadata:
  name: "example"
spec:
  version: "5.4.0"
  additional_config: |
    action.auto_create_index: .security,.monitoring*,.watches,.triggered_watches,.watcher-history*,filebeat-*,metricbeat-*,packetbeat-*,winlogbeat-*,heartbeat-*
  master:
    replicas: 2
    resources:
      limits:
        cpu: "2.5"
        memory: "16Gi"
      requests:
        memory: "14Gi"
    additional_config: |
      # test
  data:
    replicas: 4
    resources:
      limits:
        cpu: "2.5"
        memory: "16Gi"
      requests:
        memory: "14Gi"
    additional_config: |
      # test
  ingest:
    replicas: 2
    resources:
      limits:
        cpu: "2.5"
        memory: "16Gi"
      requests:
        memory: "14Gi"
    storage:
      class: rbd
      resources:
        requests:
          storage: 50Gi
    additional_config: |
      # test
```
