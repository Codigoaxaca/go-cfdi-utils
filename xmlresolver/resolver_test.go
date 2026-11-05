package xmlresolver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
)

func TestGetUsesCache(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		_, _ = w.Write([]byte("contenido " + r.URL.Path))
	}))
	defer srv.Close()

	r := NewWithLocalPath(t.TempDir())
	for i := 0; i < 3; i++ {
		data, err := r.Get(context.Background(), srv.URL+"/a/b.xsd")
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "contenido /a/b.xsd" {
			t.Fatalf("contenido inesperado %q", data)
		}
	}
	if hits != 1 {
		t.Fatalf("se esperaba una descarga, hubo %d", hits)
	}
	local, _ := r.LocalFileFor(srv.URL + "/a/b.xsd")
	if _, err := os.Stat(local); err != nil {
		t.Fatal(err)
	}
}

func TestGetWithoutCacheAndErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/404" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()
	r := &Resolver{}
	if r.HasLocalPath() {
		t.Fatal("no debe tener caché")
	}
	if _, err := r.Get(context.Background(), srv.URL+"/x"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Get(context.Background(), srv.URL+"/404"); err == nil {
		t.Fatal("se esperaba error")
	}
	if _, err := NewWithLocalPath(t.TempDir()).LocalFileFor("no-es-url"); err == nil {
		t.Fatal("se esperaba error de URL")
	}
}

func TestSatCertificateLive(t *testing.T) {
	if os.Getenv("CFDI_LIVE_TESTS") == "" {
		t.Skip("define CFDI_LIVE_TESTS=1 para descargar del SAT")
	}
	cert, err := NewWithLocalPath(t.TempDir()).SatCertificate(context.Background(), "00001000000505211329")
	if err != nil {
		t.Fatal(err)
	}
	if cert.NoCertificado() != "00001000000505211329" {
		t.Fatalf("certificado inesperado %s", cert.NoCertificado())
	}
}
