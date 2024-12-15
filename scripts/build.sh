#!/bin/bash

set -eu

appname="erago-wasm"
version="v0.0.0"
commit_hash="none"

while getopts "v:c:" opt; do
	case "$opt" in
		v)
			version="$OPTARG"
			;;
		c)
			commit_hash="$OPTARG"
			;;
		\?)
			echo "Usage: build [-v VERSION] [-c COMMIT_HASH]"
			exit 1
			;;
	esac
done
shift $((OPTIND - 1))


GOOS=js GOARCH=wasm go build -ldflags "-s -w -X main.APPNAME=$appname -X main.VERSION=$version -X main.COMMIT_HASH=$commit_hash" -o erago.wasm ./wasm
mv erago.wasm html/worker/