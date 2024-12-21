//go:build js && wasm
// +build js,wasm

package main

import (
	"errors"
	"fmt"
	"syscall/js"
)

func SendBackStatusAppLaunchOK() {
	postMessage("engineStatus", []any{"appLaunchOK", true})
}

func SendBackStatusWaitForEngineInit() {
	postMessage("engineStatus", []any{"appWaitForEngineInit", true})
}

func SendBackStatusEngineInitOK(rootPath string) {
	postMessage("engineStatus", []any{"appEngineInitOK", rootPath})
}

func SendBackStatusEngineStartOK() {
	postMessage("engineStatus", []any{"appEngineStartOK", true})
}

func SendBackStatusAppShutdown() {
	postMessage("engineStatus", []any{"appShutdown", true})
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

func SendBackStringWidth(methodName string, width int32) {
	postMessage("methodResult", []any{methodName, int(width)})
}

func SendBackMethodOK(methodName string) {
	postMessage("methodResult", []any{methodName, true})
}

func SendBackMethodNG(methodName string) {
	postMessage("methodResult", []any{methodName, false})
}

func SendBackMethodError(methodName string, err error) {
	postMessage("methodError", []any{methodName, fmt.Errorf("%s: Error: %w", methodName, err).Error()})
}

var ErrNotImplemented = errors.New("not implemented")

func SendBackMethodNotImplemented(methodName string) {
	SendBackMethodError(methodName, ErrNotImplemented)
}

func postMessage(action string, value any) {
	js.Global().Get("self").Call("postMessage", []any{action, value})
}
