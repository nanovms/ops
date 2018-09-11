# Nanovms Command Line Interface

Interactive command line interface for interacting with Nanovms Unikernel. 

# Building
Clone the repository.

First, you will need to install dependencies:
    make deps
The build 
    make build

# Binary dependencies [Temp]
Right now you need to manually download  the files from https://console.cloud.google.com/storage/browser/uniboot
and place `mkfs` in same directory as `nvm`.
Both `boot` and `stage3` should be in directory `staging` 
```bash
├── nvm
    ├── ./nvm
    ├── ./mkfs 
    ├── staging
        ├── boot
        ├── stage3
```
# Package and Run your app as Unikernel
nvm run `<ELFBinary>`