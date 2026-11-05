package satresources

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type entry struct {
	info *fileInfo
	data []byte
}

type fileSet struct {
	files map[string]*entry
	order []string
}

func newFileSet() *fileSet { return &fileSet{files: map[string]*entry{}} }

func (set *fileSet) add(k string, e *entry) {
	if set.files[k] == nil {
		set.order = append(set.order, k)
	}
	set.files[k] = e
}

type collectMode int

const (
	// modeMissing descarga solo lo que no está en la versión anterior.
	modeMissing collectMode = iota
	// modeRevalidate revisa todo con solicitudes condicionales.
	modeRevalidate
	// modeForce descarga todo sin solicitudes condicionales.
	modeForce
)

const maxResourceSize = 64 << 20

// collect recorre los recursos y sus dependencias. En modeMissing los
// recursos que ya están en prev no se descargan ni se recorren.
func (s *Store) collect(ctx context.Context, roots []string, prev *view, mode collectMode) (*fileSet, error) {
	set := newFileSet()
	queue := append([]string(nil), roots...)
	for len(queue) > 0 {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("satresources: %w", err)
		}
		resource := queue[0]
		queue = queue[1:]
		k, err := s.keyFor(resource)
		if err != nil {
			return nil, err
		}
		if set.files[k] != nil {
			continue
		}
		var old *fileInfo
		var oldPath string
		if prev != nil {
			if old = prev.m.Files[k]; old != nil {
				oldPath = prev.filePath(k)
				if !fileExists(oldPath) {
					old = nil
				}
			}
		}
		if mode == modeMissing && old != nil {
			continue
		}
		var cond *fileInfo
		if mode == modeRevalidate && old != nil && !old.Fallback {
			cond = old
		}
		body, notModified, etag, lastModified, err := s.fetch(ctx, resource, cond)
		if err != nil {
			return nil, err
		}
		if notModified {
			data, err := os.ReadFile(oldPath)
			if err == nil {
				c := *old
				if etag != "" {
					c.ETag = etag
				}
				if lastModified != "" {
					c.LastModified = lastModified
				}
				set.add(k, &entry{info: &c, data: data})
				queue = append(queue, c.Deps...)
				continue
			}
			// el archivo anterior no se puede leer: se descarga completo
			if body, _, etag, lastModified, err = s.fetch(ctx, resource, nil); err != nil {
				return nil, err
			}
		}
		info, data, err := analyze(resource, k, body)
		if err != nil {
			return nil, err
		}
		info.ETag, info.LastModified = etag, lastModified
		set.add(k, &entry{info: info, data: data})
		queue = append(queue, info.Deps...)
	}
	return set, nil
}

// fetchURL usa https para los servidores del SAT aunque la URL sea http.
func fetchURL(resource string) string {
	u, err := url.Parse(resource)
	if err != nil {
		return resource
	}
	if u.Scheme == "http" && DefaultAllowHost(u.Hostname()) {
		u.Scheme = "https"
	}
	return u.String()
}

func (s *Store) fetch(ctx context.Context, resource string, cond *fileInfo) (body []byte, notModified bool, etag, lastModified string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fetchURL(resource), nil)
	if err != nil {
		return nil, false, "", "", fmt.Errorf("satresources: %w", err)
	}
	req.Header.Set("User-Agent", "go-cfdi-utils")
	if cond != nil {
		if cond.ETag != "" {
			req.Header.Set("If-None-Match", cond.ETag)
		}
		if cond.LastModified != "" {
			req.Header.Set("If-Modified-Since", cond.LastModified)
		}
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, false, "", "", fmt.Errorf("satresources: no se pudo descargar %s: %w", resource, err)
	}
	defer resp.Body.Close()
	etag, lastModified = resp.Header.Get("ETag"), resp.Header.Get("Last-Modified")
	switch {
	case resp.StatusCode == http.StatusNotModified && cond != nil:
		return nil, true, etag, lastModified, nil
	case resp.StatusCode != http.StatusOK:
		return nil, false, "", "", fmt.Errorf("satresources: no se pudo descargar %s: estado HTTP %d", resource, resp.StatusCode)
	}
	body, err = io.ReadAll(io.LimitReader(resp.Body, maxResourceSize+1))
	if err != nil {
		return nil, false, "", "", fmt.Errorf("satresources: no se pudo leer %s: %w", resource, err)
	}
	if len(body) > maxResourceSize {
		return nil, false, "", "", fmt.Errorf("satresources: %s excede %d bytes", resource, maxResourceSize)
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return nil, false, "", "", fmt.Errorf("satresources: el recurso %s está vacío", resource)
	}
	return body, false, etag, lastModified, nil
}
