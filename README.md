# OPS

[![CircleCI](https://circleci.com/gh/nanovms/ops.svg?style=svg)](https://circleci.com/gh/nanovms/ops)

[![Go Report](https://goreportcard.com/badge/github.com/nanovms/ops)](https://goreportcard.com/badge/github.com/nanovms/ops)

Ops is the main interface for creating and running a Nanos unikernel. It is used to 
package, create and run your application as a nanos unikernel instance.

Check out the [DOCS](https://nanovms.gitbook.io/ops/)

### `ops <command> [flags] [ARG]`

# Installation

## Binary install

```sh
curl https://ops.city/get.sh -sSfL | sh
```

## Install from source

This program requires GO Version 1.10.x or greater.

Installing from source follows three general steps:

1. Clone the repository.
2. Install dependencies:
    - `make deps`
3. Build 
    - `make build`
    
For [detailed instructions](https://nanovms.gitbook.io/ops/developer/prerequisites) please consult the documentation.
    
# Basic usage examples

Before learning more about `ops` it is a good idea to see some basic usage
examples. Below are links to simple examples using various programming platforms:

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

# Setup networking

## bridge network 
`sudo ops net setup` 

## reset
`sudo ops net reset`

# Build a bootable image
`ops build <ELFBinary>`

# Package and run
    ops run <ELFBinary>
    OR
    ops run -p <port> <ELFBinary>

# Using a config file
    ops run -p <port> -c <file> <ELFBinary>

# Example config file

ops config files are plain JSON, below is an example 

    {
        "Args":["one","two"],
        "Dirs":["myapp/static"]
    }

    ## File layout on local host machine 
        -myapp
            app
            -static
                -example.html
                -stylesheet 
                    -main.css
    ## File  layout on VM
        /myapp
            app
            /static
                -example.html
                /stylesheet
                    -main.css

## Reporting Bugs

Feel free to open up a pull request. It's helpful to have your OPS
version and the release channel you are using.

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

Feel free to email security at.
