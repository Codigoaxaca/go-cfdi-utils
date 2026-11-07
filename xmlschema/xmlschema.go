// Package xmlschema valida documentos XML contra los esquemas XSD declarados
// en sus atributos xsi:schemaLocation.
//
// Go no cuenta con un validador de XML Schema en su biblioteca estándar, por
// lo que esta implementación usa la herramienta xmllint (libxml2), disponible
// por omisión en macOS y en la mayoría de distribuciones Linux (paquete
// libxml2-utils en Debian/Ubuntu). Los esquemas los obtiene el almacén
// satresources, que los guarda en disco y revisa periódicamente si el SAT
// publicó cambios.
package xmlschema

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/codigoaxaca/go-cfdi-utils/nodes"
	"github.com/codigoaxaca/go-cfdi-utils/satresources"
)

// ErrXmllintNotFound se devuelve cuando no se encuentra xmllint.
var ErrXmllintNotFound = errors.New("xmlschema: no se encontró el ejecutable xmllint")

// DefaultTimeout es lo que puede tardar xmllint en validar un documento si no
// se indica otro límite. Una validación normal tarda menos de un segundo.
const DefaultTimeout = time.Minute

// XmllintValidator valida esquemas usando xmllint.
type XmllintValidator struct {
	// Resources obtiene los esquemas. Si es nil se usa satresources.Default().
	Resources *satresources.Store
	// Executable es la ruta de xmllint.
	Executable string
	// Timeout limita lo que puede tardar xmllint en cada validación; no
	// incluye la descarga de los esquemas. Si es cero o negativo se usa
	// DefaultTimeout. El contexto recibido al validar se respeta si termina
	// antes.
	Timeout time.Duration
}

func (v *XmllintValidator) timeout() time.Duration {
	if v.Timeout > 0 {
		return v.Timeout
	}
	return DefaultTimeout
}

// NewXmllint busca xmllint en el PATH y crea el validador. Si store es nil
// se usa satresources.Default() al validar.
func NewXmllint(store *satresources.Store) (*XmllintValidator, error) {
	path, err := exec.LookPath("xmllint")
	if err != nil {
		return nil, ErrXmllintNotFound
	}
	return &XmllintValidator{Resources: store, Executable: path}, nil
}

// SchemaLocations obtiene todos los pares espacio de nombres / ubicación
// declarados en atributos xsi:schemaLocation de cualquier nodo del documento.
func SchemaLocations(root *nodes.Node) []nodes.SchemaLocation {
	var out []nodes.SchemaLocation
	seen := map[string]bool{}
	root.Walk(func(n *nodes.Node) bool {
		for _, l := range nodes.ParseSchemaLocations(n.Attr("xsi:schemaLocation")) {
			if seen[l.Namespace] {
				continue
			}
			seen[l.Namespace] = true
			out = append(out, l)
		}
		return true
	})
	return out
}

// ValidateSchemas valida el XML contra sus esquemas. Devuelve nil si es válido.
//
// Solo se usan los esquemas publicados en servidores permitidos por el
// almacén (por omisión sat.gob.mx); las ubicaciones de otros servidores, como
// las de algunas addendas, se ignoran.
//
// Los documentos con DOCTYPE se rechazan antes de ejecutar xmllint
// (nodes.ErrDTDNotAllowed), porque xmllint abriría los archivos que indique el
// DTD. Si xmllint no termina dentro del tiempo límite, el error envuelve a
// context.DeadlineExceeded y no es de tipo *Error: el documento no se validó.
func (v *XmllintValidator) ValidateSchemas(ctx context.Context, xml []byte) error {
	store := v.Resources
	if store == nil {
		store = satresources.Default()
	}
	root, err := nodes.ParseXML(xml)
	if err != nil {
		return err
	}
	locations := SchemaLocations(root)
	if len(locations) == 0 {
		return errors.New("xmlschema: el documento no declara esquemas en xsi:schemaLocation")
	}

	imported := 0
	var wrapper strings.Builder
	wrapper.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	wrapper.WriteString(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" elementFormDefault="qualified">` + "\n")
	for _, l := range locations {
		if !strings.HasPrefix(l.Location, "http://") && !strings.HasPrefix(l.Location, "https://") {
			return fmt.Errorf("xmlschema: la ubicación del esquema %q no es una URL", l.Location)
		}
		if !store.Allowed(l.Location) {
			continue
		}
		local, err := store.Path(ctx, l.Location)
		if err != nil {
			return err
		}
		imported++
		abs, err := filepath.Abs(local)
		if err != nil {
			return err
		}
		fmt.Fprintf(&wrapper, `  <xs:import namespace="%s" schemaLocation="%s"/>`+"\n", escapeAttr(l.Namespace), escapeAttr(filepath.ToSlash(abs)))
	}
	wrapper.WriteString(`</xs:schema>` + "\n")
	if imported == 0 {
		return errors.New("xmlschema: el documento no declara esquemas del SAT en xsi:schemaLocation")
	}

	dir, err := os.MkdirTemp("", "go-cfdi-utils-xsd-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	schemaFile := filepath.Join(dir, "schemas.xsd")
	xmlFile := filepath.Join(dir, "document.xml")
	if err := os.WriteFile(schemaFile, []byte(wrapper.String()), 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(xmlFile, xml, 0o600); err != nil {
		return err
	}

	runCtx, cancel := context.WithTimeout(ctx, v.timeout())
	defer cancel()
	cmd := exec.CommandContext(runCtx, v.Executable, "--noout", "--nonet", "--schema", schemaFile, xmlFile)
	// Al terminar el proceso no se espera más de esto a sus procesos hijos, que
	// podrían conservar abierta la salida de errores.
	cmd.WaitDelay = time.Second
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if runCtx.Err() != nil {
			return fmt.Errorf("xmlschema: xmllint no terminó: %w", runCtx.Err())
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return &Error{Messages: cleanMessages(stderr.String(), xmlFile, schemaFile)}
		}
		return fmt.Errorf("xmlschema: no se pudo ejecutar xmllint: %w", err)
	}
	return nil
}

// Error contiene los mensajes de error de validación de esquemas.
type Error struct {
	Messages []string
}

func (e *Error) Error() string {
	if len(e.Messages) == 0 {
		return "el documento no cumple con los esquemas XSD"
	}
	return strings.Join(e.Messages, "; ")
}

func cleanMessages(output, xmlFile, schemaFile string) []string {
	var out []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasSuffix(line, "fails to validate") || strings.HasSuffix(line, "validates") {
			continue
		}
		line = strings.ReplaceAll(line, xmlFile+":", "línea ")
		line = strings.ReplaceAll(line, schemaFile, "esquemas")
		out = append(out, line)
	}
	return out
}

func escapeAttr(s string) string {
	return strings.NewReplacer("&", "&amp;", `"`, "&quot;", "<", "&lt;").Replace(s)
}
