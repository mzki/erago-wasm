//go:build js && wasm
// +build js,wasm

package main

import (
	"path/filepath"
	"syscall/js"

	model "github.com/mzki/erago/mobile/model/v2"
)

func RunPackager(fsys *WebFileSystem, rootPath string) (cancelFunc func()) {
	pkgCallbacks := js.FuncOf(func(this js.Value, args []js.Value) any {
		data := args[0].Get("data")
		go func() {
			switch methodName := data.Index(0).String(); methodName {
			case "install_package":
				bs := ToGoBytes(data.Index(1))
				var baseName string
				if data.Index(2).IsUndefined() {
					baseName = "eragoPkg"
				} else {
					baseName = data.Index(2).String()
				}
				subFSys, err := fsys.Sub(baseName, true)
				if err != nil {
					SendBackMethodError(methodName, err)
					return
				}
				extractedDir, err := model.InstallPackage(subFSys, bs)
				if err != nil {
					SendBackMethodError(methodName, err)
					return
				}
				installedPath := filepath.Join(rootPath, baseName, extractedDir)
				SendBackInstalledPath(methodName, installedPath)

			case "uninstall_package":
				fpath := data.Index(1).String()
				if err := fsys.Remove(fpath); err != nil {
					SendBackMethodError(methodName, err)
					return
				}
				SendBackMethodOK(methodName)

			case "validate_package":
				rootPath := data.Index(1).String()
				if fsys.Exist(rootPath) {
					SendBackMethodOK(methodName)
				} else {
					SendBackMethodNG(methodName)
				}

			case "exportsav":
				rootPath := data.Index(1).String()
				savBs, err := model.ExportSav(rootPath, fsys)
				if err != nil {
					SendBackMethodError(methodName, err)
					return
				}
				jsBs := ToJsBytes(savBs)
				SendBackSavZipBytes(methodName, jsBs)

			case "importsav":
				rootPath := data.Index(1).String()
				bs := ToGoBytes(data.Index(2))
				if err := model.ImportSav(rootPath, fsys, bs); err != nil {
					SendBackMethodError(methodName, err)
					return
				}
				SendBackMethodOK(methodName)

			case "exportlog":
				rootPath := data.Index(1).String()
				logBs, err := model.ExportLog(rootPath, fsys)
				if err != nil {
					SendBackMethodError(methodName, err)
					return
				}
				jsBs := ToJsBytes(logBs)
				SendBackLogBytes(methodName, jsBs)

			default:
				SendBackMethodNotImplemented(methodName)
			}
		}()
		return nil
	})

	js.Global().Get("self").Call("addEventListener", "message", pkgCallbacks, false)

	cancelFunc = func() {
		js.Global().Get("self").Call("removeEventListener", "message", pkgCallbacks)
		pkgCallbacks.Release()
	}
	return
}
