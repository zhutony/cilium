#!/bin/bash

set -e

docker pull busybox

docker tag busybox $1/cilium/cilium:$2
docker tag busybox $1/cilium/cilium-dev:$2
docker tag busybox $1/cilium/operator:$2

docker push $1/cilium/cilium:$2
docker push $1/cilium/cilium-dev:$2
docker push $1/cilium/operator:$2
