donutbin &
envoy -c /etc/envoy.json --service-cluster service${SERVICE_NAME} --service-node `hostname`