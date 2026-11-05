package estadosat

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultServiceURL es la URL del servicio de consulta del SAT.
const DefaultServiceURL = "https://consultaqr.facturaelectronica.sat.gob.mx/ConsultaCFDIService.svc"

const soapAction = "http://tempuri.org/IConsultaCFDIService/Consulta"

// Config configura el servicio web.
type Config struct {
	// ServiceURL es la URL del servicio, si está vacía se usa DefaultServiceURL.
	ServiceURL string
	// Timeout es el tiempo máximo de espera, si es cero se usan 10 segundos.
	// Se ignora si se establece HTTPClient.
	Timeout time.Duration
	// HTTPClient permite usar un cliente propio (proxy, TLS, transporte, etc.).
	HTTPClient *http.Client
}

// WebService consume el servicio de consulta de estado de CFDI del SAT.
type WebService struct {
	serviceURL string
	client     *http.Client
}

// NewWebService crea un consumidor del servicio, config puede ser nil.
func NewWebService(config *Config) *WebService {
	if config == nil {
		config = &Config{}
	}
	ws := &WebService{serviceURL: config.ServiceURL, client: config.HTTPClient}
	if ws.serviceURL == "" {
		ws.serviceURL = DefaultServiceURL
	}
	if ws.client == nil {
		timeout := config.Timeout
		if timeout == 0 {
			timeout = 10 * time.Second
		}
		ws.client = &http.Client{Timeout: timeout}
	}
	return ws
}

// ServiceURL devuelve la URL del servicio.
func (ws *WebService) ServiceURL() string { return ws.serviceURL }

// Request consulta el estado del CFDI con los parámetros dados.
func (ws *WebService) Request(ctx context.Context, params *RequestParameters) (*StatusResponse, error) {
	if params == nil {
		return nil, errors.New("estadosat: no se recibieron parámetros")
	}
	return ws.RequestExpression(ctx, params.Expression())
}

// RequestXML consulta el estado del CFDI a partir de su contenido XML.
func (ws *WebService) RequestXML(ctx context.Context, cfdiXML []byte) (*StatusResponse, error) {
	params, err := RequestParametersFromXML(cfdiXML)
	if err != nil {
		return nil, err
	}
	return ws.Request(ctx, params)
}

// RequestExpression consulta el estado usando una expresión impresa ya construida.
func (ws *WebService) RequestExpression(ctx context.Context, expression string) (*StatusResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var body bytes.Buffer
	body.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	body.WriteString(`<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:tem="http://tempuri.org/">`)
	body.WriteString(`<soapenv:Header/><soapenv:Body><tem:Consulta><tem:expresionImpresa>`)
	if err := xml.EscapeText(&body, []byte(expression)); err != nil {
		return nil, fmt.Errorf("estadosat: %w", err)
	}
	body.WriteString(`</tem:expresionImpresa></tem:Consulta></soapenv:Body></soapenv:Envelope>`)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ws.serviceURL, &body)
	if err != nil {
		return nil, fmt.Errorf("estadosat: %w", err)
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", `"`+soapAction+`"`)
	req.Header.Set("Accept", "text/xml")

	resp, err := ws.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("estadosat: error al consultar el servicio: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("estadosat: error al leer la respuesta: %w", err)
	}
	status, parseErr := ParseResponse(raw)
	if parseErr != nil {
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("estadosat: el servicio respondió con estado HTTP %d: %w", resp.StatusCode, parseErr)
		}
		return nil, parseErr
	}
	return status, nil
}

// ParseResponse interpreta el sobre SOAP de respuesta del servicio.
func ParseResponse(raw []byte) (*StatusResponse, error) {
	dec := xml.NewDecoder(bytes.NewReader(raw))
	values := map[string]string{}
	var stack []string
	var text strings.Builder
	inResult := false
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("estadosat: la respuesta del servicio no es XML válido: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			stack = append(stack, t.Name.Local)
			text.Reset()
			if t.Name.Local == "ConsultaResult" {
				inResult = true
			}
		case xml.CharData:
			text.Write(t)
		case xml.EndElement:
			name := t.Name.Local
			if inResult && name != "ConsultaResult" {
				values[name] = strings.TrimSpace(text.String())
			}
			if name == "faultstring" {
				values["faultstring"] = strings.TrimSpace(text.String())
			}
			if name == "ConsultaResult" {
				inResult = false
			}
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			text.Reset()
		}
	}
	if fault, ok := values["faultstring"]; ok {
		return nil, fmt.Errorf("estadosat: el servicio devolvió un error SOAP: %s", fault)
	}
	code, okCode := values["CodigoEstatus"]
	estado, okEstado := values["Estado"]
	if !okCode {
		return nil, errors.New("estadosat: la respuesta no contiene ConsultaResult/CodigoEstatus")
	}
	if !okEstado {
		return nil, errors.New("estadosat: la respuesta no contiene ConsultaResult/Estado")
	}
	return &StatusResponse{
		CodigoEstatus:      code,
		Estado:             estado,
		EsCancelable:       values["EsCancelable"],
		EstatusCancelacion: values["EstatusCancelacion"],
		ValidacionEFOS:     values["ValidacionEFOS"],
	}, nil
}

// StatusResponse es la respuesta del servicio de consulta.
type StatusResponse struct {
	// CodigoEstatus es el estado de la consulta, por ejemplo
	// "S - Comprobante obtenido satisfactoriamente." o
	// "N - 601: La expresión impresa proporcionada no es válida.".
	CodigoEstatus string
	// Estado es el estado del CFDI: "Vigente", "Cancelado" o "No Encontrado".
	Estado string
	// EsCancelable: "No cancelable", "Cancelable sin aceptación" o "Cancelable con aceptación".
	EsCancelable string
	// EstatusCancelacion: "" (sin solicitud), "Cancelado sin aceptación", "En proceso",
	// "Plazo vencido", "Cancelado con aceptación" o "Solicitud rechazada".
	EstatusCancelacion string
	// ValidacionEFOS: "200" el emisor no está en el listado de EFOS, "100" sí está.
	ValidacionEFOS string
}

// ResponseWasOk informa si la consulta fue satisfactoria (CodigoEstatus inicia con "S - ").
func (r *StatusResponse) ResponseWasOk() bool { return strings.HasPrefix(r.CodigoEstatus, "S - ") }

// IsVigente informa si el CFDI está vigente.
func (r *StatusResponse) IsVigente() bool { return r.Estado == "Vigente" }

// IsCancelado informa si el CFDI está cancelado.
func (r *StatusResponse) IsCancelado() bool { return r.Estado == "Cancelado" }

// IsNoEncontrado informa si el SAT no encontró el CFDI.
func (r *StatusResponse) IsNoEncontrado() bool { return r.Estado == "No Encontrado" }

// IsCancelable informa si el CFDI se puede cancelar (con o sin aceptación).
func (r *StatusResponse) IsCancelable() bool {
	return strings.HasPrefix(r.EsCancelable, "Cancelable")
}

// RequiresAcceptance informa si la cancelación requiere aceptación del receptor.
func (r *StatusResponse) RequiresAcceptance() bool {
	return r.EsCancelable == "Cancelable con aceptación"
}

// IsEfosListed informa si el emisor se encuentra en el listado de EFOS.
func (r *StatusResponse) IsEfosListed() bool { return r.ValidacionEFOS == "100" }
