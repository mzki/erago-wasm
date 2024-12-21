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

	var rootPath string
	{
		cancelRunPkg := RunPackager(store, rootDir)
		pathCh, cancelPathSelect := AwaitPathSelect()
		cancelNotImpl := RunNotImplemented()
		for {
			rootPath = <-pathCh
			if !strings.HasPrefix(rootPath, rootDir) {
				SendBackStatusPathInvalid(rootPath)
				continue
			}
			break
		}
		SendBackStatusPathSelected(rootPath)
		cancelRunPkg()
		cancelPathSelect()
		cancelNotImpl()
	}

	rootPathStore, err := store.Sub(rootPath, false)
	if err != nil {
		SendBackStatusEngineInitNG(err)
		return
	}
	messenger, quitFunc, err := InitEngine(rootPath, rootPathStore)
	if err != nil {
		SendBackStatusEngineInitNG(err)
		return
	}
	defer quitFunc()

	waitRunEngine := AwaitRunEngine()
	cancelRunIO := RunIO()
	defer cancelRunIO()
	cancelNotImpl := RunNotImplemented()
	defer cancelNotImpl()
	SendBackStatusEngineInitOK()

	<-waitRunEngine
	RunEngine(messenger)
	SendBackStatusEngineStartOK()

	<-messenger.Done()
}

func AwaitPathSelect() (path <-chan string, cancelFunc func()) {
	selectPath := make(chan string)

	var callback js.Func
	cancelFunc = func() {
		js.Global().Get("self").Call("removeEventListener", "message", callback)
		callback.Release()
		close(selectPath)
	}
	callback = js.FuncOf(func(this js.Value, args []js.Value) any {
		data := args[0].Get("data")
		switch methodName := data.Index(0).String(); methodName {
		case "init_engine_with_path":
			ConsumeMessageEvent(args[0])
			rootPath := data.Index(1).String()
			go func() {
				selectPath <- rootPath
				SendBackMethodOK(methodName)
			}()
		}
		return nil
	})
	js.Global().Get("self").Call("addEventListener", "message", callback, false)
	return selectPath, cancelFunc
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
