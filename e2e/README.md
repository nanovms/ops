# E2E tests
Simple E2E tests with Ops CLI for different packages.

## Running
```
$ ~/go/src/github.com/nanovms/ops
$ dep ensure
$ go build && go install
$ cd e2e && run_e2e.sh
```

## Sample Run
For any failures, check /tmp/e2erun.log

```
$ ./run_e2e.sh 
go: PASSED
python_3.6.7: PASSED
ruby_2.5.1: PASSED
nginx_1.15.6: FAILED
node_v11.5.0: PASSED
```

