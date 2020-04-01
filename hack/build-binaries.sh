#!/bin/bash

set -e -x -u

if [ -z "$VERSION" ]; then
  echo "Error: Must set VERSION environment variable"
  exit 1
fi

go build .

rm -rf tmp/binaries
mkdir -p tmp/binaries

(
	set -e

	cd tmp/binaries/
	mkdir {darwin_amd64,linux_amd64,windows_amd64}

	# makes builds reproducible
	export CGO_ENABLED=0
	repro_flags="-ldflags=-buildid= -trimpath"

	GOOS=darwin GOARCH=amd64 go build $repro_flags \
		-o darwin_amd64/terraform-provider-k14sx_${VERSION} ../..
	GOOS=linux GOARCH=amd64 go build $repro_flags \
		-o linux_amd64/terraform-provider-k14sx_${VERSION} ../..
	GOOS=windows GOARCH=amd64 go build $repro_flags \
		-o windows_amd64/terraform-provider-k14sx_${VERSION} ../..

	COPYFILE_DISABLE=1 tar czvf ../terraform-provider-k14sx-binaries.tgz .
)

shasum -a 256 tmp/terraform-provider-k14sx-binaries.tgz