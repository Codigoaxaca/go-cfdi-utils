package xmlschema

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/codigoaxaca/go-cfdi-utils/nodes"
	"github.com/codigoaxaca/go-cfdi-utils/xmlresolver"
)

func TestSchemaLocations(t *testing.T) {
	root, err := nodes.ParseXMLFile("../testdata/cfdi/cfdi40-real.xml")
	if err != nil {
		t.Fatal(err)
	}
	locs := SchemaLocations(root)
	if len(locs) < 2 || locs[0].Namespace != "http://www.sat.gob.mx/cfd/4" {
		t.Fatalf("ubicaciones inesperadas %+v", locs)
	}
}

func TestValidateSchemas(t *testing.T) {
	v, err := NewXmllint(xmlresolver.New())
	if errors.Is(err, ErrXmllintNotFound) {
		t.Skip("xmllint no está instalado")
	}
	data, err := os.ReadFile("../testdata/cfdi/cfdi40-real.xml")
	if err != nil {
		t.Fatal(err)
	}
	if err := v.ValidateSchemas(context.Background(), data); err != nil {
		t.Fatalf("el CFDI real debería ser válido: %v", err)
	}
	invalid := strings.Replace(string(data), `Exportacion="01"`, `Exportacion="99"`, 1)
	err = v.ValidateSchemas(context.Background(), []byte(invalid))
	var schemaErr *Error
	if !errors.As(err, &schemaErr) || !strings.Contains(err.Error(), "Exportacion") {
		t.Fatalf("se esperaba un error de esquema por Exportacion, se obtuvo %v", err)
	}
	t.Log(err)
}
