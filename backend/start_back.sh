#!/bin/bash

envsubst </etc/jaeger.yaml >/etc/envoy-jaeger.yaml
donutbin --service_hostport="front-envoy:80" & 
