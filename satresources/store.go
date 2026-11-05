package satresources

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Valores predeterminados de Config.
const (
	DefaultMaxAge        = 30 * 24 * time.Hour
	DefaultRetryInterval = time.Hour
)

var (
	// ErrOffline indica que el almacén está configurado sin acceso a internet.
	ErrOffline = errors.New("satresources: el almacén está en modo sin conexión")
	// ErrNotAllowed indica una URL de un servidor no permitido (por omisión
	// solo se descargan recursos de sat.gob.mx).
	ErrNotAllowed = errors.New("satresources: servidor no permitido")
	// ErrVerification indica que una versión descargada no pasó la verificación
	// y no se activó.
	ErrVerification = errors.New("satresources: la versión descargada no pasó la verificación")
	// ErrNotAvailable indica que un recurso no está guardado y no se pudo obtener.
	ErrNotAvailable = errors.New("satresources: recurso no disponible")
)

// Config configura un Store. Su valor cero es una configuración válida.
type Config struct {
	// Dir es el directorio donde se guardan los recursos. Por omisión DefaultDir().
	Dir string
	// MaxAge es cada cuánto se revisa si el SAT publicó cambios. La revisión
	// se hace en segundo plano al usar los recursos. Cero usa DefaultMaxAge
	// (30 días); un valor negativo desactiva la revisión automática (solo se
	// actualiza con Refresh o con el comando cfdi-sat-resources).
	MaxAge time.Duration
	// RetryInterval es el tiempo de espera antes de reintentar una revisión
	// automática que falló (por ejemplo si el SAT no respondió). Cero usa
	// DefaultRetryInterval (1 hora).
	RetryInterval time.Duration
	// Offline impide cualquier descarga: solo se usan los recursos de Dir o
	// el respaldo incluido (paquete satresources/embedded).
	Offline bool
	// HTTPClient es el cliente para descargar. Por omisión uno con tiempo
	// límite de 60 segundos por solicitud.
	HTTPClient *http.Client
	// Roots agrega recursos a mantener además de DefaultRoots(), por ejemplo
	// el XSD de un complemento que se usa siempre.
	Roots []string
	// Logger recibe los avisos de actualizaciones y errores. Por omisión
	// slog.Default(); usar slog.New(slog.DiscardHandler) para no registrar.
	Logger *slog.Logger
	// AllowHost decide de qué servidores se puede descargar. Por omisión solo
	// de sat.gob.mx y sus subdominios.
	AllowHost func(host string) bool
	// Fallback es el respaldo que se usa cuando no hay recursos en disco y no
	// se pueden descargar. Por omisión el registrado con RegisterFallback (al
	// importar satresources/embedded).
	Fallback fs.FS
}

// DefaultDir devuelve el directorio predeterminado:
// <directorio de caché del usuario>/go-cfdi-utils/satresources.
func DefaultDir() string {
	dir, err := os.UserCacheDir()
	if err != nil || dir == "" {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "go-cfdi-utils", "satresources")
}

// DefaultAllowHost permite sat.gob.mx y sus subdominios.
func DefaultAllowHost(host string) bool {
	host = strings.ToLower(host)
	return host == "sat.gob.mx" || strings.HasSuffix(host, ".sat.gob.mx")
}

var (
	fallbackMu sync.RWMutex
	fallbackFS fs.FS
)

// RegisterFallback registra el respaldo predeterminado. Lo usa el paquete
// satresources/embedded; fsys debe tener manifest.json y files/<host>/<ruta>,
// como lo genera Store.Export.
func RegisterFallback(fsys fs.FS) {
	fallbackMu.Lock()
	fallbackFS = fsys
	fallbackMu.Unlock()
}

var defaultStore atomic.Pointer[Store]

// Default devuelve el almacén que usa la librería cuando no se indica otro.
// Se crea con Config{} la primera vez que se pide.
func Default() *Store {
	if s := defaultStore.Load(); s != nil {
		return s
	}
	s := New(Config{})
	if defaultStore.CompareAndSwap(nil, s) {
		return s
	}
	return defaultStore.Load()
}

// SetDefault reemplaza el almacén predeterminado. Conviene llamarlo al
// arrancar la aplicación, antes de crear o validar CFDI.
func SetDefault(s *Store) {
	if s == nil {
		panic("satresources: SetDefault(nil)")
	}
	defaultStore.Store(s)
}

// Store guarda versiones completas de los recursos del SAT en disco. Es
// seguro para uso concurrente y varios procesos pueden compartir el mismo
// directorio.
type Store struct {
	cfg    Config
	dir    string
	maxAge time.Duration
	retry  time.Duration
	client *http.Client
	log    *slog.Logger
	allow  func(string) bool

	mu         sync.RWMutex
	view       *view
	viewLoaded time.Time

	writeMu sync.Mutex

	stMu        sync.Mutex
	lastErr     error
	lastErrAt   time.Time
	lastAttempt time.Time

	refreshing atomic.Bool
	bg         sync.WaitGroup

	// para pruebas
	now        func() time.Time
	pointerTTL time.Duration
}

// New crea un almacén. No accede al disco ni a la red hasta que se usa.
func New(cfg Config) *Store {
	s := &Store{cfg: cfg, now: time.Now, pointerTTL: 2 * time.Second}
	s.dir = cfg.Dir
	if s.dir == "" {
		s.dir = DefaultDir()
	}
	if abs, err := filepath.Abs(s.dir); err == nil {
		s.dir = abs
	}
	s.maxAge = cfg.MaxAge
	if s.maxAge == 0 {
		s.maxAge = DefaultMaxAge
	}
	s.retry = cfg.RetryInterval
	if s.retry <= 0 {
		s.retry = DefaultRetryInterval
	}
	s.allow = cfg.AllowHost
	if s.allow == nil {
		s.allow = DefaultAllowHost
	}
	client := http.Client{Timeout: 60 * time.Second}
	if cfg.HTTPClient != nil {
		client = *cfg.HTTPClient
	}
	// las redirecciones también deben ir a servidores permitidos
	checkRedirect := client.CheckRedirect
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if !s.allow(req.URL.Hostname()) {
			return fmt.Errorf("%w: redirección a %s", ErrNotAllowed, req.URL.Redacted())
		}
		if checkRedirect != nil {
			return checkRedirect(req, via)
		}
		if len(via) >= 10 {
			return errors.New("demasiadas redirecciones")
		}
		return nil
	}
	s.client = &client
	s.log = cfg.Logger
	if s.log == nil {
		s.log = slog.Default()
	}
	return s
}

// Dir devuelve el directorio del almacén.
func (s *Store) Dir() string { return s.dir }

// Allowed informa si el recurso es una URL válida de un servidor permitido.
func (s *Store) Allowed(resource string) bool {
	_, err := s.keyFor(resource)
	return err == nil
}

func (s *Store) keyFor(resource string) (string, error) {
	k, err := Key(resource)
	if err != nil {
		return "", err
	}
	host, _, _ := strings.Cut(k, "/")
	if !s.allow(host) {
		return "", fmt.Errorf("%w: %s", ErrNotAllowed, resource)
	}
	return k, nil
}

func (s *Store) fallback() fs.FS {
	if s.cfg.Fallback != nil {
		return s.cfg.Fallback
	}
	fallbackMu.RLock()
	defer fallbackMu.RUnlock()
	return fallbackFS
}

// ---------------------------------------------------------------------------
// Lectura

// Path devuelve la ruta en disco de un recurso (por ejemplo un XSD para
// xmllint). Si no está guardado se descarga junto con todo lo que incluye o
// importa. Los XSD guardados referencian a sus dependencias con rutas
// relativas, por lo que se pueden usar sin red.
func (s *Store) Path(ctx context.Context, resource string) (string, error) {
	k, err := s.keyFor(resource)
	if err != nil {
		return "", err
	}
	v, err := s.current()
	if err != nil {
		return "", err
	}
	if v != nil && v.m.Files[k] != nil {
		p := v.filePath(k)
		if _, err := os.Stat(p); err == nil {
			s.maybeAutoRefresh(v)
			return p, nil
		}
	}
	v, err = s.ensure(ctx, []string{resource})
	if err != nil {
		return "", err
	}
	if v == nil || v.m.Files[k] == nil {
		return "", fmt.Errorf("%w: %s", ErrNotAvailable, resource)
	}
	s.maybeAutoRefresh(v)
	return v.filePath(k), nil
}

// ReadFile devuelve el contenido guardado de un recurso; si no está guardado
// se descarga como en Path. En los XSD las referencias a otros esquemas son
// rutas relativas.
func (s *Store) ReadFile(ctx context.Context, resource string) ([]byte, error) {
	p, err := s.Path(ctx, resource)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err == nil {
		return data, nil
	}
	// la versión pudo eliminarse por otro proceso (Clear); se intenta una vez más
	s.invalidate()
	if p, err = s.Path(ctx, resource); err != nil {
		return nil, err
	}
	return os.ReadFile(p)
}

// Version devuelve el identificador de la versión activa ("" si no hay
// recursos guardados). Cambia cada vez que se activa una actualización.
func (s *Store) Version() string {
	v, _ := s.current()
	if v == nil {
		return ""
	}
	return v.id
}

// ---------------------------------------------------------------------------
// Administración

// Prepare se asegura de que estén guardados todos los recursos de
// DefaultRoots() y Config.Roots, descargando solo los que falten (no revisa
// cambios de los existentes). Conviene llamarlo al arrancar la aplicación
// para no descargar durante la primera factura. Si no hay red y existe el
// respaldo incluido, se usa el respaldo.
func (s *Store) Prepare(ctx context.Context) error {
	v, err := s.current()
	if err != nil {
		return err
	}
	var missing []string
	for _, r := range s.roots(v) {
		k, err := s.keyFor(r)
		if err != nil {
			return err
		}
		if v == nil || v.m.Files[k] == nil {
			missing = append(missing, r)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	_, err = s.ensure(ctx, missing)
	return err
}

// RefreshOption modifica el comportamiento de Refresh.
type RefreshOption int

// Force descarga todos los recursos sin usar la caché (sin solicitudes
// condicionales) y activa una versión nueva aunque no haya cambios. La
// versión anterior se conserva hasta que la nueva pasa la verificación.
const Force RefreshOption = 1

// RefreshResult describe el resultado de una actualización.
type RefreshResult struct {
	// Version es la versión activa al terminar.
	Version string
	// Changed indica que se activó una versión nueva.
	Changed bool
	// Checked es el número de archivos revisados.
	Checked int
	// Updated, Added y Removed listan las URL modificadas, nuevas y eliminadas.
	Updated, Added, Removed []string
}

// Refresh revisa con el SAT todos los recursos guardados (y descarga los de
// DefaultRoots y Config.Roots que falten). Si hubo cambios, descarga la
// versión nueva completa, la verifica y la activa. Si algo falla devuelve el
// error y se sigue usando la versión actual.
func (s *Store) Refresh(ctx context.Context, opts ...RefreshOption) (*RefreshResult, error) {
	if s.cfg.Offline {
		return nil, ErrOffline
	}
	force := false
	for _, o := range opts {
		if o == Force {
			force = true
		}
	}
	return s.refresh(ctx, force, false)
}

// Clear borra todos los recursos guardados en Dir. Si hay una actualización
// en curso espera a que termine. Los siguientes usos vuelven a descargar.
func (s *Store) Clear() error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	unlock, err := s.lock(ctx)
	if err != nil {
		return err
	}
	defer unlock()
	if err := os.Remove(s.pointerPath()); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("satresources: %w", err)
	}
	if err := os.RemoveAll(s.snapshotsDir()); err != nil {
		return fmt.Errorf("satresources: %w", err)
	}
	s.mu.Lock()
	s.view = nil
	s.viewLoaded = time.Time{}
	s.mu.Unlock()
	s.stMu.Lock()
	s.lastErr, s.lastErrAt, s.lastAttempt = nil, time.Time{}, time.Time{}
	s.stMu.Unlock()
	return nil
}

// Export copia la versión activa (manifest.json y files/) en dir. Sirve para
// generar el respaldo incluido o para copiar los recursos a otro equipo.
func (s *Store) Export(dir string) error {
	v, err := s.current()
	if err != nil {
		return err
	}
	if v == nil {
		return fmt.Errorf("%w: no hay recursos guardados", ErrNotAvailable)
	}
	for k := range v.m.Files {
		data, err := os.ReadFile(v.filePath(k))
		if err != nil {
			return fmt.Errorf("satresources: %w", err)
		}
		if err := writeFileAtomic(filepath.Join(dir, "files", filepath.FromSlash(k)), data); err != nil {
			return err
		}
	}
	return writeManifest(filepath.Join(dir, "manifest.json"), v.m)
}

// Status describe el estado del almacén.
type Status struct {
	Dir string
	// Version es la versión activa, vacía si no hay recursos guardados.
	Version string
	// CreatedAt es cuándo se descargó la versión activa.
	CreatedAt time.Time
	// CheckedAt es la última vez que se revisaron cambios con el SAT. Es cero
	// si la versión viene del respaldo incluido y aún no se ha revisado.
	CheckedAt time.Time
	// NextCheck es a partir de cuándo se revisarán cambios de forma
	// automática; cero si la revisión automática está desactivada.
	NextCheck time.Time
	// Files es el número de archivos de la versión activa.
	Files int
	// Roots son los recursos que se mantienen actualizados.
	Roots []string
	// Fallback indica que algunos archivos vienen del respaldo incluido.
	Fallback bool
	// Offline indica el modo sin conexión.
	Offline bool
	// Refreshing indica que hay una revisión en curso en este proceso.
	Refreshing bool
	// LastError es el último error de actualización (nil si la última terminó bien).
	LastError   error
	LastErrorAt time.Time
}

// String devuelve el estado en texto legible.
func (st Status) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "directorio: %s\n", st.Dir)
	if st.Version == "" {
		b.WriteString("versión: (sin recursos guardados)\n")
	} else {
		fmt.Fprintf(&b, "versión: %s\n", st.Version)
		fmt.Fprintf(&b, "archivos: %d\n", st.Files)
		fmt.Fprintf(&b, "descargada: %s\n", formatTime(st.CreatedAt))
		fmt.Fprintf(&b, "revisada: %s\n", formatTime(st.CheckedAt))
		if st.Fallback {
			b.WriteString("origen: respaldo incluido (se reemplazará al revisar con el SAT)\n")
		} else {
			b.WriteString("origen: SAT\n")
		}
	}
	switch {
	case st.Offline:
		b.WriteString("revisión automática: desactivada (sin conexión)\n")
	case st.NextCheck.IsZero():
		b.WriteString("revisión automática: desactivada\n")
	default:
		fmt.Fprintf(&b, "próxima revisión: %s\n", formatTime(st.NextCheck))
	}
	if st.Refreshing {
		b.WriteString("revisión en curso\n")
	}
	if st.LastError != nil {
		fmt.Fprintf(&b, "último error (%s): %v\n", formatTime(st.LastErrorAt), st.LastError)
	}
	return b.String()
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "nunca"
	}
	return t.Local().Format("2006-01-02 15:04:05 MST")
}

// Status devuelve el estado actual.
func (s *Store) Status() Status {
	v, err := s.current()
	st := Status{Dir: s.dir, Offline: s.cfg.Offline, Refreshing: s.refreshing.Load()}
	if v != nil {
		st.Version = v.id
		st.CreatedAt = v.m.Created
		st.CheckedAt = v.m.Checked
		st.Files = len(v.m.Files)
		st.Roots = s.roots(v)
		for _, f := range v.m.Files {
			if f.Fallback {
				st.Fallback = true
				break
			}
		}
		if s.maxAge > 0 && !s.cfg.Offline {
			st.NextCheck = v.m.Checked.Add(s.maxAge)
		}
	} else if s.maxAge > 0 && !s.cfg.Offline {
		st.NextCheck = s.now()
	}
	s.stMu.Lock()
	st.LastError, st.LastErrorAt = s.lastErr, s.lastErrAt
	s.stMu.Unlock()
	if st.LastError == nil && err != nil {
		st.LastError = err
	}
	return st
}

// ---------------------------------------------------------------------------
// Versiones en disco
//
//	<Dir>/current                         identificador de la versión activa
//	<Dir>/snapshots/<id>/manifest.json    metadatos de cada archivo
//	<Dir>/snapshots/<id>/files/<host>/<ruta>
//	<Dir>/.lock                           candado entre procesos

type manifest struct {
	Format  int                  `json:"format"`
	ID      string               `json:"id"`
	Created time.Time            `json:"created"`
	Checked time.Time            `json:"checked"`
	Roots   []string             `json:"roots"`
	Files   map[string]*fileInfo `json:"files"`
}

type fileInfo struct {
	URL          string   `json:"url"`
	Kind         string   `json:"kind"`
	ETag         string   `json:"etag,omitempty"`
	LastModified string   `json:"lastModified,omitempty"`
	SHA256       string   `json:"sha256"`
	Size         int64    `json:"size"`
	Deps         []string `json:"deps,omitempty"`
	Fallback     bool     `json:"fallback,omitempty"`
}

type view struct {
	id    string
	dir   string
	m     *manifest
	mtime time.Time
}

func (v *view) filePath(key string) string {
	return filepath.Join(v.dir, "files", filepath.FromSlash(key))
}

func (s *Store) snapshotsDir() string { return filepath.Join(s.dir, "snapshots") }
func (s *Store) pointerPath() string  { return filepath.Join(s.dir, "current") }

func (s *Store) current() (*view, error) {
	s.mu.RLock()
	v, at := s.view, s.viewLoaded
	s.mu.RUnlock()
	if !at.IsZero() && s.now().Sub(at) < s.pointerTTL {
		return v, nil
	}
	return s.reload()
}

func (s *Store) invalidate() {
	s.mu.Lock()
	s.viewLoaded = time.Time{}
	s.mu.Unlock()
}

// reload vuelve a leer el apuntador y el manifiesto de la versión activa.
func (s *Store) reload() (*view, error) {
	s.mu.RLock()
	old := s.view
	s.mu.RUnlock()
	var nv *view
	raw, err := os.ReadFile(s.pointerPath())
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return nil, fmt.Errorf("satresources: %w", err)
	default:
		id := strings.TrimSpace(string(raw))
		dir := filepath.Join(s.snapshotsDir(), id)
		mpath := filepath.Join(dir, "manifest.json")
		fi, statErr := os.Stat(mpath)
		if old != nil && old.id == id && statErr == nil && fi.ModTime().Equal(old.mtime) {
			nv = old
		} else if statErr == nil {
			m, err := readManifest(mpath)
			if err != nil {
				s.log.Warn("satresources: no se pudo leer la versión activa, se descargará de nuevo", "dir", s.dir, "error", err)
			} else {
				nv = &view{id: id, dir: dir, m: m, mtime: fi.ModTime()}
			}
		}
	}
	s.mu.Lock()
	s.view = nv
	s.viewLoaded = s.now()
	s.mu.Unlock()
	return nv, nil
}

func readManifest(p string) (*manifest, error) {
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	return parseManifest(data)
}

func parseManifest(data []byte) (*manifest, error) {
	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("satresources: manifest.json: %w", err)
	}
	if m.Files == nil {
		m.Files = map[string]*fileInfo{}
	}
	return &m, nil
}

func writeManifest(p string, m *manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(p, append(data, '\n'))
}

func (s *Store) roots(v *view) []string {
	var all []string
	all = append(all, DefaultRoots()...)
	all = append(all, s.cfg.Roots...)
	if v != nil {
		all = append(all, v.m.Roots...)
	}
	seen := map[string]bool{}
	out := all[:0:0]
	for _, r := range all {
		k, err := Key(r)
		if err != nil || seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, r)
	}
	return out
}

func (s *Store) stale(v *view) bool {
	return s.now().Sub(v.m.Checked) >= s.maxAge
}

func (s *Store) maybeAutoRefresh(v *view) {
	if s.maxAge < 0 || s.cfg.Offline || v == nil || !s.stale(v) {
		return
	}
	s.stMu.Lock()
	if !s.lastAttempt.IsZero() && s.now().Sub(s.lastAttempt) < s.retry {
		s.stMu.Unlock()
		return
	}
	s.lastAttempt = s.now()
	s.stMu.Unlock()
	if !s.refreshing.CompareAndSwap(false, true) {
		return
	}
	s.bg.Add(1)
	go func() {
		defer s.bg.Done()
		defer s.refreshing.Store(false)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		_, _ = s.refresh(ctx, false, true)
	}()
}

func (s *Store) recordError(err error) {
	s.stMu.Lock()
	s.lastErr, s.lastErrAt = err, s.now()
	s.stMu.Unlock()
}

func (s *Store) clearError() {
	s.stMu.Lock()
	s.lastErr, s.lastErrAt = nil, time.Time{}
	s.stMu.Unlock()
}

// ensure guarda los recursos indicados (y sus dependencias) que falten en la
// versión activa. Si no se pueden descargar intenta con el respaldo.
func (s *Store) ensure(ctx context.Context, resources []string) (*view, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	unlock, err := s.lock(ctx)
	if err != nil {
		return nil, err
	}
	defer unlock()
	v, err := s.reload()
	if err != nil {
		return nil, err
	}
	var missing []string
	for _, r := range resources {
		k, err := s.keyFor(r)
		if err != nil {
			return nil, err
		}
		if v == nil || v.m.Files[k] == nil || !fileExists(v.filePath(k)) {
			missing = append(missing, r)
		}
	}
	if len(missing) == 0 {
		return v, nil
	}

	downloadErr := ErrOffline
	if !s.cfg.Offline {
		set, err := s.collect(ctx, missing, v, modeMissing)
		if err == nil {
			nv, err := s.commit(v, set, missing, s.now())
			if err == nil {
				return nv, nil
			}
			downloadErr = err
		} else {
			downloadErr = err
		}
		s.recordError(downloadErr)
	}

	if fb := s.fallback(); fb != nil {
		set, roots, err := s.fallbackSet(fb, missing, v)
		if err == nil {
			// la versión del respaldo se revisará con el SAT en cuanto sea posible
			nv, err := s.commit(v, set, roots, time.Time{})
			if err == nil {
				if !s.cfg.Offline {
					s.log.Warn("satresources: no se pudieron descargar los recursos del SAT, se usa el respaldo incluido", "error", downloadErr)
				}
				return nv, nil
			}
			return nil, err
		}
		if !errors.Is(err, ErrNotAvailable) {
			return nil, err
		}
	}
	return nil, downloadErr
}

// commit guarda un conjunto de archivos. Si no hay versión activa crea una
// nueva; si la hay, agrega los archivos nuevos a la versión activa.
func (s *Store) commit(v *view, set *fileSet, roots []string, checked time.Time) (*view, error) {
	if v == nil {
		m := &manifest{Format: 1, Created: s.now().UTC(), Checked: checked.UTC(), Files: map[string]*fileInfo{}}
		m.Roots = appendRoots(nil, roots)
		return s.createSnapshot(m, set, nil)
	}
	// Los archivos nuevos se escriben en la versión activa, pero nadie los usa
	// hasta que se agregan al manifiesto, después de verificarlos completos.
	m := cloneManifest(v.m)
	var written []string
	discard := func() {
		for _, k := range written {
			_ = os.Remove(v.filePath(k))
		}
	}
	for _, k := range set.order {
		e := set.files[k]
		if v.m.Files[k] != nil && fileExists(v.filePath(k)) {
			continue
		}
		if err := writeFileAtomic(v.filePath(k), e.data); err != nil {
			discard()
			return nil, err
		}
		written = append(written, k)
		m.Files[k] = e.info
	}
	if err := s.verify(set, v, v.dir); err != nil {
		discard()
		return nil, err
	}
	m.Roots = appendRoots(m.Roots, roots)
	if checked.IsZero() {
		m.Checked = time.Time{}
	}
	if err := writeManifest(filepath.Join(v.dir, "manifest.json"), m); err != nil {
		return nil, err
	}
	return s.reload()
}

// createSnapshot escribe una versión nueva en un directorio temporal, la
// verifica y la activa.
func (s *Store) createSnapshot(m *manifest, set *fileSet, prev *view) (*view, error) {
	id := newID(s.now())
	m.ID = id
	tmp := filepath.Join(s.snapshotsDir(), ".tmp-"+id)
	defer os.RemoveAll(tmp)
	for _, k := range set.order {
		e := set.files[k]
		if err := writeFileAtomic(filepath.Join(tmp, "files", filepath.FromSlash(k)), e.data); err != nil {
			return nil, err
		}
		m.Files[k] = e.info
	}
	if err := writeManifest(filepath.Join(tmp, "manifest.json"), m); err != nil {
		return nil, err
	}
	if err := s.verify(set, nil, tmp); err != nil {
		return nil, err
	}
	final := filepath.Join(s.snapshotsDir(), id)
	if err := os.Rename(tmp, final); err != nil {
		return nil, fmt.Errorf("satresources: %w", err)
	}
	if err := writeFileAtomic(s.pointerPath(), []byte(id+"\n")); err != nil {
		return nil, err
	}
	prevID := ""
	if prev != nil {
		// el plazo para borrar la versión anterior cuenta desde que deja de estar activa
		prevID = prev.id
		now := time.Now()
		_ = os.Chtimes(prev.dir, now, now)
	}
	s.prune(id, prevID)
	return s.reload()
}

// prune elimina las versiones que no son la activa ni la anterior y que
// dejaron de estar activas hace más de una hora (otro proceso podría estar
// leyéndolas), además de directorios temporales abandonados.
func (s *Store) prune(keep ...string) {
	entries, err := os.ReadDir(s.snapshotsDir())
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		skip := false
		for _, k := range keep {
			if k != "" && name == k {
				skip = true
			}
		}
		if skip {
			continue
		}
		info, err := e.Info()
		if err != nil || time.Since(info.ModTime()) < time.Hour {
			continue
		}
		_ = os.RemoveAll(filepath.Join(s.snapshotsDir(), name))
	}
}

func (s *Store) refresh(ctx context.Context, force, auto bool) (*RefreshResult, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	unlock, err := s.lock(ctx)
	if err != nil {
		s.recordError(err)
		return nil, err
	}
	defer unlock()
	v, err := s.reload()
	if err != nil {
		s.recordError(err)
		return nil, err
	}
	if auto && v != nil && !s.stale(v) {
		return &RefreshResult{Version: v.id}, nil
	}
	roots := s.roots(v)
	mode := modeRevalidate
	if force || v == nil {
		mode = modeForce
	}
	set, err := s.collect(ctx, roots, v, mode)
	if err != nil {
		s.recordError(err)
		s.log.Warn("satresources: no se pudieron revisar los recursos del SAT, se siguen usando los actuales", "error", err)
		return nil, err
	}
	res := &RefreshResult{Checked: len(set.order)}
	if v != nil {
		for _, k := range set.order {
			old := v.m.Files[k]
			switch {
			case old == nil:
				res.Added = append(res.Added, set.files[k].info.URL)
			case old.SHA256 != set.files[k].info.SHA256:
				res.Updated = append(res.Updated, set.files[k].info.URL)
			}
		}
		for k, old := range v.m.Files {
			if set.files[k] == nil {
				res.Removed = append(res.Removed, old.URL)
			}
		}
	}
	unchanged := v != nil && !force && len(res.Added) == 0 && len(res.Updated) == 0 && len(res.Removed) == 0 && !hasFallback(v.m)
	if unchanged {
		m := cloneManifest(v.m)
		m.Checked = s.now().UTC()
		m.Roots = appendRoots(m.Roots, roots)
		for k, e := range set.files {
			m.Files[k] = e.info
		}
		if err := writeManifest(filepath.Join(v.dir, "manifest.json"), m); err != nil {
			s.recordError(err)
			return nil, err
		}
		s.clearError()
		nv, err := s.reload()
		if err != nil {
			return nil, err
		}
		res.Version = nv.id
		return res, nil
	}
	m := &manifest{Format: 1, Created: s.now().UTC(), Checked: s.now().UTC(), Roots: appendRoots(nil, roots), Files: map[string]*fileInfo{}}
	nv, err := s.createSnapshot(m, set, v)
	if err != nil {
		s.recordError(err)
		s.log.Warn("satresources: la versión nueva de los recursos del SAT no se activó, se siguen usando los actuales", "error", err)
		return nil, err
	}
	s.clearError()
	res.Version, res.Changed = nv.id, true
	s.log.Info("satresources: recursos del SAT actualizados", "version", nv.id, "actualizados", len(res.Updated), "nuevos", len(res.Added), "eliminados", len(res.Removed))
	return res, nil
}

func hasFallback(m *manifest) bool {
	for _, f := range m.Files {
		if f.Fallback {
			return true
		}
	}
	return false
}

func cloneManifest(m *manifest) *manifest {
	c := *m
	c.Roots = append([]string(nil), m.Roots...)
	c.Files = make(map[string]*fileInfo, len(m.Files))
	for k, f := range m.Files {
		c.Files[k] = f
	}
	return &c
}

func appendRoots(dst []string, roots []string) []string {
	seen := map[string]bool{}
	for _, r := range dst {
		if k, err := Key(r); err == nil {
			seen[k] = true
		}
	}
	for _, r := range roots {
		k, err := Key(r)
		if err != nil || seen[k] {
			continue
		}
		seen[k] = true
		dst = append(dst, r)
	}
	return dst
}

func newID(now time.Time) string {
	var b [3]byte
	_, _ = rand.Read(b[:])
	return now.UTC().Format("20060102T150405Z") + "-" + hex.EncodeToString(b[:])
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// fallbackSet obtiene del respaldo los recursos indicados y sus
// dependencias. Si no hay versión activa incluye todo el respaldo.
func (s *Store) fallbackSet(fb fs.FS, resources []string, v *view) (*fileSet, []string, error) {
	raw, err := fs.ReadFile(fb, "manifest.json")
	if err != nil {
		return nil, nil, fmt.Errorf("%w: respaldo sin manifest.json: %v", ErrNotAvailable, err)
	}
	fm, err := parseManifest(raw)
	if err != nil {
		return nil, nil, err
	}
	roots := append([]string(nil), resources...)
	if v == nil {
		roots = appendRoots(roots, fm.Roots)
	}
	set := newFileSet()
	queue := append([]string(nil), roots...)
	for len(queue) > 0 {
		r := queue[0]
		queue = queue[1:]
		k, err := s.keyFor(r)
		if err != nil {
			return nil, nil, err
		}
		if set.files[k] != nil || (v != nil && v.m.Files[k] != nil && fileExists(v.filePath(k))) {
			continue
		}
		info := fm.Files[k]
		if info == nil {
			return nil, nil, fmt.Errorf("%w: %s no está en el respaldo", ErrNotAvailable, r)
		}
		data, err := fs.ReadFile(fb, "files/"+k)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: %s: %v", ErrNotAvailable, r, err)
		}
		c := *info
		c.Fallback = true
		set.add(k, &entry{info: &c, data: data})
		queue = append(queue, c.Deps...)
	}
	return set, roots, nil
}

// ---------------------------------------------------------------------------
// Utilerías de archivos

func writeFileAtomic(p string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("satresources: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".tmp-*")
	if err != nil {
		return fmt.Errorf("satresources: %w", err)
	}
	_, werr := tmp.Write(data)
	cerr := tmp.Close()
	if err := errors.Join(werr, cerr); err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("satresources: %w", err)
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("satresources: %w", err)
	}
	if err := os.Rename(tmp.Name(), p); err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("satresources: %w", err)
	}
	return nil
}
