//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"syscall/js"

	model "github.com/mzki/erago/mobile/model/v2"
)

type EngineCallbackID int

const (
	EngineOnPublishJson EngineCallbackID = iota
	EngineOnPublishJsonTemporary
	EngineOnRemove
	EngineOnRemoveAll

	EngineOnCommandRequested
	EngineOnInputRequested
	EngineOnInputRequestClosed

	EngineNotifyQuit
)

type uiMessanger struct {
	done chan (struct{})
}

func newUiMessenger() *uiMessanger {
	return &uiMessanger{done: make(chan struct{})}
}

func (ui *uiMessanger) OnPublishJson(s string) error {
	sendEventToJs(EngineOnPublishJson, s)
	return nil
}
func (ui *uiMessanger) OnPublishJsonTemporary(s string) error {
	sendEventToJs(EngineOnPublishJsonTemporary, s)
	return nil
}
func (ui *uiMessanger) OnRemove(nParagraph int) error {
	sendEventToJs(EngineOnRemove, nParagraph)
	return nil
}
func (ui *uiMessanger) OnRemoveAll() error { sendEventToJs(EngineOnRemoveAll); return nil }

// it is called when mobile.app requires inputting
// user's command.
func (ui *uiMessanger) OnCommandRequested() { sendEventToJs(EngineOnCommandRequested) }

// it is called when mobile.app requires just input any command.
func (ui *uiMessanger) OnInputRequested() { sendEventToJs(EngineOnInputRequested) }

// it is called when mobile.app no longer requires any input,
// such as just-input and command.
func (ui *uiMessanger) OnInputRequestClosed() { sendEventToJs(EngineOnInputRequestClosed) }

func (ui *uiMessanger) NotifyQuit(err error) {
	close(ui.done)
	sendEventToJs(EngineNotifyQuit, err)
}

func (ui *uiMessanger) Done() <-chan (struct{}) {
	return ui.done
}

func sendEventToJs(cbID EngineCallbackID, args ...any) {
	// jsargs = Array.of(args)
	// options = { details: jsargs }
	// self.dispatchEvent(new CustomEvent(cbID.String(), options ))
	jsargs := jsArrayOf(args...)
	options := JsOptions(map[string]any{"detail": jsargs})
	ev := js.Global().Get("CustomEvent").New(cbID.String(), options)
	js.Global().Get("self").Call("dispatchEvent", ev)
}

func InitEngine(baseDir string, fsys model.FileSystemGlob) (messenger *uiMessanger, quitFunc func(), err error) {
	messenger = newUiMessenger()
	if err := model.Init(messenger, baseDir, &model.InitOptions{
		ImageFetchType: model.ImageFetchEncodedPNG,
		FileSystem:     fsys,
	}); err != nil {
		return nil, nil, fmt.Errorf("init Error: %w", err)
	}
	quitFunc = func() {
		model.Quit()
	}
	err = nil
	return
}

func RunEngine(appCtx model.AppContext) {
	model.Main(appCtx)
}

func RunIO() (cancelFunc func()) {
	ioCallbacks := js.FuncOf(func(this js.Value, args []js.Value) any {
		data := args[0].Get("data")
		go func() {
			switch methodName := data.Index(0).String(); methodName {
			case "send_command":
				model.SendCommand(data.Index(1).String())
				SendBackMethodOK(methodName)
			case "send_ctrl_skipping_wait":
				model.SendSkippingWait()
				SendBackMethodOK(methodName)
			case "send_ctrl_stop_skipping_wait":
				model.SendStopSkippingWait()
				SendBackMethodOK(methodName)
			case "send_quit":
				model.Quit()
				SendBackMethodOK(methodName)
			case "set_textunit_px":
				wPx := data.Index(1).Float()
				hPx := data.Index(2).Float()
				if err := model.SetTextUnitPx(wPx, hPx); err != nil {
					SendBackMethodError(methodName, err)
				}
				SendBackMethodOK(methodName)

			case "set_viewsize":
				lineCount := data.Index(1).Int()
				lineWidth := data.Index(2).Int()
				if err := model.SetViewSize(lineCount, lineWidth); err != nil {
					SendBackMethodError(methodName, err)
				}
				SendBackMethodOK(methodName)

			default:
				SendBackMethodNotImplemented(methodName)
			}
		}()
		return nil
	})

	js.Global().Get("self").Call("addEventListener", "message", ioCallbacks)

	cancelFunc = func() {
		js.Global().Get("self").Call("removeEventListener", "message", ioCallbacks)
		ioCallbacks.Release()
	}
	return
}
