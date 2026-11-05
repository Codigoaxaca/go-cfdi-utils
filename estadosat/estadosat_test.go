package estadosat

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestExpression40(t *testing.T) {
	p, err := NewRequestParameters("4.0", "EKU9003173C9", "COSC8001137NA", "1,234.5678", "CEE4BE01-ADFA-4DEB-8421-ADD60F0BEDAC", "...abcfe/1234==")
	if err != nil {
		t.Fatal(err)
	}
	want := "https://verificacfdi.facturaelectronica.sat.gob.mx/default.aspx?id=CEE4BE01-ADFA-4DEB-8421-ADD60F0BEDAC&re=EKU9003173C9&rr=COSC8001137NA&tt=1234.5678&fe=e/1234=="
	if got := p.Expression(); got != want {
		t.Fatalf("got %s", got)
	}
}

func TestExpressionTotals(t *testing.T) {
	cases := map[string]string{"0.99": "tt=0.99&", "1": "tt=1.0&", "0": "tt=0.0&", "1000.000001": "tt=1000.000001&", "12.1234567": "tt=12.123457&"}
	for total, want := range cases {
		p, _ := NewRequestParameters("3.3", "A&B010101AAA", "XAXX010101000", total, "uuid", "")
		if !strings.Contains(p.Expression(), want) {
			t.Errorf("total %s: %s", total, p.Expression())
		}
		if !strings.Contains(p.Expression(), "re=A&amp;B010101AAA") {
			t.Errorf("rfc not escaped: %s", p.Expression())
		}
	}
}

func TestExpression32(t *testing.T) {
	p, _ := NewRequestParameters("3.2", "EKU9003173C9", "COSC8001137NA", "1234.5678", "UUID", "")
	want := "?re=EKU9003173C9&rr=COSC8001137NA&tt=0000001234.567800&id=UUID"
	if got := p.Expression(); got != want {
		t.Fatalf("got %s", got)
	}
}

func TestInvalidVersion(t *testing.T) {
	if _, err := NewRequestParameters("1.0", "", "", "", "", ""); err == nil {
		t.Fatal("expected error")
	}
}

func TestFromXML(t *testing.T) {
	for _, file := range []string{"cfdi40-real.xml", "cfdi33-real.xml"} {
		data, err := os.ReadFile("../testdata/cfdi/" + file)
		if err != nil {
			t.Fatal(err)
		}
		p, err := RequestParametersFromXML(data)
		if err != nil {
			t.Fatal(err)
		}
		if p.UUID == "" || p.RfcEmisor == "" || p.RfcReceptor == "" || p.Total == "" || len(p.Sello) < 8 {
			t.Fatalf("%s: incomplete params %+v", file, p)
		}
	}
}

const sampleResponse = `<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/"><s:Body><ConsultaResponse xmlns="http://tempuri.org/"><ConsultaResult xmlns:a="http://schemas.datacontract.org/2004/07/Sat.Cfdi.Negocio.ConsultaCfdi.Servicio" xmlns:i="http://www.w3.org/2001/XMLSchema-instance"><a:CodigoEstatus>S - Comprobante obtenido satisfactoriamente.</a:CodigoEstatus><a:EsCancelable>Cancelable con aceptación</a:EsCancelable><a:Estado>Vigente</a:Estado><a:EstatusCancelacion/><a:ValidacionEFOS>200</a:ValidacionEFOS></ConsultaResult></ConsultaResponse></s:Body></s:Envelope>`

func TestWebServiceWithFakeServer(t *testing.T) {
	var gotBody, gotAction string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		gotAction = r.Header.Get("SOAPAction")
		w.Header().Set("Content-Type", "text/xml")
		_, _ = io.WriteString(w, sampleResponse)
	}))
	defer srv.Close()

	ws := NewWebService(&Config{ServiceURL: srv.URL})
	p, _ := NewRequestParameters("4.0", "EKU9003173C9", "COSC8001137NA", "100", "UUID", "12345678")
	resp, err := ws.Request(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.ResponseWasOk() || !resp.IsVigente() || !resp.IsCancelable() || !resp.RequiresAcceptance() || resp.IsEfosListed() || resp.EstatusCancelacion != "" {
		t.Fatalf("unexpected response %+v", resp)
	}
	if gotAction != `"http://tempuri.org/IConsultaCFDIService/Consulta"` {
		t.Fatalf("bad SOAPAction %s", gotAction)
	}
	if !strings.Contains(gotBody, "re=EKU9003173C9&amp;rr=COSC8001137NA") {
		t.Fatalf("bad body %s", gotBody)
	}
}

func TestParseFault(t *testing.T) {
	fault := `<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/"><s:Body><s:Fault><faultcode>a:ActionNotSupported</faultcode><faultstring xml:lang="es-MX">Error</faultstring></s:Fault></s:Body></s:Envelope>`
	if _, err := ParseResponse([]byte(fault)); err == nil || !strings.Contains(err.Error(), "Error") {
		t.Fatalf("expected fault error, got %v", err)
	}
	if _, err := ParseResponse([]byte("<html/>")); err == nil {
		t.Fatal("expected error")
	}
}

// TestLiveService consulta el servicio real del SAT, se ejecuta solo con
// CFDI_LIVE_TESTS=1.
func TestLiveService(t *testing.T) {
	if os.Getenv("CFDI_LIVE_TESTS") == "" {
		t.Skip("define CFDI_LIVE_TESTS=1 para consultar el servicio del SAT")
	}
	data, _ := os.ReadFile("../testdata/cfdi/cfdi40-real.xml")
	resp, err := NewWebService(nil).RequestXML(context.Background(), data)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", resp)
	if !resp.ResponseWasOk() {
		t.Fatalf("unexpected response %+v", resp)
	}
}
