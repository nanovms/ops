#!/bin/sh

export VERSION="0.1.25"
plat="$(uname -s | awk '{print tolower($0)}')"

GO111MODULE=on GOOS=linux go build -ldflags "-X github.com/nanovms/ops/lepton.Version=$VERSION"
gsutil cp ops gs://cli/linux

hash="ops-linux-$VERSION.md5"

if [ "$plat" = 'darwin' ]
then
  md5 -q ops > "$hash"
else
  md5sum ops | awk '{print $1}' > "$hash"
fi

gsutil cp ops gs://cli/linux/release/"$VERSION"/ops
gsutil setacl public-read gs://cli/linux/release/"$VERSION"/ops

gsutil cp "$hash" gs://cli/linux/release/"$VERSION"/"$hash"
gsutil setacl public-read gs://cli/linux/release/"$VERSION"/"$hash"

GO111MODULE=on GOOS=darwin go build -ldflags "-w -X github.com/nanovms/ops/lepton.Version=$VERSION"
gsutil cp ops gs://cli/darwin

hash="ops-darwin-$VERSION.md5"

if [ "$plat" = 'darwin' ]
then
  md5 -q ops > "$hash"
else
  md5sum ops | awk '{print $1}' > "$hash"
fi

gsutil cp ops gs://cli/darwin/release/"$VERSION"/ops
gsutil setacl public-read gs://cli/darwin/release/"$VERSION"/ops

gsutil cp "$hash" gs://cli/darwin/release/"$VERSION"/"$hash"
gsutil setacl public-read gs://cli/darwin/release/"$VERSION"/"$hash"

GO111MODULE=on GOOS=linux GOARCH=arm64 go build -ldflags "-X github.com/nanovms/ops/lepton.Version=$VERSION"
gsutil cp ops gs://cli/linux/aarch64/

hash="ops-linux-aarch64-$VERSION.md5"

if [ "$plat" = 'darwin' ]
then
  md5 -q ops > "$hash"
else
  md5sum ops | awk '{print $1}' > "$hash"
fi

gsutil cp ops gs://cli/linux/aarch64/release/"$VERSION"/ops
gsutil setacl public-read gs://cli/linux/aarch64/release/"$VERSION"/ops

gsutil cp "$hash" gs://cli/linux/aarch64/release/"$VERSION"/"$hash"
gsutil setacl public-read gs://cli/linux/aarch64/release/"$VERSION"/"$hash"

gsutil -D setacl public-read gs://cli/linux/ops
gsutil -D setacl public-read gs://cli/linux/aarch64/ops
gsutil -D setacl public-read gs://cli/darwin/ops
