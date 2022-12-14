# OPS

[![CircleCI](https://circleci.com/gh/nanovms/ops.svg?style=svg)](https://circleci.com/gh/nanovms/ops)
[![Go Report](https://goreportcard.com/badge/github.com/nanovms/ops)](https://goreportcard.com/report/github.com/nanovms/ops)
[![Go Docs](https://godoc.org/github.com/nanovms/ops?status.svg)](http://godoc.org/github.com/nanovms/ops)

<p align="center">
  <img src="https://i.imgur.com/OtfAABU.png" style="width:200px;"/>
</p>

Ops is a tool for creating and running a [Nanos](https://github.com/nanovms/nanos) unikernel. It is used to
package, create, and run your application as a [nanos](https://github.com/nanovms/nanos) unikernel instance.

1. [Installation](#installation)
2. [Hello World](#hello-world)
3. [Cloud](#cloud)
4. [Support](#support)

Check out the [DOCS](https://nanovms.gitbook.io/ops/).

# Installation

Most users should just download the binary from the website:

## Binary install

```sh
curl https://ops.city/get.sh -sSfL | sh
```

If you don't like this option you can also download pre-made packages
for various systems [here](https://ops.city/downloads) and you can also
build from source.

## Debian / Redhat:

Add a deb src:

```
sudo vi /etc/apt/sources.list.d/fury.list
```

```
deb [trusted=yes] https://apt.fury.io/nanovms/ /
```

Update your sources && install:

```
sudo apt-get update && sudo apt-get install ops
```

## Build and Install from source

Building from source is easy if you have used Go before.

This program requires GO Version 1.13.x or greater.

Installing from source follows these general steps:

Install dependencies:

```sh
make deps
```

Build:

```sh
make build
```

macOS notes:

```sh
GO111MODULE=on go build -ldflags "-w"
```

For [detailed instructions](https://nanovms.gitbook.io/ops/developer/prerequisites), please consult the documentation.

# Hello World

Before learning more about `ops` it is a good idea to see some basic usage
examples. Below are links to simple examples using various programming platforms:

Let's run your first unikernel right now.

[![asciicast](https://asciinema.org/a/256914.svg)](https://asciinema.org/a/256914)

Throw this into hi.js:

```javascript
const http = require('http');
http.createServer((req, res) => {
    res.writeHead(200, { 'Content-Type': 'text/plain' });
    res.end('Hello World\n');
}).listen(8083, "0.0.0.0");
console.log('Server running at http://127.0.0.1:8083/');
```

Then you can run it like so:

```sh
ops pkg load eyberg/node:v16.5.0 -p 8083 -n -a hi.js
```

Note: Since the node package is inside the unikernel you do not need to
install node locally to use it.

# Cloud

Want to push your app out to the cloud? No complex orchestration like
K8S is necessary. OPS pushes all the orchestration onto the cloud
provider of choice so you don't need to manage anything. Be sure to try
this out as the next step after running a hello world locally as it will
answer many questions you might have.

- [Azure](https://docs.ops.city/ops/azure)
- [AWS](https://docs.ops.city/ops/aws)
- [Google Cloud](https://docs.ops.city/ops/google_cloud)

- [Digital Ocean](https://docs.ops.city/ops/digital_ocean)
- [Vultr](https://docs.ops.city/ops/vultr)
- [UpCloud](https://docs.ops.city/ops/upcloud)

You can find many more pre-made packages at the public repo:

[https://repo.ops.city/](https://repo.ops.city/)

Or via the shell:

```sh
ops pkg list
```

You can also upload your own with a free account.

Languages:

Various langauge examples can be found at
[https://github.com/nanovms/ops-examples](https://github.com/nanovms/ops-examples).
In general [https://nanos.org](Nanos) supports any languages and is not
language specific.

You can find more examples and tutorial on youtube as well:

[https://www.youtube.com/channel/UC3mqDqCVu3moVKzmP2YNmlg](https://www.youtube.com/channel/UC3mqDqCVu3moVKzmP2YNmlg)

## Apple M1/M2 Users

The Apple M1 and M2 are ARM based. OPS is built for users primarily
deploying to x86 based servers. While you can certainly run ARM builds
with Nanos and OPS be aware that if you are trying to run x86 builds
(the default) on ARM based M1s you won't be able to use hardware
acceleration for x86.

# Build a bootable image
`ops build <app>`

# Package and run
```sh
ops run <app>
# or
ops run -p <port> <app>
```

# Using a config file
```sh
ops run -p <port> -c <file> <app>
```

# Use golang string interoplation in config files
To enable set `ops_render_config` to `true`. Both `${ENV_VAR}` and `$ENV_VAR` are supported.

## Example Command
```sh
ops_render_config=true ops run -p <port> -c <file> <app>
# or
export ops_render_config=true
ops run -p <port> -c <file> <app>
```

## Example config
```JSON
{
  "Args":[
    "--user",
    "${USER}",
    "--password",
    "$PASSWORD"
  ],
  "Dirs":["myapp/static"]
}
```

# Example config file

OPS config files are plain JSON, below is an example.

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

## Useful Ops Environment Variables

The following environment variables are available to you

* `ops_render_config` - Set to `true` to use Golang ENV var interpolation to render your JSON configs.


## Reporting Bugs

Feel free to open up a pull request. It's helpful to have your OPS
version and the release channel you are using.

Also, if it doesn't work on the main release, then you can try the nightly.
The main release can tail the nightly by many weeks sometimes.

```sh
ops version
```

If you are using a package, get the package hash:

```sh
jq '."gnatsd_1.4.1"' ~/.ops/packages/manifest.json
```

## Pull Requests

If you have an idea for a new feature and it might take longer than a
few hours or days to complete, then it's worth opening a feature request ticket to
ideate it first before jumping into code. There might be someone already
working on the feature or plans to do something entirely different.

## Security

[Security](https://github.com/nanovms/ops/blob/master/SECURITY.md)

Feel free to email security[at]nanovms[dot]com.

# Support

If you are having trouble running a particular application please feel
free to open an issue and we can take a look. In general we'll only want
to support the latest release from a given application/project, however,
if you really want/need support for something older there are paid
support plans available - contact the folks at https://nanovms.com.

