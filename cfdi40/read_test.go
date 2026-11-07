package cfdi40_test

import (
	"errors"
	"testing"

	"github.com/codigoaxaca/go-cfdi-utils/cfdi40"
	"github.com/codigoaxaca/go-cfdi-utils/nodes"
)

func TestDTDIsRejected(t *testing.T) {
	doc := []byte(`<?xml version="1.0"?><!DOCTYPE a [<!ENTITY % x SYSTEM "file:///etc/passwd"> %x;]><cfdi:Comprobante xmlns:cfdi="http://www.sat.gob.mx/cfd/4" Version="4.0"/>`)
	if _, err := cfdi40.ReadXML(doc); !errors.Is(err, nodes.ErrDTDNotAllowed) {
		t.Errorf("ReadXML: se esperaba ErrDTDNotAllowed, se obtuvo %v", err)
	}
	if _, err := cfdi40.ValidateXML(doc); !errors.Is(err, nodes.ErrDTDNotAllowed) {
		t.Errorf("ValidateXML: se esperaba ErrDTDNotAllowed, se obtuvo %v", err)
	}
}

func TestReadFile(t *testing.T) {
	c, err := cfdi40.ReadFile("../testdata/cfdi/cfdi40-real.xml")
	if err != nil {
		t.Fatal(err)
	}
	if c.SearchAttribute("cfdi:Emisor", "Rfc") != "ISD950921HE5" {
		t.Fatal("rfc emisor inesperado")
	}
	if len(c.Conceptos().Items()) == 0 {
		t.Fatal("sin conceptos")
	}
	if _, err := cfdi40.ReadFile("../testdata/cfdi/cfdi33-real.xml"); err == nil {
		t.Fatal("se esperaba error con CFDI 3.3")
	}
}
