donutbin &
envoy -c /etc/envoy.json --service-cluster donutsalon-${SERVICE_NAME} --service-node `hostname`