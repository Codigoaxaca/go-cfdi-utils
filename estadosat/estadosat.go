// Package estadosat consulta el estado de un CFDI en el servicio web del SAT
// (https://consultaqr.facturaelectronica.sat.gob.mx/ConsultaCFDIService.svc).
//
// El servicio recibe la "expresión impresa", que es la misma información
// contenida en el código QR de la representación impresa del CFDI.
//
//	params, err := estadosat.RequestParametersFromXML(xmlBytes)
//	ws := estadosat.NewWebService(nil)
//	resp, err := ws.Request(ctx, params)
//	fmt.Println(resp.Estado) // Vigente
package estadosat

import (
	"errors"
	"fmt"
	"strings"

	"github.com/codigoaxaca/go-cfdi-utils/internal/decimal"
	"github.com/codigoaxaca/go-cfdi-utils/nodes"
)

// ErrInvalidVersion se devuelve cuando la versión del CFDI no es 3.2, 3.3 o 4.0.
var ErrInvalidVersion = errors.New("estadosat: la versión del CFDI no está soportada (3.2, 3.3 o 4.0)")

// VerificaCfdiURL es la URL del sitio de verificación usada en la expresión
// impresa de CFDI 3.3 y 4.0 (código QR).
const VerificaCfdiURL = "https://verificacfdi.facturaelectronica.sat.gob.mx/default.aspx"

// RequestParameters son los datos de un CFDI necesarios para consultar su estado.
type RequestParameters struct {
	Version     string
	RfcEmisor   string
	RfcReceptor string
	// Total es el total tal como aparece en el CFDI (puede contener comas de millares).
	Total string
	UUID  string
	// Sello es el sello del comprobante (atributo Sello), se usan los últimos 8 caracteres.
	Sello string
}

// NewRequestParameters crea los parámetros validando la versión.
func NewRequestParameters(version, rfcEmisor, rfcReceptor, total, uuid, sello string) (*RequestParameters, error) {
	p := &RequestParameters{
		Version:     version,
		RfcEmisor:   rfcEmisor,
		RfcReceptor: rfcReceptor,
		Total:       total,
		UUID:        uuid,
		Sello:       sello,
	}
	if err := p.validateVersion(); err != nil {
		return nil, err
	}
	return p, nil
}

// RequestParametersFromXML obtiene los parámetros desde el contenido XML de un CFDI timbrado.
func RequestParametersFromXML(xml []byte) (*RequestParameters, error) {
	root, err := nodes.ParseXML(xml)
	if err != nil {
		return nil, fmt.Errorf("estadosat: %w", err)
	}
	return RequestParametersFromNode(root)
}

// RequestParametersFromNode obtiene los parámetros desde el nodo raíz de un
// CFDI timbrado. La lectura no distingue prefijos ni mayúsculas, por lo que
// funciona con CFDI 3.2, 3.3 y 4.0.
func RequestParametersFromNode(comprobante *nodes.Node) (*RequestParameters, error) {
	emisor := childCI(comprobante, "Emisor")
	receptor := childCI(comprobante, "Receptor")
	tfd := childCI(childCI(comprobante, "Complemento"), "TimbreFiscalDigital")
	p := &RequestParameters{
		Version:     attrCI(comprobante, "Version"),
		RfcEmisor:   attrCI(emisor, "Rfc"),
		RfcReceptor: attrCI(receptor, "Rfc"),
		Total:       attrCI(comprobante, "Total"),
		UUID:        attrCI(tfd, "UUID"),
		Sello:       attrCI(comprobante, "Sello"),
	}
	if err := p.validateVersion(); err != nil {
		return nil, err
	}
	if p.UUID == "" {
		return nil, errors.New("estadosat: el CFDI no contiene el UUID del TimbreFiscalDigital")
	}
	return p, nil
}

func (p *RequestParameters) validateVersion() error {
	switch p.Version {
	case "3.2", "3.3", "4.0":
		return nil
	}
	return fmt.Errorf("%w: %q", ErrInvalidVersion, p.Version)
}

// TotalFloat devuelve el total como número.
func (p *RequestParameters) TotalFloat() float64 { return decimal.Parse(p.Total) }

// Expression devuelve la expresión impresa según la versión del CFDI. Para
// CFDI 3.3 y 4.0 es la URL del código QR, por ejemplo:
//
//	https://verificacfdi.facturaelectronica.sat.gob.mx/default.aspx?id=...&re=...&rr=...&tt=1234.5&fe=...
func (p *RequestParameters) Expression() string {
	switch p.Version {
	case "3.2":
		total := decimal.Format(p.TotalFloat(), 6)
		if len(total) < 17 {
			total = strings.Repeat("0", 17-len(total)) + total
		}
		return "?" + strings.Join([]string{
			"re=" + xmlEntities(p.RfcEmisor),
			"rr=" + xmlEntities(p.RfcReceptor),
			"tt=" + total,
			"id=" + p.UUID,
		}, "&")
	case "3.3", "4.0":
		total := strings.TrimRight(decimal.Format(p.TotalFloat(), 6), "0")
		if strings.HasSuffix(total, ".") {
			total += "0"
		}
		sello := p.Sello
		if len(sello) > 8 {
			sello = sello[len(sello)-8:]
		}
		return VerificaCfdiURL + "?" + strings.Join([]string{
			"id=" + p.UUID,
			"re=" + xmlEntities(p.RfcEmisor),
			"rr=" + xmlEntities(p.RfcReceptor),
			"tt=" + total,
			"fe=" + sello,
		}, "&")
	}
	return ""
}

// QRText es un alias de Expression, el texto que debe contener el código QR.
func (p *RequestParameters) QRText() string { return p.Expression() }

func xmlEntities(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;").Replace(s)
}

func localName(name string) string {
	if i := strings.LastIndexByte(name, ':'); i >= 0 {
		return name[i+1:]
	}
	return name
}

func childCI(n *nodes.Node, name string) *nodes.Node {
	if n == nil {
		return nil
	}
	for _, c := range n.Children() {
		if strings.EqualFold(localName(c.Name()), name) {
			return c
		}
	}
	return nil
}

func attrCI(n *nodes.Node, name string) string {
	if n == nil {
		return ""
	}
	if v, ok := n.LookupAttr(name); ok {
		return v
	}
	for _, a := range n.Attributes() {
		if strings.EqualFold(a.Name, name) {
			return a.Value
		}
	}
	return ""
}
