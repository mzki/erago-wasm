//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall/js"

	model "github.com/mzki/erago/mobile/model/v2"
)

type WebFileSystem struct {
	root        js.Value
	absRootPath string
}

func NewWebFilesystem(absBaseDir string) *WebFileSystem {
	if !filepath.IsAbs(absBaseDir) {
		panic("must be absolute dir, but passing: " + absBaseDir)
	}
	root, jsErr := Await1(js.Global().Get("navigator").Get("storage").Call("getDirectory"))
	if !jsErr.IsNull() {
		panic("must be get OPFS root, but got error: " + jsErr.Error())
	}
	subRoot, jsErr := recursiveGetDirHandle(root, absBaseDir[1:], JsOptions(map[string]any{"create": true}))
	if !jsErr.IsNull() {
		panic("must be passed exist path, but got error for " + absBaseDir + " " + jsErr.Error())
	}
	return &WebFileSystem{
		root:        subRoot,
		absRootPath: absBaseDir,
	}
}

func (fsys *WebFileSystem) Sub(subDir string, create bool) (*WebFileSystem, error) {
	subDir, err := fsys.relPath(subDir)
	if err != nil {
		return nil, err
	}
	subRoot, jsErr := recursiveGetDirHandle(fsys.root, subDir, JsOptions(map[string]any{"create": create}))
	if !jsErr.IsNull() {
		return nil, &fs.PathError{Op: "open-subdir", Path: subDir, Err: jsErr}
	}
	return &WebFileSystem{
		root:        subRoot,
		absRootPath: filepath.Join(fsys.absRootPath, subDir),
	}, nil
}

func (fsys *WebFileSystem) jsDirEntries() ([]js.Value, js.Error) {
	entries := make([]js.Value, 0, 4)
	nameIter := fsys.root.Call("values")
	for {
		ret, jsErr := Await1(nameIter.Call("next"))
		if !jsErr.IsNull() {
			return nil, jsErr
		}
		if ret.Get("done").Truthy() {
			break
		}
		entries = append(entries, ret.Get("value"))
	}
	return entries, jsErrNull
}

func (fsys *WebFileSystem) relPath(fpath string) (string, error) {
	if filepath.IsAbs(fpath) {
		return filepath.Rel(fsys.absRootPath, fpath)
	} else {
		return fpath, nil
	}
}

func (fsys *WebFileSystem) openSyncAccessHandle(fpath string, create bool) (js.Value, error) {
	fpath, err := fsys.relPath(fpath)
	if err != nil {
		return js.Null(), err
	}

	fileHandle, jsErr := recursiveGetFileHandle(fsys.root, fpath, JsOptions(map[string]any{"create": create}))
	if !jsErr.IsNull() {
		return js.Null(), &fs.PathError{Op: "open-handle", Path: fpath, Err: jsErr}
	}
	accessHandle, jsErr := Await1(fileHandle.Call("createSyncAccessHandle"))
	if !jsErr.IsNull() {
		return js.Null(), &fs.PathError{Op: "open-syncaccess", Path: fpath, Err: jsErr}
	}
	return accessHandle, nil
}

func (fsys *WebFileSystem) Load(fpath string) (model.ReadCloser, error) {
	syncReader, err := fsys.openSyncAccessHandle(fpath, false)
	if err != nil {
		return nil, &fs.PathError{Op: "open-read", Path: fpath, Err: err}
	}
	fileSize := syncReader.Call("getSize")
	return newWebReader(fileSize.Int(), fpath, syncReader), nil
}

func (fsys *WebFileSystem) Store(fpath string) (model.WriteCloser, error) {
	syncWriter, err := fsys.openSyncAccessHandle(fpath, true)
	if err != nil {
		return nil, &fs.PathError{Op: "open-write", Path: fpath, Err: err}
	}
	return newWebWriter(fpath, syncWriter), nil
}

func (fsys *WebFileSystem) Exist(fpath string) bool {
	fpath, err := fsys.relPath(fpath)
	if err != nil {
		return false
	}
	_, jsErr := recursiveGetFileHandle(fsys.root, fpath, JsOptions(map[string]any{"create": false}))
	return jsErr.IsNull()
}

func (fsys *WebFileSystem) Glob(pattern string) (*model.StringList, error) {
	pattern, err := fsys.relPath(pattern)
	if err != nil {
		return nil, &fs.PathError{Op: "glob", Path: pattern, Err: err}
	}

	matches, err := fsys.glob(0, "", pattern)
	if err != nil {
		return nil, &fs.PathError{Op: "glob", Path: pattern, Err: err}
	}

	// adjust model.StringList
	slist := model.NewStringList()
	for _, m := range matches {
		slist.Append(m)
	}
	return slist, nil
}

var ErrTooManyFilesInGlobPatten = fmt.Errorf("too many files found by glob pattern, considered infinite loop")

func (fsys *WebFileSystem) glob(nMatches int, parentDir string, pattern string) ([]string, error) {
	// to avoid infinite file travasal
	const maxMatches = 10000
	if nMatches > maxMatches {
		return nil, ErrTooManyFilesInGlobPatten
	}
	// empty pattern
	if pattern == "" {
		return []string{}, nil
	}

	dirPtn, restPtn := splitParent(pattern)
	// check whether pattern is bad.
	if _, err := filepath.Match(dirPtn, ""); err != nil {
		return nil, err
	}

	matches := make([]string, 0, 4)
	entries, jsErr := fsys.jsDirEntries()
	if !jsErr.IsNull() {
		return nil, &fs.PathError{Op: "readdir-entries", Path: fsys.absRootPath, Err: jsErr}
	}
	if len(restPtn) == 0 {
		// leaf directory. collect matched files under parent
		for _, entry := range entries {
			if kind := entry.Get("kind").String(); kind == "file" {
				fileName := entry.Get("name").String()
				if ok, _ := filepath.Match(dirPtn, fileName); ok {
					matches = append(matches, filepath.Join(parentDir, fileName))
				}
			}
		}
	} else {
		// intermidate directory. search recurrsively
		for _, entry := range entries {
			if kind := entry.Get("kind").String(); kind == "directory" {
				dirName := entry.Get("name").String()
				if ok, _ := filepath.Match(dirPtn, dirName); !ok {
					continue
				}
				subRoot, err := fsys.Sub(dirName, false)
				if err != nil {
					return nil, err
				}
				m, err := subRoot.glob(nMatches+len(matches), filepath.Join(parentDir, dirName), restPtn)
				if err != nil {
					return nil, err
				}
				matches = append(matches, m...)
			}
		}
	}
	return matches, nil
}

func (fsys *WebFileSystem) Remove(fpath string) error {
	fpath, err := fsys.relPath(fpath)
	if err != nil {
		return &fs.PathError{Op: "remove", Path: fpath, Err: err}
	}
	dir, file := filepath.Split(fpath)
	if len(dir) == 0 {
		_, jsErr := Await1(
			fsys.root.Call("removeEntry", file, JsOptions(map[string]any{"recursive": true})),
		)
		if !jsErr.IsNull() {
			return &fs.PathError{Op: "remove", Path: file, Err: jsErr}
		}
	} else {
		subRoot, err := fsys.Sub(dir, false)
		if err != nil {
			return &fs.PathError{Op: "remove", Path: dir, Err: err}
		}
		err = subRoot.Remove(file)
		if err != nil {
			// this error is result of API call. no need to add more information.
			return err
		}
	}
	return nil
}

func recursiveGetFileHandle(root js.Value, relPath string, options js.Value) (ret js.Value, err js.Error) {
	return recursiveGetXXXHandle("getFileHandle", root, relPath, options)
}

func recursiveGetDirHandle(root js.Value, relPath string, options js.Value) (ret js.Value, err js.Error) {
	return recursiveGetXXXHandle("getDirectoryHandle", root, relPath, options)
}

func recursiveGetXXXHandle(getHandleMethod string, root js.Value, relPath string, options js.Value) (ret js.Value, err js.Error) {
	xxxHandleAsync, jsErr := recursiveGetXXXHandleAsync(getHandleMethod, root, relPath, options)
	if !jsErr.IsNull() {
		return js.Null(), jsErr
	}
	ret, jsErr = Await1(xxxHandleAsync)
	if !jsErr.IsNull() {
		return js.Null(), jsErr
	}
	return ret, jsErrNull
}

func recursiveGetXXXHandleAsync(getHandleMethod string, root js.Value, relPath string, options js.Value) (ret js.Value, err js.Error) {
	subDir, rest := splitParent(relPath)
	if len(rest) == 0 {
		return root.Call(getHandleMethod, relPath, options), jsErrNull
	} else {
		subDirHandleAsync := root.Call("getDirectoryHandle", subDir, options)
		subDirHandle, jsErr := Await1(subDirHandleAsync)
		if !jsErr.IsNull() {
			return js.Null(), jsErr
		}
		return recursiveGetXXXHandleAsync(getHandleMethod, subDirHandle, rest, options)
	}
}

// ========== Reader/Writer =============

type WebReader struct {
	mu        *sync.Mutex
	closed    bool
	fileSize  int
	readCount int
	path      string
	reader    js.Value
}

func newWebReader(fileSize int, path string, reader js.Value) *WebReader {
	return &WebReader{
		mu:        new(sync.Mutex),
		closed:    false,
		fileSize:  fileSize,
		readCount: 0,
		path:      path,
		reader:    reader,
	}
}

func (r *WebReader) Read(bs []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return 0, &fs.PathError{Op: "read", Path: r.path, Err: io.ErrClosedPipe}
	}
	if nBuf := r.fileSize - r.readCount; nBuf <= 0 {
		return 0, io.EOF
	}
	arrayBuffer := js.Global().Get("ArrayBuffer").New(len(bs))
	uint8Array := js.Global().Get("Uint8Array").New(arrayBuffer)
	readCount := r.reader.Call("read", uint8Array, JsOptions(map[string]any{"at": r.readCount}))
	nBytes := readCount.Int()
	copyN := js.CopyBytesToGo(bs, uint8Array)
	if copyN > nBytes {
		copyN = nBytes
	}
	r.readCount += copyN
	return copyN, nil
}

func (r *WebReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return &fs.PathError{Op: "close", Path: r.path, Err: io.ErrClosedPipe}
	}
	r.closed = true
	r.reader.Call("flush")
	r.reader.Call("close")
	return nil
}

type WebWriter struct {
	mu         *sync.Mutex
	closed     bool
	writeCount int
	path       string
	writer     js.Value
}

func newWebWriter(path string, writer js.Value) *WebWriter {
	return &WebWriter{
		mu:         new(sync.Mutex),
		closed:     false,
		writeCount: 0,
		path:       path,
		writer:     writer,
	}
}

func (w *WebWriter) Write(bs []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return 0, &fs.PathError{Op: "write", Path: w.path, Err: io.ErrClosedPipe}
	}
	arrayBuffer := js.Global().Get("ArrayBuffer").New(len(bs))
	uint8Array := js.Global().Get("Uint8Array").New(arrayBuffer)
	js.CopyBytesToJS(uint8Array, bs)
	written := w.writer.Call("write", uint8Array, JsOptions(map[string]any{"at": w.writeCount}))
	nBytes := written.Int()
	w.writeCount += nBytes
	return nBytes, nil
}

func (w *WebWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return &fs.PathError{Op: "close", Path: w.path, Err: io.ErrClosedPipe}
	}
	w.closed = true
	w.writer.Call("flush")
	w.writer.Call("close")
	return nil
}

// ========== utils =============

func splitParent(path string) (parent, rest string) {
	sepIndex := strings.Index(path, string(os.PathSeparator))
	if sepIndex < 0 {
		parent = path
	} else {
		parent = path[:sepIndex]
		rest = path[sepIndex+1:]
	}
	return
}
