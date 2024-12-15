# erago-wasm

WASM porting for [Erago](https://github.com/mzki/erago). erago-wasm assumes it runs on WebWorker and communicate messages with UI context. This architecture is inspired from  [gomeboycolor-wasm](https://github.com/djhworld/gomeboycolor-wasm)

## Build

Execute below commands. Then you can find `erago.wasm` and `wasm_exec.js` in `html/worker` directory.
You need to install Go toolchain in your environment to execute below commands.

```bash
bash scripts/build.sh
bash scripts/copy-wasm-exec-js.sh
```

## Run 

erago-wasm uses secure features, such as OPFS (Origin Private File System). So you need to launch server with TLS certification to run WASM binary built at previous section.
Once you have your certification file and its private key file, you can execute below command to launch TLS server and run sample application using erago-wasm. It will serve `index.html` and WASM files under `html`.
Preparation method of cerfication and its key is out of scope.

```bash
go run server.go -certfile "your-certfile" -keyfile "your-keyfile"
# then visit https://localhost/html to show how application runs on browser.
```

## Embedding in your app

You can create your `html/worker/engine_worker.js` to communicate with WASM binary on WebWorker. Of course you can use `engine_worker.js` as is if default one is enough.
Make sure WebWorker related files, which are under `html/worker`, should be placed at same directory. 

Then you should add code to launch WebWorker using `engine_worker.js` into your script, and communicate with the worker to archieve complete application. 

