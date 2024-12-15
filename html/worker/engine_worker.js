importScripts("wasm_exec.js");

if (!WebAssembly.instantiateStreaming) { // polyfill
    WebAssembly.instantiateStreaming = async (resp, importObject) => {
        const source = await (await resp).arrayBuffer();
        return await WebAssembly.instantiate(source, importObject);
    };
}

const go = new Go();
let mod, inst;
WebAssembly.instantiateStreaming(fetch("erago.wasm"), go.importObject).then((result) => {
    mod = result.module;
    inst = result.instance;
}).catch((err) => {
    console.error(err);
});

async function runGoApp() {
    console.clear();
    await go.run(inst);
    inst = await WebAssembly.instantiate(mod, go.importObject); // reset instance
}

self.addEventListener("message", (ev) => {
    let data = ev.data;
    if (data[0] == "run_engine_worker") {
        runGoApp();
    }
}, false);

/*
	EngineOnPublishJson EngineCallbackID = iota
	EngineOnPublishJsonTemporary
	EngineOnRemove
	EngineOnRemoveAll

	EngineOnCommandRequested
	EngineOnInputRequested
	EngineOnInputRequestClosed

	EngineNotifyQuit
*/
self.addEventListener("EngineOnPublishJson", (ev) => {
    self.postMessage(["engineEvent", ["addParagrah", ev.detail[0]]]);
})

self.addEventListener("EngineNotifyQuit", (ev) => {
    self.postMessage(["engineEvent", ["notifyQuit", ev.detail[0]]]);
})

self.postMessage(["engine_worker loaded!!!"])