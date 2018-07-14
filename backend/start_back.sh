#!/bin/bash

export JAEGER_AGENT_HOST=`getent hosts jaeger | awk '{ print $1 }'`
envsubst </etc/jaeger.yaml >/etc/envoy-jaeger.yaml
donutbin --service_hostport="front-envoy:80" --tracer_type="jaeger"
