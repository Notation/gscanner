#!/usr/bin/env bash
set -ex
gobin=`which go`
if [ -z "$gobin" ]; then
gobin="/usr/local/go/bin/go"
fi

gitbin=`which git`
if [ -z "$gitbin" ]; then
gitbin="/usr/bin/git"
fi

VERSION=$($gitbin rev-parse --short HEAD)
BRANCH=$($gitbin rev-parse --abbrev-ref HEAD)
BUILDTIME="$($gitbin log -1 --format=%cI)"
BUILDER=$($gitbin config user.name)

GOLDFLAGS="-X main.BuildBranch=$BRANCH"
GOLDFLAGS+=" -X main.BuildVersion=$VERSION"
GOLDFLAGS+=" -X main.BuildTime=$BUILDTIME"
GOLDFLAGS+=" -X main.Builder=$BUILDER"
#GOLDFLAGS+=' -extldflags "-static"'

CGO_ENABLED=1 $gobin build -buildmode=pie -ldflags "$GOLDFLAGS" "$@"
