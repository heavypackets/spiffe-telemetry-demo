#!/bin/bash

envoy -c /etc/envoy.json --service-cluster front-proxy --service-node `hostname`
