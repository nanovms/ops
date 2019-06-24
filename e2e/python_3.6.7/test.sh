#!/bin/bash
set -ex
ops load python_3.6.7 -n -c config.json &
OPS_PID=$!
sleep 10
curl http://0.0.0.0:8000
kill $OPS_PID
