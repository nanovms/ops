#!/bin/bash
set -ex
ops load node_v11.5.0 -n -c config.json &
OPS_PID=$!
sleep 10
curl http://0.0.0.0:8083
kill $OPS_PID
