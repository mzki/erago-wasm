#!/bin/bash

set -eu

outputdir="build"

while getopts o: opt; do
	case "$opt" in
		o)
			outputdir="$OPTARG"
			;;
		\?)
			echo "Usage: credits [-o OUTPUTDIR]"
			exit 1
			;;
	esac
done
shift $((OPTIND - 1))

mkdir -p $outputdir

# test depedency tools are available.
which go-licenses >/dev/null
which go > /dev/null
which curl > /dev/null

output_credit_dir=${outputdir}/credits_files
rm -rf ${output_credit_dir}

# remove own repository from credits or remove repository whose LICENSE is not found.
ignore_pkg="github.com/mzki/erago-wasm"
GOFLAGS="-tags=js,wasm" go-licenses save ./wasm \
  --ignore $ignore_pkg \
  --save_path ${output_credit_dir} \
  2>${outputdir}/credits_files_err.log 

# Additional licenses which are not found by the tool

target_dir=${output_credit_dir}/Go
mkdir -p $target_dir
echo "Get Golang license"
curl -o $target_dir/LICENSE https://go.dev/LICENSE?m=text

# finally, create CREDITS files from credits_{platform}/
output_credit () {
	local license_path=$1
	# assumes path like ./github.com/golang/groupcache/LICENSE, get github.com/golang/groupcache
	local middle_path=$(dirname ${license_path#*/})
	local repo_name=${middle_path}
	(
		echo $repo_name
		echo "================================================================================"
		cat $license_path
		echo "--------------------------------------------------------------------------------"
		echo ""
	)
}

abspath () {
	# https://qiita.com/katoy/items/c0d9ff8aff59efa8fcbb
	local arg=$1
	local abs=$(cd $(dirname $arg) && pwd)/$(basename $arg)
	echo $abs
}

target_file=$(abspath ${outputdir}/CREDITS)
(
	pushd $output_credit_dir
	output_credit ./Go/LICENSE > $target_file
	popd
)
(
	pushd ${output_credit_dir}
	for p in $(find . -type f); do output_credit $p >> $target_file; done  
	popd
)