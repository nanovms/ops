-dlopen
    compile : `cc -o dlopen dlopen.c -ldl`
    run: num run -p 8080 -c config.json dlopen
    config.json
       `{ 
            "Files":["/lib/libm.so.6"],
            "Env" : {}
        }`