#!/bin/bash

set -e

hack/build.sh

mkdir -p ~/.terraform.d/plugins

mv terraform-provider-k14sx ~/.terraform.d/plugins
