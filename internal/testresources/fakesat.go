package testresources

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/codigoaxaca/go-cfdi-utils/satresources/embedded"
)

// FakeSAT simula el servidor de recursos del SAT, con ETag y respuestas 304.
// Se usa como Transport de un http.Client.
type FakeSAT struct {
	mu        sync.Mutex
	files     map[string][]byte
	redirects map[string]string

	// Fail hace que todas las solicitudes respondan 503.
	Fail atomic.Bool
	// Requests, Downloads y NotModified cuentan solicitudes, respuestas 200 y 304.
	Requests, Downloads, NotModified atomic.Int32
}

// NewFakeSAT crea un servidor con los archivos del respaldo incluido.
func NewFakeSAT() *FakeSAT {
	f := &FakeSAT{files: map[string][]byte{}, redirects: map[string]string{}}
	fsys := embedded.FS()
	_ = fs.WalkDir(fsys, "files", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		f.files[strings.TrimPrefix(p, "files/")] = data
		return nil
	})
	return f
}

// Client devuelve un cliente HTTP que usa este servidor.
func (f *FakeSAT) Client() *http.Client { return &http.Client{Transport: f} }

// Set reemplaza o agrega el archivo con la llave host/ruta.
func (f *FakeSAT) Set(key string, data []byte) {
	f.mu.Lock()
	f.files[key] = data
	f.mu.Unlock()
}

// Get devuelve el archivo con la llave host/ruta.
func (f *FakeSAT) Get(key string) []byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.files[key]
}

// Redirect hace que la llave host/ruta responda con una redirección a location.
func (f *FakeSAT) Redirect(key, location string) {
	f.mu.Lock()
	f.redirects[key] = location
	f.mu.Unlock()
}

// Delete elimina el archivo con la llave host/ruta.
func (f *FakeSAT) Delete(key string) {
	f.mu.Lock()
	delete(f.files, key)
	f.mu.Unlock()
}

// RoundTrip implementa http.RoundTripper.
func (f *FakeSAT) RoundTrip(r *http.Request) (*http.Response, error) {
	f.Requests.Add(1)
	resp := &http.Response{Request: r, Header: http.Header{}, Body: io.NopCloser(bytes.NewReader(nil))}
	if f.Fail.Load() {
		resp.StatusCode = http.StatusServiceUnavailable
		return resp, nil
	}
	key := strings.ToLower(r.URL.Host) + r.URL.Path
	f.mu.Lock()
	data, ok := f.files[key]
	location := f.redirects[key]
	f.mu.Unlock()
	if location != "" {
		resp.StatusCode = http.StatusFound
		resp.Header.Set("Location", location)
		return resp, nil
	}
	if !ok {
		resp.StatusCode = http.StatusNotFound
		return resp, nil
	}
	sum := sha256.Sum256(data)
	etag := `"` + hex.EncodeToString(sum[:8]) + `"`
	resp.Header.Set("ETag", etag)
	resp.Header.Set("Last-Modified", "Fri, 17 Apr 2026 00:30:45 GMT")
	if r.Header.Get("If-None-Match") == etag {
		f.NotModified.Add(1)
		resp.StatusCode = http.StatusNotModified
		return resp, nil
	}
	f.Downloads.Add(1)
	resp.StatusCode = http.StatusOK
	resp.Body = io.NopCloser(bytes.NewReader(data))
	return resp, nil
}
