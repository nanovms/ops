# OPS

[![CircleCI](https://circleci.com/gh/nanovms/ops.svg?style=svg)](https://circleci.com/gh/nanovms/ops)

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

Installing from source follows three general steps:

1. Clone the repository.
2. Install dependencies:
    - `make deps`
3. Build 
    - `make build`
    
For [detailed instructions](https://nanovms.gitbook.io/ops/developer/prerequisites) please consult the documentation.
    
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
