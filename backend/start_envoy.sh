#!/bin/bash

envoy -c /etc/envoy.json --service-cluster donutsalon-${SERVICE_NAME} --service-node `hostname` --restart-epoch $RESTART_EPOCH
