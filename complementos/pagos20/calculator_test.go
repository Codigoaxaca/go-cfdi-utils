package pagos20

import (
	"github.com/codigoaxaca/go-cfdi-utils/nodes"
	"path/filepath"
	"testing"
)

func equalDecimal(t *testing.T, a, b string) {
	t.Helper()
	x, e := number(a)
	if e != nil {
		t.Fatal(e)
	}
	y, e := number(b)
	if e != nil {
		t.Fatal(e)
	}
	if x.Cmp(y) != 0 {
		t.Fatalf("esperado %s, recibido %s", a, b)
	}
}

func TestPHPFixtures(t *testing.T) {
	files, e := filepath.Glob("../../testdata/pagos20-calculator/*.xml")
	if e != nil {
		t.Fatal(e)
	}
	if len(files) != 8 {
		t.Fatalf("se esperaban ocho fixtures: %d", len(files))
	}
	files = append(files, "../../testdata/cfdi/created-cfdi40-pago20-valid.xml")
	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			doc, e := nodes.ParseXMLFile(file)
			if e != nil {
				t.Fatal(e)
			}
			p := doc.SearchNode("cfdi:Complemento", "pago20:Pagos")
			r, e := Calculate(p)
			if e != nil {
				t.Fatal(e)
			}
			expected := p.FirstChild("pago20:Totales")
			if len(r.Totales) != len(expected.Attributes()) {
				t.Fatalf("totales diferentes: %v frente a %v", r.Totales, expected.AttrsMap())
			}
			for _, a := range expected.Attributes() {
				equalDecimal(t, a.Value, r.Totales[a.Name])
			}
			for i, payment := range p.ChildrenNamed("pago20:Pago") {
				pr := r.Pagos[i]
				if mustNumber(pr.MontoMinimo).Cmp(mustNumber(payment.Attr("Monto"))) > 0 {
					t.Fatal("monto inferior al mínimo")
				}
				count := 0
				for _, kind := range []string{"Traslado", "Retencion"} {
					group := "TrasladosP"
					if kind == "Retencion" {
						group = "RetencionesP"
					}
					for _, tax := range payment.SearchNodes("pago20:ImpuestosP", "pago20:"+group, "pago20:"+kind+"P") {
						count++
						found := false
						for _, actual := range pr.Impuestos {
							if actual.Tipo != kind || actual.Impuesto != tax.Attr("ImpuestoP") || actual.TipoFactor != tax.Attr("TipoFactorP") || actual.TasaOCuota != tax.Attr("TasaOCuotaP") {
								continue
							}
							found = true
							if kind == "Traslado" {
								equalDecimal(t, tax.Attr("BaseP"), actual.Base)
							}
							equalDecimal(t, tax.Attr("ImporteP"), actual.Importe)
						}
						if !found {
							t.Fatalf("falta impuesto %s", tax)
						}
					}
				}
				if count != len(pr.Impuestos) {
					t.Fatal("cantidad de impuestos diferente")
				}
			}
			target := &Pagos{p.Clone()}
			if e := Write(target, r); e != nil {
				t.Fatal(e)
			}
			once := target.String()
			if e := Write(target, r); e != nil {
				t.Fatal(e)
			}
			if once != target.String() {
				t.Fatal("escritura no idempotente")
			}
			if _, e := nodes.ParseXMLString(target.String()); e != nil {
				t.Fatal(e)
			}
		})
	}
}

func TestRoundingAndTruncation(t *testing.T) {
	p := NewPagos(nil)
	payment := p.AddPago(Attrs{"MonedaP": "MXN", "TipoCambioP": "1"})
	payment.AddDoctoRelacionado(Attrs{"EquivalenciaDR": "1", "ImpPagado": "0.14"}).
		ImpuestosDR().TrasladosDR().AddTrasladoDR(Attrs{"BaseDR": "0.123456789", "ImporteDR": "0.01975308624", "ImpuestoDR": "002", "TipoFactorDR": "Tasa", "TasaOCuotaDR": "0.160000"})
	for _, tt := range []struct {
		opts         []Option
		base, amount string
	}{
		{nil, "0.123457", "0.019753"},
		{[]Option{WithPaymentTaxesPrecision(4)}, "0.1235", "0.0198"},
		{[]Option{WithPaymentTaxesPrecision(4), WithTaxesTruncated()}, "0.1234", "0.0197"},
	} {
		r, e := Calculate(p, tt.opts...)
		if e != nil {
			t.Fatal(e)
		}
		tax := r.Pagos[0].Impuestos[0]
		if tax.Base != tt.base || tax.Importe != tt.amount {
			t.Fatalf("%+v", tax)
		}
	}
}

func TestWriterExentoAndMissingMonto(t *testing.T) {
	p := NewPagos(nil)
	payment := p.AddPago(Attrs{"MonedaP": "MXN", "TipoCambioP": "1"})
	payment.AddDoctoRelacionado(Attrs{"EquivalenciaDR": "1", "ImpPagado": "10"}).ImpuestosDR().TrasladosDR().AddTrasladoDR(Attrs{"BaseDR": "10", "ImpuestoDR": "002", "TipoFactorDR": "Exento"})
	r, e := p.WriteTotales(WithPaymentTaxesPrecision(2))
	if e != nil {
		t.Fatal(e)
	}
	if r.Pagos[0].MontoMinimo != "10.00" || payment.Attr("Monto") != "10.00" {
		t.Fatal("monto mínimo incorrecto")
	}
	tax := payment.SearchNode("pago20:ImpuestosP", "pago20:TrasladosP", "pago20:TrasladoP")
	if tax.HasAttr("ImporteP") || tax.HasAttr("TasaOCuotaP") {
		t.Fatal("exento contiene importe o tasa")
	}
	if tax.String() != `<pago20:TrasladoP BaseP="10.00" ImpuestoP="002" TipoFactorP="Exento"/>` {
		t.Fatal(tax.String())
	}
}

func TestInvalidDecimals(t *testing.T) {
	for _, attrs := range []Attrs{{"EquivalenciaDR": "0"}, {"EquivalenciaDR": "abc"}, {"EquivalenciaDR": "1", "ImpPagado": "xyz"}} {
		p := NewPagos(nil)
		p.AddPago(Attrs{"TipoCambioP": "1"}).AddDoctoRelacionado(attrs)
		if _, e := Calculate(p); e == nil {
			t.Fatalf("se esperaba error para %v", attrs)
		}
	}
}

func TestDecimalPrecision(t *testing.T) {
	for _, tt := range []struct {
		s     string
		p     int
		round bool
		want  string
	}{
		{"-1.005", 2, true, "-1.01"}, {"1.005", 2, true, "1.01"},
		{"-0.001", 2, true, "0.00"}, {"9999999999999999999999.999999", 2, true, "10000000000000000000000.00"},
		{"0.123456789", 4, false, "0.1234"},
	} {
		if got := fixed(mustNumber(tt.s), tt.p, tt.round); got != tt.want {
			t.Fatalf("%s: %s != %s", tt.s, got, tt.want)
		}
	}
}

func TestBuildCreatedCFDI40Pago20(t *testing.T) {
	fixture, e := nodes.ParseXMLFile("../../testdata/cfdi/created-cfdi40-pago20-valid.xml")
	if e != nil {
		t.Fatal(e)
	}
	expected := fixture.SearchNode("cfdi:Complemento", "pago20:Pagos")
	p := NewPagos(nil)
	p.AddTotales(expected.FirstChild("pago20:Totales").AttrsMap())
	for _, source := range expected.ChildrenNamed("pago20:Pago") {
		payment := p.AddPago(source.AttrsMap())
		for _, doc := range source.ChildrenNamed("pago20:DoctoRelacionado") {
			related := payment.AddDoctoRelacionado(doc.AttrsMap())
			for _, tax := range doc.SearchNodes("pago20:ImpuestosDR", "pago20:TrasladosDR", "pago20:TrasladoDR") {
				related.AddImpuestosDR().AddTrasladosDR().AddTrasladoDR(tax.AttrsMap())
			}
		}
		for _, tax := range source.SearchNodes("pago20:ImpuestosP", "pago20:TrasladosP", "pago20:TrasladoP") {
			payment.AddImpuestosP().AddTrasladosP().AddTrasladoP(tax.AttrsMap())
		}
	}
	// Normalizar únicamente el orden y las declaraciones propias del complemento.
	expected.SetAttr("xmlns:pago20", Namespace).SetAttr("xsi:schemaLocation", Namespace+" "+XsdLocation)
	var normalize func(*nodes.Node, *nodes.Node)
	normalize = func(want, got *nodes.Node) {
		want.SetAttributesOrder(got.AttributesOrder()...)
		if want.Len() != got.Len() {
			t.Fatalf("cantidad de hijos en %s", want.Name())
		}
		for i, child := range want.Children() {
			normalize(child, got.Child(i))
		}
	}
	normalize(expected, p.Node)
	if p.String() != expected.String() {
		t.Fatalf("estructura distinta:\n%s\n%s", p, expected)
	}
}

func TestCurrencyPrecisionAndNoMutation(t *testing.T) {
	p := NewPagos(nil)
	p.AddPago(Attrs{"MonedaP": "JPY", "TipoCambioP": "0.1"}).AddDoctoRelacionado(Attrs{"EquivalenciaDR": "3", "ImpPagado": "10"})
	before := p.String()
	r, e := Calculate(p, WithCurrencies(map[string]int{"JPY": 0}))
	if e != nil {
		t.Fatal(e)
	}
	if p.String() != before {
		t.Fatal("el cálculo modificó el nodo")
	}
	if r.Pagos[0].MontoMinimo != "3" || r.Totales["MontoTotalPagos"] != "0.30" {
		t.Fatal(r)
	}
	if e := Write(p, &Result{}); e == nil {
		t.Fatal("se esperaba error de cantidad de pagos")
	}
	if p.String() != before {
		t.Fatal("escritura inválida modificó el nodo")
	}
}
