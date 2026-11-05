// Package xmlresolver obtiene los certificados de sellado del SAT (para
// verificar el sello del Timbre Fiscal Digital) y otros archivos que no
// cambian, y los guarda en una caché local para no descargarlos cada vez.
//
// Los recursos que sí pueden cambiar (XSLT de cadena original y esquemas XSD)
// los administra el paquete satresources, que revisa periódicamente si el SAT
// publicó versiones nuevas.
//
//	r := xmlresolver.New() // caché en el directorio de caché del usuario
//	cert, err := r.SatCertificate(ctx, "00001000000505211329")
package xmlresolver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/codigoaxaca/go-cfdi-utils/credentials"
)

// Resolver descarga recursos y los guarda en LocalPath.
type Resolver struct {
	// LocalPath es el directorio de la caché. Si está vacío no se guardan
	// los recursos en disco (se descargan cada vez que se necesitan).
	LocalPath string
	// Client es el cliente HTTP, si es nil se usa uno con tiempo límite de 30 segundos.
	Client *http.Client
}

// New crea un Resolver con caché en DefaultLocalPath().
func New() *Resolver { return &Resolver{LocalPath: DefaultLocalPath()} }

// NewWithLocalPath crea un Resolver con caché en el directorio dado.
func NewWithLocalPath(localPath string) *Resolver { return &Resolver{LocalPath: localPath} }

// DefaultLocalPath devuelve el directorio de caché predeterminado:
// <directorio de caché del usuario>/go-cfdi-utils/sat.
func DefaultLocalPath() string {
	dir, err := os.UserCacheDir()
	if err != nil || dir == "" {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "go-cfdi-utils", "sat")
}

var defaultClient = &http.Client{Timeout: 30 * time.Second}

func (r *Resolver) client() *http.Client {
	if r.Client != nil {
		return r.Client
	}
	return defaultClient
}

// HasLocalPath informa si el resolvedor guarda los recursos en disco.
func (r *Resolver) HasLocalPath() bool { return r.LocalPath != "" }

// LocalFileFor devuelve la ruta local donde se almacena un recurso remoto,
// siguiendo la estructura <LocalPath>/<host>/<ruta>. Devuelve "" si no hay caché.
func (r *Resolver) LocalFileFor(resource string) (string, error) {
	if r.LocalPath == "" {
		return "", nil
	}
	u, err := url.Parse(resource)
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("xmlresolver: URL inválida %q", resource)
	}
	clean := path.Clean("/" + u.Path)
	if strings.Contains(clean, "..") {
		return "", fmt.Errorf("xmlresolver: URL inválida %q", resource)
	}
	return filepath.Join(r.LocalPath, strings.ToLower(u.Host), filepath.FromSlash(clean)), nil
}

// Get devuelve el contenido de un recurso, desde la caché si existe o
// descargándolo (y guardándolo en la caché) si no.
func (r *Resolver) Get(ctx context.Context, resource string) ([]byte, error) {
	local, err := r.LocalFileFor(resource)
	if err != nil {
		return nil, err
	}
	if local != "" {
		if data, err := os.ReadFile(local); err == nil && len(data) > 0 {
			return data, nil
		}
	}
	data, err := r.Download(ctx, resource)
	if err != nil {
		return nil, err
	}
	if local != "" {
		if err := writeFileAtomic(local, data); err != nil {
			return nil, err
		}
	}
	return data, nil
}

// Download descarga un recurso sin usar la caché.
func (r *Resolver) Download(ctx context.Context, resource string) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, resource, nil)
	if err != nil {
		return nil, fmt.Errorf("xmlresolver: %w", err)
	}
	req.Header.Set("User-Agent", "go-cfdi-utils")
	resp, err := r.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("xmlresolver: no se pudo descargar %s: %w", resource, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("xmlresolver: no se pudo descargar %s: estado HTTP %d", resource, resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, fmt.Errorf("xmlresolver: no se pudo leer %s: %w", resource, err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("xmlresolver: el recurso %s está vacío", resource)
	}
	return data, nil
}

// SatCertificate obtiene el certificado del SAT con el número dado (por
// ejemplo el atributo NoCertificadoSAT del TimbreFiscalDigital) desde
// https://rdc.sat.gob.mx/ o desde la caché.
func (r *Resolver) SatCertificate(ctx context.Context, noCertificado string) (*credentials.Certificate, error) {
	resource, err := credentials.SatCertificateURL(noCertificado)
	if err != nil {
		return nil, err
	}
	data, err := r.Get(ctx, resource)
	if err != nil {
		return nil, err
	}
	cert, err := credentials.ParseCertificate(data)
	if err != nil {
		// si la caché tenía un archivo corrupto se descarta
		if local, _ := r.LocalFileFor(resource); local != "" {
			_ = os.Remove(local)
		}
		return nil, err
	}
	return cert, nil
}

func writeFileAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("xmlresolver: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return fmt.Errorf("xmlresolver: %w", err)
	}
	_, werr := tmp.Write(data)
	cerr := tmp.Close()
	if err := errors.Join(werr, cerr); err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("xmlresolver: %w", err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("xmlresolver: %w", err)
	}
	return nil
}
