# Nanovms Command Line Interface

[![CircleCI](https://circleci.com/gh/nanovms/ops.svg?style=svg)](https://circleci.com/gh/nanovms/ops)

Interactive command line interface for interacting with Nanovms Unikernel. 

# Building
1. Clone the repository.
2. Install dependencies:
    - `make deps`
3. Build 
    - `make build`
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
    ## File layout on VM
        /myapp
            app
            /static
                -example.html
                /stylesheet
                    -main.css
