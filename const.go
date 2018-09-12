package main

// file system manifest
const manifest string = `(
    #64 bit elf to boot from host
    children:(kernel:(contents:(host:%v))
              #user program
              hws:(contents:(host:%v)))
    # filesystem path to elf for kernel to run
    program:/%v
    fault:t
    arguments:[webg poppy]
    environment:(USER:bobby PWD:/)
)`

// boot loader
const bootImg string = "staging/boot"
// kernel
const kernelImg string = "staging/stage3"
// kernel + ELF image
const mergedImg string = "tempimage"
// final bootable image
const finalImg string = "image"

const bucketBaseUrl string = "https://storage.googleapis.com/uniboot/%v"
