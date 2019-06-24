#!/bin/bash
set -ex
go build main.go
ops run main -c config.json -n &
OPS_PID=$!
sleep 10
curl http://0.0.0.0:8080
kill $OPS_PID
