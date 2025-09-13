#!/bin/bash
VERSION=$(git describe --tags)
echo Building version $VERSION
go build -ldflags "-X github.com/ThatBathroom/yagpdb/v2/common.VERSION=${VERSION}"