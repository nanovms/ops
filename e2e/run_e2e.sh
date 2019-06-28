#!/bin/bash
TESTS="go
python_3.6.7
ruby_2.5.1
nginx_1.15.6
node_v11.5.0"

if [ -f /tmp/e2erun.log ] ; then
    rm /tmp/e2erun.log
fi

for test in $TESTS
do
    cd $test
    ./test.sh >>/tmp/e2erun.log 2>&1
    if [ $? -eq 0 ] ; then
        echo "$test: PASSED"
    else
        echo "$test: FAILED"
    fi
    cd ..
done
