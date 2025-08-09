// Copyright 2024 The erago-wasm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

importScripts("wasm_exec.js");

if (!WebAssembly.instantiateStreaming) { // polyfill
    WebAssembly.instantiateStreaming = async (resp, importObject) => {
        const source = await (await resp).arrayBuffer();
        return await WebAssembly.instantiate(source, importObject);
    };
}

var go = new Go();
let mod, inst;
WebAssembly.instantiateStreaming(fetch("erago.wasm"), go.importObject).then((result) => {
    mod = result.module;
    inst = result.instance;
}).catch((err) => {
    console.error(err);
});

async function runGoApp() {
    //console.clear();
    await go.run(inst);
    go = new Go(); // reset instance
    inst = await WebAssembly.instantiate(mod, go.importObject); // reset instance
}

self.addEventListener("message", (ev) => {
    let data = ev.data;
    if (data[0] == "run_engine_worker") {
        runGoApp();
    }
}, false);

/*
    Events published from Engine.

	EngineOnPublishBytes EngineCallbackID = iota
	EngineOnPublishBytesTemporary
	EngineOnRemove
	EngineOnRemoveAll

	EngineOnCommandRequested
	EngineOnInputRequested
	EngineOnInputRequestClosed

	EngineNotifyQuit
*/

// entry: {eventName, messageType, [arg]}
for (const entry of [
    ["EngineOnPublishBytes", "addParagraph"],
    ["EngineOnPublishBytesTemporary", "addParagraph"],
    ["EngineOnRemove", "removeParagraph"],
    ["EngineOnRemoveAll", "removeParagraph", -1],
	["EngineOnCommandRequested", "inputStatus", "commandRequested"],
	["EngineOnInputRequested", "inputStatus", "inputRequested"],
	["EngineOnInputRequestClosed", "inputStatus", "inputRequestClosed"],
]) {
    if (entry[2] === undefined) {
        // Use ev.detail directlly as postMessage arg 
        if (entry[1] == "addParagraph") {
            self.addEventListener(entry[0], (ev) => {
                let transferrables = [ev.detail[0].buffer]; // for zero copy, move sematics for large binary.
                self.postMessage(["engineEvent", [entry[1], ev.detail[0]]], transferrables)
            })
        } else {
            self.addEventListener(entry[0], (ev) => {
                self.postMessage(["engineEvent", [entry[1], ev.detail[0]]], transferrables)
            })
        }
    } else {
        // Use entry constant as postMessage arg 
        self.addEventListener(entry[0], (ev) => {
            self.postMessage(["engineEvent", [entry[1], entry[2]]])
        })
    }
}

self.addEventListener("EngineNotifyQuit", (ev) => {
    // null(means no error) treated as empty string to be consitent with string type.
    let msg = (ev.detail[0]) ? ev.detail[0] : "";
    self.postMessage(["engineEvent", ["notifyQuit", msg]]);
})

self.postMessage(["engine_worker loaded!"])