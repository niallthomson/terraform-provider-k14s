#!/bin/bash

set -e -x -u

go fmt ./k14s/...

go build -o terraform-provider-k14sx .

echo SUCCESS