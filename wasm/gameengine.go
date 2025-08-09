//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"syscall/js"

	model "github.com/mzki/erago/mobile/model/v2"
)

//go:generate stringer -type=EngineCallbackID
type EngineCallbackID int

const (
	EngineOnPublishBytes EngineCallbackID = iota
	EngineOnPublishBytesTemporary
	EngineOnRemove
	EngineOnRemoveAll

	EngineOnCommandRequested
	EngineOnInputRequested
	EngineOnInputRequestClosed

	EngineNotifyQuit
)

var _ model.UI = &uiMessenger{}

type uiMessenger struct {
	done chan (struct{})
}

func newUiMessenger() *uiMessenger {
	return &uiMessenger{done: make(chan struct{})}
}

func (ui *uiMessenger) OnPublishBytes(bs []byte) error {
	sendEventToJs(EngineOnPublishBytes, ToJsBytes(bs))
	return nil
}
func (ui *uiMessenger) OnPublishBytesTemporary(bs []byte) error {
	sendEventToJs(EngineOnPublishBytesTemporary, ToJsBytes(bs))
	return nil
}
func (ui *uiMessenger) OnRemove(nParagraph int) error {
	sendEventToJs(EngineOnRemove, nParagraph)
	return nil
}
func (ui *uiMessenger) OnRemoveAll() error { sendEventToJs(EngineOnRemoveAll); return nil }

// it is called when mobile.app requires inputting
// user's command.
func (ui *uiMessenger) OnCommandRequested() { sendEventToJs(EngineOnCommandRequested) }

// it is called when mobile.app requires just input any command.
func (ui *uiMessenger) OnInputRequested() { sendEventToJs(EngineOnInputRequested) }

// it is called when mobile.app no longer requires any input,
// such as just-input and command.
func (ui *uiMessenger) OnInputRequestClosed() { sendEventToJs(EngineOnInputRequestClosed) }

func (ui *uiMessenger) NotifyQuit(err error) {
	if err != nil {
		sendEventToJs(EngineNotifyQuit, err.Error())
	} else {
		sendEventToJs(EngineNotifyQuit, nil)
	}
	// close should be last since it blocks main()
	close(ui.done)
}

func (ui *uiMessenger) Done() <-chan (struct{}) {
	return ui.done
}

// NOTE: make sure args should be convertable via js.ValueOf.
func sendEventToJs(cbID EngineCallbackID, args ...any) {
	// jsargs = Array.of(args)
	// options = { details: jsargs }
	// self.dispatchEvent(new CustomEvent(cbID.String(), options ))
	jsargs := JsArrayOf(args...)
	options := JsOptions(map[string]any{"detail": jsargs})
	ev := js.Global().Get("CustomEvent").New(cbID.String(), options)
	js.Global().Get("self").Call("dispatchEvent", ev)
}

type EngineOptions struct {
	ImageFetchType      int
	MessageByteEncoding int
}

const (
	EngineOptionsKeyImageFetchTyoe      = "imageFetchType"
	EngineOptionsKeyMessageByteEncoding = "messageByteEncoding"
)

func ParseEngineOptions(opt js.Value) EngineOptions {
	defaultOpt := EngineOptions{
		ImageFetchType:      model.ImageFetchEncodedPNG,
		MessageByteEncoding: model.MessageByteEncodingJson,
	}
	if opt.Type() != js.TypeObject {
		return defaultOpt
	}

	if v := opt.Get(EngineOptionsKeyImageFetchTyoe); v.Type() == js.TypeNumber {
		fmt.Printf("Found options.%s = %v\n", EngineOptionsKeyImageFetchTyoe, v)
		defaultOpt.ImageFetchType = v.Int()
	}
	if v := opt.Get(EngineOptionsKeyMessageByteEncoding); v.Type() == js.TypeNumber {
		fmt.Printf("Found options.%s = %v\n", EngineOptionsKeyMessageByteEncoding, v)
		defaultOpt.MessageByteEncoding = v.Int()
	}
	return defaultOpt
}

func InitEngine(baseDir string, fsys model.FileSystemGlob, opt EngineOptions) (messenger *uiMessenger, quitFunc func(), err error) {
	messenger = newUiMessenger()
	if err := model.Init(messenger, baseDir, &model.InitOptions{
		ImageFetchType:      opt.ImageFetchType,
		MessageByteEncoding: opt.MessageByteEncoding,
		FileSystem:          fsys,
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
		switch methodName := data.Index(0).String(); methodName {
		case "send_command":
			ConsumeMessageEvent(args[0])
			go func() { // to avoid blocking js eventLoop
				model.SendCommand(data.Index(1).String())
				SendBackMethodOK(methodName)
			}()
		case "send_ctrl_skipping_wait":
			ConsumeMessageEvent(args[0])
			go func() { // to avoid blocking js eventLoop
				model.SendSkippingWait()
				SendBackMethodOK(methodName)
			}()
		case "send_ctrl_stop_skipping_wait":
			ConsumeMessageEvent(args[0])
			go func() { // to avoid blocking js eventLoop
				model.SendStopSkippingWait()
				SendBackMethodOK(methodName)
			}()
		case "send_quit":
			ConsumeMessageEvent(args[0])
			go func() { // to avoid blocking js eventLoop
				model.Quit()
				SendBackMethodOK(methodName)
			}()
		case "set_textunit_px":
			ConsumeMessageEvent(args[0])
			wPx := data.Index(1).Float()
			hPx := data.Index(2).Float()
			go func() { // to avoid blocking js eventLoop
				if err := model.SetTextUnitPx(wPx, hPx); err != nil {
					SendBackMethodError(methodName, err)
				}
				SendBackMethodOK(methodName)
			}()

		case "set_viewsize":
			ConsumeMessageEvent(args[0])
			lineCount := data.Index(1).Int()
			lineWidth := data.Index(2).Int()
			go func() { // to avoid blocking js eventLoop
				if err := model.SetViewSize(lineCount, lineWidth); err != nil {
					SendBackMethodError(methodName, err)
				}
				SendBackMethodOK(methodName)
			}()

		case "string_width":
			ConsumeMessageEvent(args[0])
			text := data.Index(1).String()
			go func() { // to avoid blocking js eventLoop
				width := model.StringWidth(text)
				SendBackStringWidth(methodName, width)
			}()

		}
		return nil
	})

	js.Global().Get("self").Call("addEventListener", "message", ioCallbacks)

	cancelFunc = func() {
		js.Global().Get("self").Call("removeEventListener", "message", ioCallbacks)
		ioCallbacks.Release()
	}
	return
}
