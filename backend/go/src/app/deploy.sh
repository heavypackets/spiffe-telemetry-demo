#!/bin/bash

HOST=donutsalon.com

GOOS=linux GOARCH=amd64 go install github.com/bhs/opentracing-sandbox/donutsalon
cp ../bin/linux_amd64/donutsalon bin/linux_amd64/donutsalon
tar cvzf ~/tmp/donutsalon.tar.gz bin/linux_amd64/donutsalon github.com/bhs/opentracing-sandbox/donutsalon
scp -i ~/.ssh/crouton_key.pem ~/tmp/donutsalon.tar.gz ubuntu@${HOST}:
ssh -i ~/.ssh/crouton_key.pem ubuntu@${HOST} "tar xvzf donutsalon.tar.gz; sudo killall donutsalon"
ssh -i ~/.ssh/crouton_key.pem ubuntu@${HOST} "sudo nohup ./bin/linux_amd64/donutsalon --port 80 --token 32f6abfbe2ec8ef46eb55eab21c785f4 --collector_host=collector-grpc-loadtest.lightstep.com --collector_port=443 </dev/null >/dev/null 2>/dev/null &"
# ssh -i ~/.ssh/crouton_key.pem ubuntu@${HOST} "sudo nohup ./bin/linux_amd64/donutsalon --port 81 --tracer_type zipkin </dev/null >/dev/null 2>/dev/null &"
