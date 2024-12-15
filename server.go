//go:build !js && !wasm
// +build !js,!wasm

package main

import (
	"embed"
	"flag"
	"fmt"
	"net/http"
	"os"
)

//go:embed html/*
var staticContent embed.FS

var (
	certFile = flag.String("certfile", "", "Required; TLS cert file path.")
	keyFile  = flag.String("keyfile", "", "Required; TLS key file path.")
)

func main() {
	flag.Parse()
	if len(*certFile) == 0 || len(*keyFile) == 0 {
		fmt.Fprintln(os.Stderr, "required -certfile and -keyfile, but either or both are not specified")
		fmt.Fprintf(os.Stderr, "certfile(%s), keyfile(%s)\n", *certFile, *keyFile)
		flag.PrintDefaults()
		os.Exit(-1)
	}

	err := http.ListenAndServeTLS(":443", *certFile, *keyFile, http.FileServer(http.FS(staticContent)))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}
}
