#!/bin/bash

set -eu

outputdir="build/archive"
version_for_file="v0.0.0"

while getopts "o:v:" opt; do
	case "$opt" in
		o)
			outputdir="$OPTARG"
			;;
		v)
			version_for_file="$OPTARG"
			;;
		\?)
			echo "Usage: archive [-o OUTPUTDIR] [-v VERSION_FOR_FILE] [EXTRA_FILES_ARCHIVED...]"
			exit 1
			;;
	esac
done
shift $((OPTIND - 1))

extrafiles="$@"


target_filename=erago-wasm-$version_for_file
target_dir=$outputdir/$target_filename
mkdir -p $target_dir
cp -t $target_dir LICENSE README.md $extrafiles
cp -r html $target_dir
(
    pushd $outputdir
    zip -r $target_filename.zip $target_filename
    popd
)
echo "Output $outputdir/$target_filename.zip"