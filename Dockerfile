FROM quay.io/prometheus/busybox:latest

ENV ARCH="linux_amd64"

ADD output/elasticsearch-operator_$ARCH /bin/operator

ENTRYPOINT ["/bin/operator"]
