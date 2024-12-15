//go:build js && wasm
// +build js,wasm

package main

import (
	"syscall/js"
)

// Await1 is same as Await except it expects number of returned values is exactly one.
func Await1(awaitable js.Value) (ret js.Value, jsErr js.Error) {
	then, catch := Await(awaitable)
	if catch != nil {
		return js.Null(), js.Error{Value: catch[0]}
	}
	return then[0], jsErrNull
}

// Await awaits js object awaitable and returns resolved then objects and errors on catch.
func Await(awaitable js.Value) (thenResult []js.Value, catchResult []js.Value) {
	then, catch, release := AwaitChan(awaitable)
	defer release()

	select {
	case result := <-then:
		return result, nil
	case err := <-catch:
		return nil, err
	}
}

// AwaitChan awaits js object awaitable and returns channels for resolved then objects and errors on catch.
// release function should be called after receiving some values from channel to release internal resource.
func AwaitChan(awaitable js.Value) (
	thenCh <-chan []js.Value,
	catchCh <-chan []js.Value,
	release func(),
) {
	// https://stackoverflow.com/a/68427221
	then := make(chan []js.Value)
	// defer close(then)
	thenFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		then <- args
		return nil
	})
	// defer thenFunc.Release()

	catch := make(chan []js.Value)
	// defer close(catch)
	catchFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		catch <- args
		return nil
	})
	// defer catchFunc.Release()

	awaitable.Call("then", thenFunc).Call("catch", catchFunc)

	return then, catch, func() {
		thenFunc.Release()
		catchFunc.Release()
		close(then)
		close(catch)
	}
}
