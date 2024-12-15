//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"strings"
	"syscall/js"
)

var (
	APPNAME string
	VERSION string
)

func main() {
	fmt.Printf("---------- Start %s %s ----------\n", APPNAME, VERSION)
	SendBackStatusLaunchOK()

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
		SendBackStatusEngineStartNG(err)
	}

	done, err := RunEngine(rootPath, rootPathStore)
	if err != nil {
		SendBackStatusEngineStartNG(err)
		return
	}
	SendBackStatusEngineStartOK()
	cancelRunIO := RunIO()
	defer cancelRunIO()

	<-done
	SendBackStatusEndsApp()
	fmt.Printf("---------- End %s %s ----------\n", APPNAME, VERSION)
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
			case "start_engine_with_path":
				rootPath := data.Index(1).String()
				SendBackMethodOK(methodName)
				selectPath <- rootPath
				cancelFunc()
			}
		}()
		return nil
	})
	js.Global().Get("self").Call("addEventListener", "message", callback, false)
	return selectPath
}
