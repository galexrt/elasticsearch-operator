# curator-object
```
apiVersion: "elasticsearch.zerbytes.net/v1alpha1"
kind: Curator
metadata:
  name: "example"
spec:
  schedule: "1 0 * * *"
  config: |
    YOUR_CONFIG
  actions:
    YOUR_ACTIONS
```
