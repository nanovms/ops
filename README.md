# Nanovms Command Line Interface

[![CircleCI](https://circleci.com/gh/nanovms/nvm.svg?style=svg)](https://circleci.com/gh/nanovms/nvm)

Interactive command line interface for interacting with Nanovms Unikernel. 

# Building
1. Clone the repository.
2. Install dependencies:
    - `make deps`
3. Build 
    - `make build`
# Setup networking
## bridge network 
`sudo nvm net setup` 
## reset
`sudo nvm net reset`
# Build a bootable image
`nvm build <ELFBinary>`
# Package and run
    nvm run <ELFBinary>
    OR
    nvm run -p <port> <ELFBinary>
# Using a config file
    nvm run -p <port> -c <file> <ELFBinary>
# Example config file
    nvm config files are plain JSON, below is an example 
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
    


