package tfd11

import "github.com/codigoaxaca/go-cfdi-utils/nodes"

// FromComprobante obtiene el timbre existente sin crear nodos ni modificar el CFDI.
func FromComprobante(comprobante nodes.Element) (*TimbreFiscalDigital, bool) {
	if comprobante == nil || comprobante.AsNode() == nil {
		return nil, false
	}
	n := comprobante.AsNode().SearchNode("cfdi:Complemento", "tfd:TimbreFiscalDigital")
	if n == nil {
		return nil, false
	}
	return &TimbreFiscalDigital{n}, true
}

// UUID devuelve el folio fiscal.
func (t *TimbreFiscalDigital) UUID() string { return t.Attr("UUID") }

// FechaTimbrado devuelve la fecha del timbrado.
func (t *TimbreFiscalDigital) FechaTimbrado() string { return t.Attr("FechaTimbrado") }

// SelloCFD devuelve el sello del comprobante.
func (t *TimbreFiscalDigital) SelloCFD() string { return t.Attr("SelloCFD") }

// SelloSAT devuelve el sello del SAT.
func (t *TimbreFiscalDigital) SelloSAT() string { return t.Attr("SelloSAT") }

// NoCertificadoSAT devuelve el número del certificado de timbrado.
func (t *TimbreFiscalDigital) NoCertificadoSAT() string { return t.Attr("NoCertificadoSAT") }

// RfcProvCertif devuelve el RFC del proveedor certificador.
func (t *TimbreFiscalDigital) RfcProvCertif() string { return t.Attr("RfcProvCertif") }
