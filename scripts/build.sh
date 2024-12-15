GOOS=js GOARCH=wasm go build -ldflags "-s -w -X main.APPNAME=erago-wasm -X main.VERSION=v0.0.0" -o erago.wasm ./wasm
mv erago.wasm html/worker/