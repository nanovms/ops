#!/bin/bash
set -ex
mkdir -p .ruby
export GEM_HOME=.ruby
gem install sinatra --no-rdoc --no-ri
ops load ruby_2.5.1 -n -c config.json &
OPS_PID=$!
sleep 10
curl http://0.0.0.0:4567
kill $OPS_PID
