#!/bin/bash
ROOT=$(pwd)
sh -c "cd go/src/app && env GOPATH=${ROOT}/go dep ensure"
env GOOS=linux GOARCH=amd64 GOPATH=${ROOT}/go go build -v -o donutbin app
