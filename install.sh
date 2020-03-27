#!/bin/bash

set -e

go build .

mkdir -p ~/.terraform.d/plugins

mv terraform-provider-k14s ~/.terraform.d/plugins
