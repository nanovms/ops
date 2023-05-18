#!/bin/sh

gon config.hcl
gon qemu.hcl

/usr/local/bin/packagesbuild -v ops-d.pkgproj

# FIXME - add version extraction
gsutil cp build/ops-d.pkg gs://cli/darwin/release/0.1.43/ops-installer.pkg
gsutil setacl public-read gs://cli/darwin/release/0.1.43/ops-installer.pkg
