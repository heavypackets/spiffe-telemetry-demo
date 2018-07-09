#!/bin/bash

env GOOS=linux GOARCH=amd64 GOPATH=$(pwd)/go go build -v -o donutbin app && \
docker-compose build
