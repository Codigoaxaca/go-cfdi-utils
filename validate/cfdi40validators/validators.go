// Package cfdi40validators contiene los validadores de CFDI 4.0.
//
// Normalmente no se usan directamente: cfdi40.Creator.Validate() y
// cfdi40.Validator los ejecutan. Se exportan para poder armar un
// validate.MultiValidator a la medida.
package cfdi40validators

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/codigoaxaca/go-cfdi-utils/cadenaorigen"
	"github.com/codigoaxaca/go-cfdi-utils/credentials"
	"github.com/codigoaxaca/go-cfdi-utils/nodes"
	"github.com/codigoaxaca/go-cfdi-utils/validate"
)

const (
	cfdi40Namespace = "http://www.sat.gob.mx/cfd/4"
	tfdNamespace    = "http://www.sat.gob.mx/TimbreFiscalDigital"
)

// NewCreatedValidator devuelve el conjunto de validadores para CFDI 4.0 creados.
func NewCreatedValidator() *validate.MultiValidator {
	return validate.NewMultiValidator("4.0", append(baseValidators(), StandardValidators()...)...)
}

// NewReceivedValidator devuelve el conjunto de validadores para CFDI 4.0 recibidos.
func NewReceivedValidator() *validate.MultiValidator {
	return validate.NewMultiValidator("4.0", append(baseValidators(), StandardValidators()...)...)
}

func baseValidators() []validate.Validator {
	return []validate.Validator{
		XmlFollowSchema{},
		XmlDefinition{},
		SelloDigitalCertificado{},
		TimbreFiscalDigitalSello{},
		TimbreFiscalDigitalVersion{},
	}
}

// version40 es la base de los validadores que solo aplican a CFDI 4.0.
type version40 struct{}

// CanValidateCfdiVersion devuelve verdadero para la versión 4.0.
func (version40) CanValidateCfdiVersion(version string) bool { return version == "4.0" }

// ---------------------------------------------------------------------------

// XmlFollowSchema valida que el XML siga los esquemas XSD declarados.
//
//   - XSD01: El contenido XML sigue los esquemas XSD
//
// Si falla, detiene el resto de validaciones (MustStop).
type XmlFollowSchema struct{ version40 }

// Validate implementa validate.Validator.
func (XmlFollowSchema) Validate(ctx context.Context, _ *nodes.Node, env *validate.Environment, asserts *validate.Asserts) {
	a := asserts.Put("XSD01", "El contenido XML sigue los esquemas XSD")
	if env.SchemaValidator == nil {
		a.SetExplanation("No se realizó la validación porque no se cuenta con un validador de esquemas (xmllint)")
		return
	}
	if len(env.XML) == 0 {
		a.SetExplanation("No se realizó la validación porque no se cuenta con el contenido XML")
		return
	}
	if err := env.SchemaValidator.ValidateSchemas(ctx, env.XML); err != nil {
		a.SetStatus(validate.StatusError, err.Error())
		asserts.SetMustStop(true)
		return
	}
	a.SetStatus(validate.StatusOk)
}

// ---------------------------------------------------------------------------

// XmlDefinition valida que el XML tenga las especificaciones de CFDI 4.0.
//
//   - XML01: El XML implementa el namespace http://www.sat.gob.mx/cfd/4 con el prefijo cfdi
//   - XML02: El nodo principal se llama cfdi:Comprobante
//   - XML03: La versión es 4.0
type XmlDefinition struct{ version40 }

// Validate implementa validate.Validator.
func (XmlDefinition) Validate(_ context.Context, c *nodes.Node, _ *validate.Environment, asserts *validate.Asserts) {
	asserts.Put("XML01", "El XML implementa el namespace "+cfdi40Namespace+" con el prefijo cfdi").
		SetStatus(validate.StatusWhen(c.Attr("xmlns:cfdi") == cfdi40Namespace), "Valor de xmlns:cfdi: "+c.Attr("xmlns:cfdi"))
	asserts.Put("XML02", "El nodo principal se llama cfdi:Comprobante").
		SetStatus(validate.StatusWhen(c.Name() == "cfdi:Comprobante"), "Nombre: "+c.Name())
	asserts.Put("XML03", "La versión es 4.0").
		SetStatus(validate.StatusWhen(c.Attr("Version") == "4.0"), "Versión: "+c.Attr("Version"))
}

// ---------------------------------------------------------------------------

// SelloDigitalCertificado valida el sello del comprobante y su certificado.
//
//   - SELLO01: Se puede obtener el certificado del comprobante
//   - SELLO02: El número de certificado del comprobante igual al encontrado en el certificado
//   - SELLO03: El RFC del comprobante igual al encontrado en el certificado
//   - SELLO04: El nombre del emisor del comprobante es igual al encontrado en el certificado
//   - SELLO05: La fecha del documento es mayor o igual a la fecha de inicio de vigencia del certificado
//   - SELLO06: La fecha del documento menor o igual a la fecha de fin de vigencia del certificado
//   - SELLO07: El sello del comprobante está en base 64
//   - SELLO08: El sello del comprobante coincide con el certificado y la cadena de origen generada
type SelloDigitalCertificado struct{ version40 }

// Validate implementa validate.Validator.
func (SelloDigitalCertificado) Validate(_ context.Context, c *nodes.Node, env *validate.Environment, asserts *validate.Asserts) {
	titles := [][2]string{
		{"SELLO01", "Se puede obtener el certificado del comprobante"},
		{"SELLO02", "El número de certificado del comprobante igual al encontrado en el certificado"},
		{"SELLO03", "El RFC del comprobante igual al encontrado en el certificado"},
		{"SELLO04", "El nombre del emisor del comprobante es igual al encontrado en el certificado"},
		{"SELLO05", "La fecha del documento es mayor o igual a la fecha de inicio de vigencia del certificado"},
		{"SELLO06", "La fecha del documento menor o igual a la fecha de fin de vigencia del certificado"},
		{"SELLO07", "El sello del comprobante está en base 64"},
		{"SELLO08", "El sello del comprobante coincide con el certificado y la cadena de origen generada"},
	}
	for _, t := range titles {
		asserts.Put(t[0], t[1])
	}

	if strings.TrimSpace(c.Attr("Certificado")) == "" {
		asserts.PutStatus("SELLO01", validate.StatusError, "El comprobante no contiene el atributo Certificado")
		return
	}
	cert, err := credentials.ParseCertificate([]byte(c.Attr("Certificado")))
	if err != nil {
		asserts.PutStatus("SELLO01", validate.StatusError, err.Error())
		return
	}
	asserts.PutStatus("SELLO01", validate.StatusOk)

	// SELLO02
	noCertificado := c.Attr("NoCertificado")
	asserts.PutStatus("SELLO02", validate.StatusWhen(cert.NoCertificado() == noCertificado),
		fmt.Sprintf("Certificado: %s, Comprobante: %s", cert.NoCertificado(), noCertificado))

	// SELLO03 y SELLO04, no aplican cuando el CFDI es de registro fiscal
	// emitido con el certificado del SAT
	hasRegistroFiscal := len(c.SearchNodes("cfdi:Complemento", "registrofiscal:CFDIRegistroFiscal")) > 0
	noCertificadoSAT := c.SearchAttribute("cfdi:Complemento", "tfd:TimbreFiscalDigital", "NoCertificadoSAT")
	if !hasRegistroFiscal || noCertificado != noCertificadoSAT {
		emisorRfc := c.SearchAttribute("cfdi:Emisor", "Rfc")
		asserts.PutStatus("SELLO03", validate.StatusWhen(cert.Rfc() == emisorRfc),
			fmt.Sprintf("Rfc certificado: %s, Rfc comprobante: %s", cert.Rfc(), emisorRfc))

		emisorNombre := c.SearchAttribute("cfdi:Emisor", "Nombre")
		if emisorNombre == "" {
			asserts.PutStatus("SELLO04", validate.StatusError, "Nombre del emisor vacío")
		} else {
			expected := cert.Name()
			if len([]rune(emisorRfc)) == 12 {
				expected = cert.NameWithoutRegimenCapital()
			}
			asserts.PutStatus("SELLO04", validate.StatusWhen(expected == emisorNombre),
				fmt.Sprintf("Nombre certificado: %s, Nombre comprobante: %s.", cert.Name(), emisorNombre))
		}
	}

	// SELLO05 y SELLO06
	if fecha, ok := ParseFecha(c.Attr("Fecha")); ok {
		const layout = "2006-01-02 15:04:05"
		explanation := fmt.Sprintf("Validez del certificado: %s hasta %s, Fecha comprobante: %s",
			cert.ValidFrom().In(MexicoLocation).Format(layout),
			cert.ValidTo().In(MexicoLocation).Format(layout),
			fecha.Format(layout))
		asserts.PutStatus("SELLO05", validate.StatusWhen(!fecha.Before(cert.ValidFrom())), explanation)
		asserts.PutStatus("SELLO06", validate.StatusWhen(!fecha.After(cert.ValidTo())), explanation)
	}

	// SELLO07
	sello, err := base64.StdEncoding.DecodeString(c.Attr("Sello"))
	asserts.PutStatus("SELLO07", validate.StatusWhen(err == nil && len(sello) > 0))
	if err != nil || len(sello) == 0 {
		return
	}

	// SELLO08
	if env.CadenaOrigen == nil || len(env.XML) == 0 {
		asserts.Get("SELLO08").SetExplanation("No se puede generar la cadena de origen porque no hay un generador o contenido XML")
		return
	}
	cadena, err := env.CadenaOrigen.Build(env.XML, cadenaorigen.XsltCfdi40)
	if err != nil {
		asserts.PutStatus("SELLO08", validate.StatusError, "No se pudo generar la cadena de origen: "+err.Error())
		return
	}
	if !cert.Verify([]byte(cadena), sello) {
		asserts.PutStatus("SELLO08", validate.StatusError,
			"La verificación del sello del CFDI no coincide, probablemente el CFDI fue alterado o mal generado")
		return
	}
	asserts.PutStatus("SELLO08", validate.StatusOk)
}

// MexicoLocation es la zona horaria usada para interpretar el atributo Fecha
// de los comprobantes (hora del centro de México, UTC-6, sin horario de verano
// desde 2022).
var MexicoLocation = time.FixedZone("CST", -6*60*60)

// ParseFecha interpreta una fecha de CFDI con formato AAAA-MM-DDThh:mm:ss.
func ParseFecha(value string) (time.Time, bool) {
	t, err := time.ParseInLocation("2006-01-02T15:04:05", strings.TrimSpace(value), MexicoLocation)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// ---------------------------------------------------------------------------

// TimbreFiscalDigitalSello valida que el sello del SAT en el Timbre Fiscal
// Digital corresponde al certificado del SAT, es decir, que el CFDI no fue
// modificado después de timbrarse.
//
//   - TFDSELLO01: El Sello SAT del Timbre Fiscal Digital corresponde al certificado SAT
//
// Requiere descargar el certificado del SAT (https://rdc.sat.gob.mx/) usando
// el Resolver del entorno.
type TimbreFiscalDigitalSello struct{ version40 }

// Validate implementa validate.Validator.
func (TimbreFiscalDigitalSello) Validate(ctx context.Context, c *nodes.Node, env *validate.Environment, asserts *validate.Asserts) {
	a := asserts.Put("TFDSELLO01", "El Sello SAT del Timbre Fiscal Digital corresponde al certificado SAT")
	if env.Resolver == nil {
		a.SetExplanation("No se puede hacer la validación porque carece de un objeto resolvedor")
		return
	}
	tfd := c.SearchNode("cfdi:Complemento", "tfd:TimbreFiscalDigital")
	if tfd == nil {
		a.SetExplanation("El CFDI no contiene un Timbre Fiscal Digital")
		return
	}
	if tfd.Attr("Version") != "1.1" {
		a.SetExplanation("La versión del timbre fiscal digital no es 1.1")
		return
	}
	if c.Attr("Sello") != tfd.Attr("SelloCFD") {
		a.SetStatus(validate.StatusError, "El atributo SelloCFD del Timbre Fiscal Digital no coincide con el atributo Sello del Comprobante")
		return
	}
	noCertificadoSAT := tfd.Attr("NoCertificadoSAT")
	if !credentials.IsValidNoCertificado(noCertificadoSAT) {
		a.SetStatus(validate.StatusError, fmt.Sprintf("El atributo NoCertificadoSAT con el valor %q no es válido", noCertificadoSAT))
		return
	}
	cert, err := env.Resolver.SatCertificate(ctx, noCertificadoSAT)
	if err != nil {
		a.SetStatus(validate.StatusError, fmt.Sprintf("No se ha podido obtener el certificado %s: %s", noCertificadoSAT, err))
		return
	}
	if env.CadenaOrigen == nil {
		a.SetExplanation("No se puede hacer la validación porque no hay un generador de cadena de origen")
		return
	}
	cadena, err := env.CadenaOrigen.Build([]byte(TfdXML(tfd)), cadenaorigen.XsltTfd11)
	if err != nil {
		a.SetStatus(validate.StatusError, "No se pudo generar la cadena de origen del timbre: "+err.Error())
		return
	}
	selloSAT, _ := base64.StdEncoding.DecodeString(tfd.Attr("SelloSAT"))
	if !cert.Verify([]byte(cadena), selloSAT) {
		a.SetStatus(validate.StatusError, "La verificación del timbrado fue negativa, posiblemente el CFDI fue modificado después de generar el sello")
		return
	}
	a.SetStatus(validate.StatusOk)
}

// TfdXML genera el XML independiente de un nodo tfd:TimbreFiscalDigital
// asegurando que incluya la declaración de su espacio de nombres (que
// normalmente está en el nodo raíz del CFDI).
func TfdXML(tfd *nodes.Node) string {
	clone := tfd.Clone()
	if !clone.HasAttr("xmlns:tfd") {
		clone.SetAttr("xmlns:tfd", tfdNamespace)
	}
	if clone.HasAttr("xsi:schemaLocation") && !clone.HasAttr("xmlns:xsi") {
		clone.SetAttr("xmlns:xsi", "http://www.w3.org/2001/XMLSchema-instance")
	}
	return clone.String()
}

// ---------------------------------------------------------------------------

// TimbreFiscalDigitalVersion valida la versión del Timbre Fiscal Digital.
//
//   - TFDVERSION01: Si existe el complemento timbre fiscal digital, entonces su versión debe ser 1.1
type TimbreFiscalDigitalVersion struct{ version40 }

// Validate implementa validate.Validator.
func (TimbreFiscalDigitalVersion) Validate(_ context.Context, c *nodes.Node, _ *validate.Environment, asserts *validate.Asserts) {
	a := asserts.Put("TFDVERSION01", "Si existe el complemento timbre fiscal digital, entonces su versión debe ser 1.1")
	if tfd := c.SearchNode("cfdi:Complemento", "tfd:TimbreFiscalDigital"); tfd != nil {
		a.SetStatus(validate.StatusWhen(tfd.Attr("Version") == "1.1"))
	}
}
