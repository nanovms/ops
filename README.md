# Nanovms Command Line Interface

Interactive command line interface for interacting with Nanovms Unikernel. 

# Building
1. Clone the repository.
2. Install dependencies:
    - `make deps`
3. Build 
    - `make build`

# Setting up bridge network
`sudo nvm net setup`
# Package and Run your app as Unikernel
`nvm run <ELFBinary>`
# Reset bridge network
`sudo nvm net reset`