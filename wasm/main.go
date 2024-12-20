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
	SendBackStatusLaunchOK()
	defer func() {
		fmt.Printf("---------- End %s-%s-%s ----------\n", APPNAME, VERSION, COMMIT_HASH)
		SendBackStatusEndsApp()
	}()

	const rootDir = "/erago-wasm"
	store := NewWebFilesystem(rootDir)

	var rootPath string
	{
		cancelRunPkg := RunPackager(store, rootDir)
		for {
			rootPath = <-AwaitPathSelect()
			if !strings.HasPrefix(rootPath, rootDir) {
				SendBackStatusPathInvalid(rootPath)
				continue
			}
			break
		}
		SendBackStatusPathSelected(rootPath)
		cancelRunPkg()
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
	cancelRunIO := RunIO()
	defer cancelRunIO()
	SendBackStatusEngineInitOK()

	<-AwaitRunEngine()
	RunEngine(messenger)
	SendBackStatusEngineStartOK()

	<-messenger.Done()
}

func AwaitPathSelect() <-chan string {
	selectPath := make(chan string)

	var callback js.Func
	cancelFunc := func() {
		js.Global().Get("self").Call("removeEventListener", "message", callback)
		callback.Release()
		close(selectPath)
	}
	callback = js.FuncOf(func(this js.Value, args []js.Value) any {
		data := args[0].Get("data")
		go func() {
			switch methodName := data.Index(0).String(); methodName {
			case "init_engine_with_path":
				rootPath := data.Index(1).String()
				SendBackMethodOK(methodName)
				// consume this event so that preventing call other APIs
				args[0].Call("stopImmediatePropagation")
				selectPath <- rootPath
				cancelFunc()
			}
		}()
		return nil
	})
	js.Global().Get("self").Call("addEventListener", "message", callback, false)
	return selectPath
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
		go func() {
			switch methodName := data.Index(0).String(); methodName {
			case "start_engine":
				SendBackMethodOK(methodName)
				// consume this event so that preventing call other APIs
				args[0].Call("stopImmediatePropagation")
				runEngine <- struct{}{}
				cancelFunc()
			}
		}()
		return nil
	})
	js.Global().Get("self").Call("addEventListener", "message", callback, false)
	return runEngine
}
