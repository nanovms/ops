#!/bin/bash
set -ex
ops load nginx_1.15.6 -n -c config.json &
OPS_PID=$!
sleep 10
curl http://0.0.0.0:8084
kill $OPS_PID
