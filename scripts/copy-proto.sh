set -e

outputdir=${1:-./proto}

set -ux

modcache=$(go env GOMODCACHE)
pkg=$(go mod graph | grep "github.com/mzki/erago@" | tail -n 1 | cut -d " " -f 1)
pkgroot=$modcache/$pkg
protofile=$pkgroot/view/exp/text/pubdata/pubdata.proto

cp $protofile $outputdir