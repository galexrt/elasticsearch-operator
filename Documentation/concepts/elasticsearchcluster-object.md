# elasticsearchcluster-object
```
apiVersion: "elasticsearch.zerbytes.net/v1alpha1"
kind: "Elasticsearch"
metadata:
  name: "example"
spec:
  version: "5.4.0"
  # automatically calculate java memory opts
  javaMemoryControl: true
  paused: false
  baseImage: "quay.io/.."
  imagePullSecrets: "example"
  additionalConfig: |
    action.auto_create_index: .security,.monitoring*,.watches,.triggered_watches,.watcher-history*,filebeat-*,metricbeat-*,packetbeat-*,winlogbeat-*,heartbeat-*
  master:
    replicas: 2
    resources:
      limits:
        cpu: "2.5"
        memory: "16Gi"
      requests:
        memory: "14Gi"
    javaOpts: ""
    additionalConfig: |
      # test
    storage:
      class: rbd
      resources:
        requests:
          storage: 10Gi
  data:
    replicas: 4
    nodeSelector:
      "k8s.io/hostname": "example"
    resources:
      limits:
        cpu: "2.5"
        memory: "16Gi"
      requests:
        memory: "14Gi"
    javaOpts: ""
    additionalConfig: |
      # test
    storage:
      class: rbd
      resources:
        requests:
          storage: 50Gi
  ingest:
    replicas: 2
    resources:
      limits:
        cpu: "2.5"
        memory: "16Gi"
      requests:
        memory: "14Gi"
    javaOpts: ""
    additionalConfig: |
      # test
```
