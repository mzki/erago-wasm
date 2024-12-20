//go:build js && wasm
// +build js,wasm

package main

import (
	"errors"
	"fmt"
	"syscall/js"
)

func SendBackStatusLaunchOK() {
	postMessage("enginestatus", []any{"launchOK", true})
}

func SendBackStatusPathSelected(rootPath string) {
	postMessage("enginestatus", []any{"pathSelected", rootPath})
}

func SendBackStatusPathInvalid(rootPath string) {
	postMessage("enginestatus", []any{"pathInvalid", rootPath})
}

func SendBackStatusEngineStartOK() {
	postMessage("enginestatus", []any{"engineStartOK", true})
}

func SendBackStatusEngineStartNG(err error) {
	postMessage("enginestatus", []any{"engineStartNG", fmt.Errorf("engine start Fail: %w", err).Error()})
}

func SendBackStatusEndsApp() {
	postMessage("enginestatus", []any{"endsApp", true})
}

func SendBackInstalledPath(methodName, installedPath string) {
	postMessage("methodResult", []any{methodName, installedPath})
}

func SendBackLogBytes(methodName string, bs js.Value) {
	postMessage("methodResult", []any{methodName, bs})
}

func SendBackSavZipBytes(methodName string, bs js.Value) {
	postMessage("methodResult", []any{methodName, bs})
}

func SendBackMethodOK(methodName string) {
	postMessage("methodResult", []any{methodName, true})
}

func SendBackMethodNG(methodName string) {
	postMessage("methodResult", []any{methodName, false})
}

func SendBackMethodError(methodName string, err error) {
	postMessage("methodError", []any{methodName, fmt.Errorf("%s Error: %w", methodName, err).Error()})
}

var ErrNotImplemented = errors.New("not implemented")

func SendBackMethodNotImplemented(methodName string) {
	SendBackMethodError(methodName, ErrNotImplemented)
}

func postMessage(action string, value any) {
	js.Global().Get("self").Call("postMessage", []any{action, value})
}
