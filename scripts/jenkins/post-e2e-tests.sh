#!/usr/bin/env bash
# This is a cleanup script, if one command fails we still want all others to run
# set -e
# only exit with zero if all commands of the pipeline exit successfully
set -o pipefail
# error on unset variables
set -u
# print each command before executing it
set -x

PO_GOPATH=/go/src/github.com/galexrt/elasticsearch-operator

docker run \
       --rm \
       -e AWS_ACCESS_KEY_ID -e AWS_SECRET_ACCESS_KEY \
       -v $PWD:$PO_GOPATH \
       -w $PO_GOPATH/scripts/jenkins \
       cluster-setup-env \
       /bin/bash -c "make clean"

docker rmi quay.io/galexrt/elasticsearch-operator-dev:$BUILD_ID
