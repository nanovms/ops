# What is a Unikernel?

A unikernel is a single-process system that is specifically designed to
run only one application. It is a great fit for modern cloud
environments because of its performance, security, and size.

## Single Process

Linux systems have archaic design concepts rooted in the 1960s from when
machines cost half a million dollars and needed to run multiple programs
for multiple users. Today, developers go out of their way to isolate
programs from each other if for no other reason than manageability
concerns.

A unikernel embraces the single-process concept while allowing the use
of multiple threads. For languages such as Go, this fits well. For
interpreted languages such as Ruby and Python, developers in these
languages typically load balance a set of application servers to enable
a greater degree of concurrency. In the unikernel world, we do the same
thing but those app servers become full-fledged VMs and can make use of
existing load balancers without having to do backflips.

## No Shell/No Users

This is a security design constraint. There is no shell to remotely log
into and there is no concept of users. While OPS has a stubbed/fake user
it is only present to implement underlying libc calls and has no
relevance otherwise. This also means that modern Unix permissions don't
have much meaning inside of a unikernel because there is only one
program running and no users.
