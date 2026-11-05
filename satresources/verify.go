package satresources

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/codigoaxaca/go-cfdi-utils/internal/xslt"
)

// Documentos mínimos para comprobar que las hojas de estilo de cadena
// original generan una cadena con la forma esperada.
var samples = map[string]struct{ xml, prefix, contains string }{
	XsltCfdi40: {
		xml: `<cfdi:Comprobante xmlns:cfdi="http://www.sat.gob.mx/cfd/4" Version="4.0" Serie="A" Folio="1" Fecha="2024-01-01T00:00:00" SubTotal="1.00" Moneda="MXN" Total="1.16" TipoDeComprobante="I" Exportacion="01" MetodoPago="PUE" FormaPago="01" LugarExpedicion="01000">` +
			`<cfdi:Emisor Rfc="EKU9003173C9" Nombre="ESCUELA KEMPER URGATE" RegimenFiscal="601"/>` +
			`<cfdi:Receptor Rfc="XAXX010101000" Nombre="PUBLICO EN GENERAL" DomicilioFiscalReceptor="01000" RegimenFiscalReceptor="616" UsoCFDI="S01"/>` +
			`<cfdi:Conceptos><cfdi:Concepto ClaveProdServ="01010101" Cantidad="1" ClaveUnidad="ACT" Descripcion="Prueba" ValorUnitario="1.00" Importe="1.00" ObjetoImp="02">` +
			`<cfdi:Impuestos><cfdi:Traslados><cfdi:Traslado Base="1.00" Impuesto="002" TipoFactor="Tasa" TasaOCuota="0.160000" Importe="0.16"/></cfdi:Traslados></cfdi:Impuestos>` +
			`</cfdi:Concepto></cfdi:Conceptos>` +
			`<cfdi:Impuestos TotalImpuestosTrasladados="0.16"><cfdi:Traslados><cfdi:Traslado Base="1.00" Impuesto="002" TipoFactor="Tasa" TasaOCuota="0.160000" Importe="0.16"/></cfdi:Traslados></cfdi:Impuestos>` +
			`</cfdi:Comprobante>`,
		prefix:   "||4.0|A|1|2024-01-01T00:00:00|01|",
		contains: "|EKU9003173C9|ESCUELA KEMPER URGATE|601|XAXX010101000|",
	},
	XsltCfdi33: {
		xml: `<cfdi:Comprobante xmlns:cfdi="http://www.sat.gob.mx/cfd/3" Version="3.3" Fecha="2021-01-01T00:00:00" FormaPago="01" SubTotal="1.00" Moneda="MXN" Total="1.00" TipoDeComprobante="I" MetodoPago="PUE" LugarExpedicion="01000">` +
			`<cfdi:Emisor Rfc="EKU9003173C9" Nombre="ESCUELA KEMPER URGATE" RegimenFiscal="601"/><cfdi:Receptor Rfc="XAXX010101000" UsoCFDI="G03"/>` +
			`<cfdi:Conceptos><cfdi:Concepto ClaveProdServ="01010101" Cantidad="1" ClaveUnidad="ACT" Descripcion="Prueba" ValorUnitario="1.00" Importe="1.00"/></cfdi:Conceptos>` +
			`</cfdi:Comprobante>`,
		prefix:   "||3.3|2021-01-01T00:00:00|01|",
		contains: "|EKU9003173C9|ESCUELA KEMPER URGATE|601|XAXX010101000|G03|",
	},
	XsltTfd11: {
		xml:      `<tfd:TimbreFiscalDigital xmlns:tfd="http://www.sat.gob.mx/TimbreFiscalDigital" Version="1.1" UUID="00000000-0000-0000-0000-000000000000" FechaTimbrado="2024-01-01T00:00:00" RfcProvCertif="SAT970701NN3" SelloCFD="c2VsbG8=" NoCertificadoSAT="00001000000000000000"/>`,
		prefix:   "||1.1|00000000-0000-0000-0000-000000000000|2024-01-01T00:00:00|SAT970701NN3|",
		contains: "|c2VsbG8=|00001000000000000000||",
	},
	XsltRetenciones20: {
		xml: `<retenciones:Retenciones xmlns:retenciones="http://www.sat.gob.mx/esquemas/retencionpago/2" Version="2.0" NoCertificado="00001000000000000000" FechaExp="2024-01-01T00:00:00" LugarExpRetenc="01000" CveRetenc="16">` +
			`<retenciones:Emisor RfcE="EKU9003173C9" NomDenRazSocE="ESCUELA KEMPER URGATE" RegimenFiscalE="601"/>` +
			`<retenciones:Receptor NacionalidadR="Nacional"><retenciones:Nacional RfcR="XAXX010101000" NomDenRazSocR="PUBLICO EN GENERAL" DomicilioFiscalR="01000"/></retenciones:Receptor>` +
			`<retenciones:Periodo MesIni="01" MesFin="01" Ejercicio="2024"/>` +
			`<retenciones:Totales MontoTotOperacion="1.00" MontoTotGrav="1.00" MontoTotExent="0.00" MontoTotRet="0.00"/>` +
			`</retenciones:Retenciones>`,
		prefix:   "||2.0|00001000000000000000|2024-01-01T00:00:00|01000|16|",
		contains: "|EKU9003173C9|ESCUELA KEMPER URGATE|601|",
	},
}

// verify comprueba un conjunto de archivos antes de guardarlo o activarlo.
// base es la versión activa (para dependencias que ya están guardadas) o nil;
// dir es el directorio donde ya están escritos todos los archivos (el de la
// versión nueva o el de la versión activa), para compilar los XSD con xmllint.
func (s *Store) verify(set *fileSet, base *view, dir string) error {
	info := func(k string) *fileInfo {
		if e := set.files[k]; e != nil {
			return e.info
		}
		if base != nil {
			return base.m.Files[k]
		}
		return nil
	}
	get := func(k string) ([]byte, error) {
		if e := set.files[k]; e != nil {
			return e.data, nil
		}
		if base != nil && base.m.Files[k] != nil {
			return os.ReadFile(base.filePath(k))
		}
		return nil, fmt.Errorf("falta %s", k)
	}
	var errs []error
	for _, k := range set.order {
		e := set.files[k]
		for _, dep := range e.info.Deps {
			dk, err := s.keyFor(dep)
			if err == nil && info(dk) == nil {
				err = fmt.Errorf("falta %s", dk)
			}
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: dependencia %s: %w", e.info.URL, dep, err))
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%w: %w", ErrVerification, errors.Join(errs...))
	}

	load := func(loc string) ([]byte, error) {
		k, err := s.keyFor(loc)
		if err != nil {
			return nil, err
		}
		return get(k)
	}
	for _, k := range verifyTargets(kindXSLT, set, base, info) {
		f := info(k)
		sheet, err := xslt.Compile(f.URL, load)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", f.URL, err))
			continue
		}
		for loc, sample := range samples {
			if sk, _ := Key(loc); sk != k {
				continue
			}
			doc, err := xslt.Parse([]byte(sample.xml))
			if err != nil {
				errs = append(errs, err)
				continue
			}
			out, err := sheet.Transform(doc)
			switch {
			case err != nil:
				errs = append(errs, fmt.Errorf("%s: cadena de prueba: %w", f.URL, err))
			case !strings.HasPrefix(out, sample.prefix) || !strings.HasSuffix(out, "||") || !strings.Contains(out, sample.contains):
				errs = append(errs, fmt.Errorf("%s: cadena de prueba inesperada: %q", f.URL, out))
			}
		}
	}
	if dir != "" && len(errs) == 0 {
		for _, k := range verifyTargets(kindXSD, set, base, info) {
			if err := compileSchema(filepath.Join(dir, "files", filepath.FromSlash(k))); err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", info(k).URL, err))
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%w: %w", ErrVerification, errors.Join(errs...))
	}
	return nil
}

// verifyTargets elige qué archivos del tipo indicado se compilan para que
// cada archivo del conjunto quede cubierto: los puntos de entrada (archivos
// que nadie incluye) cuya clausura contiene archivos del conjunto, y para
// lo que no quede cubierto (por ejemplo un ciclo de inclusiones), el propio
// archivo.
func verifyTargets(kind string, set *fileSet, base *view, info func(string) *fileInfo) []string {
	keys := append([]string(nil), set.order...)
	if base != nil {
		var extra []string
		for k := range base.m.Files {
			if set.files[k] == nil {
				extra = append(extra, k)
			}
		}
		sort.Strings(extra)
		keys = append(keys, extra...)
	}
	depKeys := func(k string) []string {
		f := info(k)
		if f == nil {
			return nil
		}
		var out []string
		for _, d := range f.Deps {
			if dk, err := Key(d); err == nil {
				out = append(out, dk)
			}
		}
		return out
	}
	included := map[string]bool{}
	for _, k := range keys {
		for _, d := range depKeys(k) {
			included[d] = true
		}
	}
	closures := map[string]map[string]bool{}
	closure := func(root string) map[string]bool {
		if c := closures[root]; c != nil {
			return c
		}
		c := map[string]bool{}
		stack := []string{root}
		for len(stack) > 0 {
			k := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if c[k] {
				continue
			}
			c[k] = true
			stack = append(stack, depKeys(k)...)
		}
		closures[root] = c
		return c
	}
	var tops []string
	for _, k := range keys {
		if f := info(k); f != nil && f.Kind == kind && !included[k] {
			tops = append(tops, k)
		}
	}
	var targets []string
	covered := map[string]bool{}
	for _, k := range set.order {
		if set.files[k].info.Kind != kind || covered[k] {
			continue
		}
		target := k
		for _, t := range tops {
			if closure(t)[k] {
				target = t
				break
			}
		}
		targets = append(targets, target)
		for c := range closure(target) {
			covered[c] = true
		}
	}
	return targets
}

// compileSchema compila un XSD con xmllint, si está instalado, validando un
// documento de prueba: xmllint termina con 0, 3 o 4 cuando el esquema
// compila (el documento de prueba no es válido) y con 5 cuando no compila.
// Cualquier otro resultado (tiempo agotado, proceso interrumpido) es un error.
func compileSchema(xsd string) error {
	exe, err := exec.LookPath("xmllint")
	if err != nil {
		return nil
	}
	tmp, err := os.MkdirTemp("", "satresources-xsd-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	probe := filepath.Join(tmp, "probe.xml")
	if err := os.WriteFile(probe, []byte("<probe/>"), 0o644); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, "--noout", "--nonet", "--schema", xsd, probe)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err = cmd.Run()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return nil
	case ctx.Err() != nil:
		return fmt.Errorf("xmllint no terminó: %w", ctx.Err())
	case errors.As(err, &exitErr) && (exitErr.ExitCode() == 3 || exitErr.ExitCode() == 4):
		return nil
	}
	msg := strings.TrimSpace(stderr.String())
	if len(msg) > 2000 {
		msg = msg[:2000] + "…"
	}
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 5 {
		return fmt.Errorf("el esquema no compila con xmllint: %s", msg)
	}
	return fmt.Errorf("xmllint: %v: %s", err, msg)
}
