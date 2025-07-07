#!/bin/bash
VERSION=$(git describe --tags)
echo Building version $VERSION
go build -ldflags "-X github.com/ThatBathroom/yagpdb/common.VERSION=${VERSION}"