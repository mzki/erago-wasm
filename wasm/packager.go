//go:build js && wasm
// +build js,wasm

package main

import (
	"path/filepath"
	"syscall/js"

	"github.com/mzki/erago/app"
	model "github.com/mzki/erago/mobile/model/v2"
)

func RunPackager(fsys *WebFileSystem, rootPath string) (cancelFunc func()) {
	pkgCallbacks := js.FuncOf(func(this js.Value, args []js.Value) any {
		data := args[0].Get("data")
		switch methodName := data.Index(0).String(); methodName {
		case "install_package":
			ConsumeMessageEvent(args[0])
			bs := ToGoBytes(data.Index(1))
			var baseName string
			if data.Index(2).IsUndefined() {
				baseName = "eragoPkg"
			} else {
				baseName = data.Index(2).String()
			}
			go func() { // to avoid blocking js eventLoop
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
			}()

		case "uninstall_package":
			ConsumeMessageEvent(args[0])
			fpath := data.Index(1).String()
			go func() { // to avoid blocking js eventLoop
				if err := fsys.Remove(fpath); err != nil {
					SendBackMethodError(methodName, err)
					return
				}
				SendBackMethodOK(methodName)
			}()

		case "validate_package":
			ConsumeMessageEvent(args[0])
			rootPath := data.Index(1).String()
			confPath := filepath.Join(rootPath, app.ConfigFile)
			go func() { // to avoid blocking js eventLoop
				if fsys.ExistDir(rootPath) && fsys.Exist(confPath) {
					SendBackMethodOK(methodName)
				} else {
					SendBackMethodNG(methodName)
				}
			}()

		case "exportsav":
			ConsumeMessageEvent(args[0])
			rootPath := data.Index(1).String()
			go func() { // to avoid blocking js eventLoop
				subFsys, err := fsys.Sub(rootPath, false)
				if err != nil {
					SendBackMethodError(methodName, err)
					return
				}
				savBs, err := model.ExportSav(rootPath, subFsys)
				if err != nil {
					if model.IsExportFileNotFound(err) {
						savBs = []byte{} // suceeded with empty bytes.
					} else {
						SendBackMethodError(methodName, err)
						return
					}
				}
				jsBs := ToJsBytes(savBs)
				SendBackSavZipBytes(methodName, jsBs)
			}()

		case "importsav":
			ConsumeMessageEvent(args[0])
			rootPath := data.Index(1).String()
			go func() { // to avoid blocking js eventLoop
				bs := ToGoBytes(data.Index(2))
				subFsys, err := fsys.Sub(rootPath, false)
				if err != nil {
					SendBackMethodError(methodName, err)
					return
				}
				if err := model.ImportSav(rootPath, subFsys, bs); err != nil {
					SendBackMethodError(methodName, err)
					return
				}
				SendBackMethodOK(methodName)
			}()

		case "exportlog":
			ConsumeMessageEvent(args[0])
			rootPath := data.Index(1).String()
			go func() { // to avoid blocking js eventLoop
				subFsys, err := fsys.Sub(rootPath, false)
				if err != nil {
					SendBackMethodError(methodName, err)
					return
				}
				logBs, err := model.ExportLog(rootPath, subFsys)
				if err != nil {
					if model.IsExportFileNotFound(err) {
						logBs = []byte{} // suceeded with empty bytes.
					} else {
						SendBackMethodError(methodName, err)
						return
					}
				}
				jsBs := ToJsBytes(logBs)
				SendBackLogBytes(methodName, jsBs)
			}()

		}
		return nil
	})

	js.Global().Get("self").Call("addEventListener", "message", pkgCallbacks, false)

	cancelFunc = func() {
		js.Global().Get("self").Call("removeEventListener", "message", pkgCallbacks)
		pkgCallbacks.Release()
	}
	return
}
