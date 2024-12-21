//go:build js && wasm
// +build js,wasm

package main

import "syscall/js"

var jsErrNull = js.Error{Value: js.Null()}

func ToGoBytes(jsBs js.Value) []byte {
	bs := make([]byte, jsBs.Length())
	js.CopyBytesToGo(bs, jsBs)
	return bs
}

func ToJsBytes(bs []byte) js.Value {
	jsBs := js.Global().Get("Uint8Array").New(len(bs))
	js.CopyBytesToJS(jsBs, bs)
	return jsBs
}

func JsOptions(opt map[string]any) js.Value {
	jsOpt := js.Global().Get("Object").New()
	for k, v := range opt {
		jsOpt.Set(k, js.ValueOf(v))
	}
	return jsOpt
}

func JsArrayOf(args ...any) js.Value {
	return js.Global().Get("Array").Call("of", args...)
}

func ConsumeMessageEvent(ev js.Value) {
	ev.Call("stopImmediatePropagation")
}
