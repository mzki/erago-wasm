//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"strings"
	"syscall/js"
)

var (
	APPNAME     string
	VERSION     string
	COMMIT_HASH string
)

func main() {
	fmt.Printf("---------- Start %s-%s-%s ----------\n", APPNAME, VERSION, COMMIT_HASH)
	SendBackStatusAppLaunchOK()
	defer func() {
		fmt.Printf("---------- End %s-%s-%s ----------\n", APPNAME, VERSION, COMMIT_HASH)
		SendBackStatusAppShutdown()
	}()

	const rootDir = "/erago-wasm"
	store := NewWebFilesystem(rootDir)

	var initResult engineInitResult
	{
		initResultCh, cancelInitEngine := AwaitInitEngineWithPath(store, rootDir)
		cancelRunPkg := RunPackager(store, rootDir)
		cancelNotImpl := RunNotImplemented()
		SendBackStatusWaitForEngineInit()

		initResult = <-initResultCh

		cancelInitEngine()
		cancelRunPkg()
		cancelNotImpl()
	}
	defer initResult.quitFunc()

	waitRunEngine := AwaitRunEngine()
	cancelRunIO := RunIO()
	defer cancelRunIO()
	cancelNotImpl := RunNotImplemented()
	defer cancelNotImpl()
	SendBackStatusEngineInitOK(initResult.rootPath)

	<-waitRunEngine
	RunEngine(initResult.messenger)
	SendBackStatusEngineStartOK()

	<-initResult.messenger.Done()
}

type engineInitResult struct {
	messenger *uiMessenger
	quitFunc  func()
	rootPath  string
}

func AwaitInitEngineWithPath(
	store *WebFileSystem,
	rootDir string,
) (
	resultChan <-chan engineInitResult,
	cancelFunc func(),
) {
	result := make(chan engineInitResult)
	resultChan = result

	var callback js.Func
	cancelFunc = func() {
		js.Global().Get("self").Call("removeEventListener", "message", callback)
		callback.Release()
		close(result)
	}
	callback = js.FuncOf(func(this js.Value, args []js.Value) any {
		data := args[0].Get("data")
		switch methodName := data.Index(0).String(); methodName {
		case "init_engine_with_path":
			ConsumeMessageEvent(args[0])
			rootPath := data.Index(1).String()
			if !strings.HasPrefix(rootPath, rootDir) {
				SendBackMethodError(methodName, fmt.Errorf("selected path(%s) should be under %s", rootPath, rootDir))
				return nil
			}
			go func() { // to avoid blocking js eventLoop
				rootPathStore, err := store.Sub(rootPath, false)
				if err != nil {
					SendBackMethodError(methodName, err)
					return
				}
				messenger, quitFunc, err := InitEngine(rootPath, rootPathStore)
				if err != nil {
					SendBackMethodError(methodName, err)
					return
				}
				result <- engineInitResult{
					messenger: messenger,
					quitFunc:  quitFunc,
					rootPath:  rootPath,
				}
				SendBackMethodOK(methodName)
			}()
		}
		return nil
	})
	js.Global().Get("self").Call("addEventListener", "message", callback, false)
	return
}

func AwaitRunEngine() <-chan struct{} {
	runEngine := make(chan struct{})

	var callback js.Func
	cancelFunc := func() {
		js.Global().Get("self").Call("removeEventListener", "message", callback)
		callback.Release()
		close(runEngine)
	}
	callback = js.FuncOf(func(this js.Value, args []js.Value) any {
		data := args[0].Get("data")
		switch methodName := data.Index(0).String(); methodName {
		case "start_engine":
			ConsumeMessageEvent(args[0])
			go func() {
				runEngine <- struct{}{}
				SendBackMethodOK(methodName)
				cancelFunc()
			}()
		}
		return nil
	})
	js.Global().Get("self").Call("addEventListener", "message", callback, false)
	return runEngine
}

func RunNotImplemented() (cancelFunc func()) {
	var callback js.Func
	cancelFunc = func() {
		js.Global().Get("self").Call("removeEventListener", "message", callback)
		callback.Release()
	}
	callback = js.FuncOf(func(this js.Value, args []js.Value) any {
		data := args[0].Get("data")
		methodName := data.Index(0).String()
		ConsumeMessageEvent(args[0])
		SendBackMethodNotImplemented(methodName)
		return nil
	})
	js.Global().Get("self").Call("addEventListener", "message", callback, false)
	return
}
