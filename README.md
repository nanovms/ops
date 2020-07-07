# OPS

[![CircleCI](https://circleci.com/gh/nanovms/ops.svg?style=svg)](https://circleci.com/gh/nanovms/ops)
[![Go Report](https://goreportcard.com/badge/github.com/nanovms/ops)](https://goreportcard.com/report/github.com/nanovms/ops)
[![](https://godoc.org/github.com/nanovms/ops?status.svg)](http://godoc.org/github.com/nanovms/ops)

<p align="center">
  <img src="https://i.imgur.com/OtfAABU.png" style="width:200px;"/>
</p>

Ops is a tool for creating and running a [Nanos](https://github.com/nanovms/nanos) unikernel. It is used to 
package, create and run your application as a [nanos](https://github.com/nanovms/nanos) unikernel instance.

Check out the [DOCS](https://nanovms.gitbook.io/ops/)

# Installation

Most users should just download the binary from the website:

## Binary install

```sh
curl https://ops.city/get.sh -sSfL | sh
```

## Build and Install from source

Building from source is easy if you have used Go before.

This program requires GO Version 1.13.x or greater.

Installing from source follows these general steps:

Install dependencies:

    - `make deps`

Build:

    - `make build`

osx notes:

```
GO111MODULE=on go build -ldflags "-w"
```
 
For [detailed instructions](https://nanovms.gitbook.io/ops/developer/prerequisites) please consult the documentation.
    
# Basic usage examples

Before learning more about `ops` it is a good idea to see some basic usage
examples. Below are links to simple examples using various programming platforms:

Let's run your first unikernel right now.

[![asciicast](https://asciinema.org/a/256914.svg)](https://asciinema.org/a/256914)

Throw this into hi.js:

```javascript
var http = require('http');
http.createServer(function (req, res) {
    res.writeHead(200, {'Content-Type': 'text/plain'});
    res.end('Hello World\n');
}).listen(8083, "0.0.0.0");
console.log('Server running at http://127.0.0.1:8083/');
```

Then you can run it like so:

```bash
ops load node_v11.5.0 -p 8083 -f -n -a hi.js
```

Want to push your app out to the cloud?

For Google: https://nanovms.gitbook.io/ops/google_cloud

For AWS: https://nanovms.gitbook.io/ops/aws

Languages:

* [Golang](https://nanovms.gitbook.io/ops/basic_usage#running-golang-hello-world)
* [PHP](https://nanovms.gitbook.io/ops/basic_usage#running-php-hello-world)
* [NodeJS](https://nanovms.gitbook.io/ops/basic_usage#running-a-nodejs-script)
* [Lua](https://github.com/nanovms/ops-examples/tree/master/lua/01-hello-world)
* [Perl](https://github.com/nanovms/ops-examples/tree/master/perl/01-hello-world)
* [Python2.7](https://github.com/nanovms/ops-examples/tree/master/python2.7)
* [Python3.6](https://github.com/nanovms/ops-examples/tree/master/python3.6/01-hello-world)
* [Ruby2.3](https://github.com/nanovms/ops-examples/tree/master/ruby/01-hello-world)
* [Rust](https://github.com/nanovms/ops-examples/tree/master/rust/01-hello-world)
* [Scheme](https://github.com/nanovms/ops-examples/tree/master/scheme/01-hello-world)
* [Forth](https://github.com/nanovms/ops-examples/tree/master/forth/01-hello-world)
* [Java](https://github.com/nanovms/ops-examples/tree/master/java/01-hello-world-example)

Applications:

* Nginx
* HAProxy
* Tarantool
* Hiawatha
* Mosquitto
* Kache
* Gnatsd
* [Wasmer](https://github.com/nanovms/ops-examples/tree/master/wasm/01-hello-world)

You can always find more pre-made packages via:

```bash
ops pkg list
```


# Build a bootable image
`ops build <app>`

# Package and run
    ops run <app>
    OR
    ops run -p <port> <app>

# Using a config file
    ops run -p <port> -c <file> <app>

# Example config file

ops config files are plain JSON, below is an example 

```JSON
  {
    "Args":["one","two"],
    "Dirs":["myapp/static"]
  }
```

## Setup networking

New users wishing to play around in a dev environment are encouraged to
use the default user-mode networking. Other production users are
encouraged to utilize native cloud builds such as [Google
Cloud](https://nanovms.gitbook.io/ops/google_cloud) which
handle networking for you.

Only advanced/power users should use the bridge networking option.

## Reporting Bugs

Feel free to open up a pull request. It's helpful to have your OPS
version and the release channel you are using.

Also - if it doesn't work on the main release you can try the nightly -
the main release can tail the nightly by many weeks sometimes.

```
ops version
```

get the release channel (or nightly)
```
ls .ops/
```

if using a package
get the package hash:
```
cat .ops/packages/manifest.json| jq '."gnatsd_1.4.1"'
```

## Pull Requests

If you have an idea for a new feature and it might take longer than a
few hours or days to do it's worth opening a feature request tkt to
ideate it first before jumping into code. There might be someone already
working on the feature or plans to do something entirely different.

## Security

[Security](https://github.com/nanovms/ops/blob/master/SECURITY.md)

Feel free to email security at.

## Support

If you are having trouble running a particular application please feel
free to open an issue and we can take a look. In general we'll only want
to support the latest release from a given application/project, however
if you really want/need support for something older there are paid
support plans available - contact the folks at https://nanovms.com .
