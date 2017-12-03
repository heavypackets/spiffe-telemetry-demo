donutbin &
envoy -c /etc/envoy.json --service-cluster donutsalon2-${SERVICE_NAME} --service-node `hostname`