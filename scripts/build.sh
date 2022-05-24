#!/bin/bash
#
# Builds the app with the latest commit hash (short)
# E.g.: ./app -v --> APPNAME 75316fe

DIR="$( cd -- "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"

commit_hash=$(git rev-parse --short HEAD)
go build -ldflags="-s -w -X github.com/gohornet/inx-indexer/core/app.Version=$commit_hash"
