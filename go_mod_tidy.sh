#!/bin/bash

gci write --skip-generated -s default *
gofumpt -d -e -extra -l -w .
go mod tidy
