//go:build !js && !wasm && demo
// +build !js,!wasm,demo

package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
)

var (
	certFile = flag.String("certfile", "", "Required; TLS cert file path.")
	keyFile  = flag.String("keyfile", "", "Required; TLS key file path.")
	htmlDir  = flag.String("html", "", "Required; html directory to be served.")
)

func main() {
	flag.Parse()
	if len(*certFile) == 0 || len(*keyFile) == 0 || len(*htmlDir) == 0 {
		fmt.Fprintln(os.Stderr, "required -certfile and -keyfile, but either or both are not specified")
		fmt.Fprintf(os.Stderr, "certfile(%s), keyfile(%s)\n", *certFile, *keyFile)
		flag.PrintDefaults()
		os.Exit(-1)
	}

	dirFs := os.DirFS(*htmlDir)
	// if err != nil {
	// 	fmt.Fprintln(os.Stderr, "got error at open directory(%v): %v\n", *htmlDir, err)
	// 	os.Exit(1)
	// }
	if err := http.ListenAndServeTLS(":443", *certFile, *keyFile, http.FileServerFS(dirFs)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}
}
