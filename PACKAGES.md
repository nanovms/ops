# Packages

A package is a way of distributing a given application with it's
dependencies. All of this functionality is present in OPS proper but the
package interface allows users to distribute their packages as a bundled
resource without having to compile or add missing dependencies.

Sometimes this is OS related and sometimes it's application specific but
for OPS purpose there's no difference.

Think of common software that you would ```sudo apt-get install``` or
web frameworks with native dependencies or something of that sort - it's
not intended for developer specific in-house applications.

Ready to create your own package?

### Create from Docker:

You can create a package from Docker like so:

```
ops pkg from-docker node:16.3.0 -f node
```

Or you can create one manually:

### Create Directory

For example if we want to make a package for Lua 5.2.4 we'd have the
following:

```
export PKGNAME=lua
export PKGVERSION=5.2.4

mkdir "$PKGNAME"_"$PKGVERSION"
```

### Populate it

For example:

```
eyberg@s1:~/plz/lua_5.2.4$ tree
.
├── lua
├── package.manifest
└── sysroot
    ├── lib
    │   └── x86_64-linux-gnu
    │       ├── libc.so.6
    │       ├── libdl.so.2
    │       ├── libm.so.6
    │       ├── libreadline.so.6
    │       └── libtinfo.so.5
    └── lib64
        └── ld-linux-x86-64.so.2

4 directories, 8 files
```

Your package.manifest should look something like this:

```
{
   "Program":"lua_5.2.4/lua",
   "Args" : ["lua"],
   "Version":"5.2.4"
}
```

In many cases this is a dump from ldd:

```
eyberg@s1:~/plz/lua_5.2.4$ ldd lua
        linux-vdso.so.1 =>  (0x00007ffd18bf3000)
        libreadline.so.6 => /lib/x86_64-linux-gnu/libreadline.so.6 (0x00007f74e8836000)
        libm.so.6 => /lib/x86_64-linux-gnu/libm.so.6 (0x00007f74e852d000)
        libdl.so.2 => /lib/x86_64-linux-gnu/libdl.so.2 (0x00007f74e8329000)
        libc.so.6 => /lib/x86_64-linux-gnu/libc.so.6 (0x00007f74e7f5f000)
        libtinfo.so.5 => /lib/x86_64-linux-gnu/libtinfo.so.5 (0x00007f74e7d36000)
        /lib64/ld-linux-x86-64.so.2 (0x00007f74e8a7c000)
```

Don't forget libnss, libresolv and friends. This all assumes you are
using libc which will be necessary for ordinary applications.

```
/lib/x86_64-linux-gnu/libnss_dns.so.2
```

```
/etc/ssl/certs
```

### Tar it up

The name needs to reflect this format:

```
tar czf "$PKGNAME"_"$PKGVERSION".tar.gz "$PKGNAME"_"$PKGVERSION"
```

### Upload

Now you are ready to upload it. If you don't have an account you can
create one for free at https://repo.ops.city. Just sign in with
your github account.

You can upload it via the web interface at https://repo.ops.city or you
can use the shell:

```
ops pkg login <api_key>
ops pkg push <my_package>
```
