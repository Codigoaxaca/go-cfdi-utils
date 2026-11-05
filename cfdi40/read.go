package cfdi40

import (
	"fmt"
	"os"

	"github.com/codigoaxaca/go-cfdi-utils/nodes"
)

// ReadXML interpreta un contenido XML de CFDI 4.0 y devuelve su Comprobante.
// Devuelve error si el XML no es válido o si el nodo raíz no es un
// cfdi:Comprobante versión 4.0.
//
// Recuerda que los métodos como Emisor() o Impuestos() crean el nodo si no
// existe; para solo leer sin modificar usa SearchNode o SearchAttribute.
func ReadXML(xml []byte) (*Comprobante, error) {
	root, err := nodes.ParseXML(xml)
	if err != nil {
		return nil, fmt.Errorf("cfdi40: %w", err)
	}
	if root.Name() != "cfdi:Comprobante" || root.Attr("Version") != Version {
		return nil, fmt.Errorf("cfdi40: el documento no es un CFDI 4.0 (nodo %s, versión %q)", root.Name(), root.Attr("Version"))
	}
	return WrapComprobante(root), nil
}

// ReadFile lee un archivo XML de CFDI 4.0 y devuelve su Comprobante.
func ReadFile(path string) (*Comprobante, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cfdi40: %w", err)
	}
	return ReadXML(data)
}
