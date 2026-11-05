package pagos20

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/codigoaxaca/go-cfdi-utils/nodes"
)

// Result contiene los totales SAT y los datos calculados de cada pago.
type Result struct {
	// Totales contiene únicamente los atributos aplicables, expresados en MXN.
	Totales Attrs
	// Pagos conserva el orden de los pagos del complemento.
	Pagos []PaymentResult
}

// PaymentResult contiene los importes y los impuestos de un pago.
type PaymentResult struct {
	// Monto es el importe declarado o el mínimo calculado.
	Monto string
	// MontoMinimo es la suma convertida y truncada a los decimales de la moneda.
	MontoMinimo string
	// TipoCambioP es el factor de conversión a MXN.
	TipoCambioP string
	// Impuestos contiene los impuestos agrupados.
	Impuestos []TaxResult
}

// TaxResult contiene un impuesto agregado sin usar punto flotante.
type TaxResult struct {
	// Tipo distingue Traslado de Retencion.
	Tipo string
	// Impuesto es la clave SAT.
	Impuesto string
	// TipoFactor identifica Tasa, Cuota o Exento.
	TipoFactor string
	// TasaOCuota conserva la representación de la tasa original.
	TasaOCuota string
	// Base contiene la base decimal.
	Base string
	// Importe contiene el impuesto decimal.
	Importe string
}

// Option configura el cálculo de pagos.
type Option func(*Calculator)

// Calculator calcula importes con aritmética decimal de 24 posiciones.
type Calculator struct {
	precision  int
	truncated  bool
	currencies map[string]int
}

// NewCalculator crea una calculadora con seis decimales y redondeo.
func NewCalculator(opts ...Option) *Calculator {
	c := &Calculator{precision: 6, currencies: map[string]int{}}
	for _, o := range opts {
		o(c)
	}
	return c
}

// WithPaymentTaxesPrecision limita los decimales de impuestos entre cero y seis.
func WithPaymentTaxesPrecision(p int) Option {
	return func(c *Calculator) { c.precision = min(6, max(0, p)) }
}

// WithTaxesTruncated activa el truncado de impuestos en lugar del redondeo.
func WithTaxesTruncated() Option { return func(c *Calculator) { c.truncated = true } }

// WithCurrencies define los decimales de monedas; las no registradas usan dos.
func WithCurrencies(values map[string]int) Option {
	return func(c *Calculator) {
		c.currencies = map[string]int{}
		for k, v := range values {
			c.currencies[k] = min(4, max(0, v))
		}
	}
}

// Calculate calcula totales e impuestos sin modificar el complemento.
func Calculate(pagos nodes.Element, opts ...Option) (*Result, error) {
	return NewCalculator(opts...).Calculate(pagos)
}

func number(s string) (*big.Rat, error) {
	if s == "" {
		return new(big.Rat), nil
	}
	if strings.ContainsAny(s, "/eE") {
		return nil, fmt.Errorf("decimal inválido: %q", s)
	}
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		return nil, fmt.Errorf("decimal inválido: %q", s)
	}
	return r, nil
}
func mustNumber(s string) *big.Rat { r, _ := number(s); return r }
func fixed(r *big.Rat, p int, round bool) string {
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(p)), nil)
	n := new(big.Int).Mul(r.Num(), scale)
	q, rem := new(big.Int), new(big.Int)
	q.QuoRem(n, r.Denom(), rem)
	if round && new(big.Int).Mul(new(big.Int).Abs(rem), big.NewInt(2)).Cmp(r.Denom()) >= 0 {
		if r.Sign() < 0 {
			q.Sub(q, big.NewInt(1))
		} else {
			q.Add(q, big.NewInt(1))
		}
	}
	neg := q.Sign() < 0
	s := new(big.Int).Abs(q).String()
	if p > 0 {
		for len(s) <= p {
			s = "0" + s
		}
		s = s[:len(s)-p] + "." + s[len(s)-p:]
	}
	if neg {
		s = "-" + s
	}
	return s
}
func op(a, b string, div bool) (string, error) {
	x, e := number(a)
	if e != nil {
		return "", e
	}
	y, e := number(b)
	if e != nil {
		return "", e
	}
	if div {
		if y.Sign() == 0 {
			return "", fmt.Errorf("EquivalenciaDR debe ser distinta de cero")
		}
		x.Quo(x, y)
	} else {
		x.Mul(x, y)
	}
	return fixed(x, 24, false), nil
}
func add(a, b string) string { return fixed(new(big.Rat).Add(mustNumber(a), mustNumber(b)), 24, false) }
func taxKey(t TaxResult) string {
	if t.Tipo == "Retencion" {
		return t.Tipo + "|" + t.Impuesto
	}
	return strings.Join([]string{t.Tipo, t.Impuesto, t.TipoFactor, t.TasaOCuota}, "|")
}
func aggregate(taxes []TaxResult, t TaxResult) []TaxResult {
	for i, v := range taxes {
		if taxKey(v) == taxKey(t) {
			taxes[i].Base = add(v.Base, t.Base)
			taxes[i].Importe = add(v.Importe, t.Importe)
			return taxes
		}
	}
	return append(taxes, t)
}

// Calculate obtiene los pagos y convierte sus impuestos a la moneda del pago y a MXN.
func (c *Calculator) Calculate(pagos nodes.Element) (*Result, error) {
	if pagos == nil || pagos.AsNode() == nil {
		return nil, fmt.Errorf("se requiere el nodo Pagos")
	}
	result := &Result{Totales: Attrs{}}
	total := "0"
	var taxes []TaxResult
	for pi, p := range pagos.AsNode().ChildrenNamed("pago20:Pago") {
		pr := PaymentResult{TipoCambioP: p.Attr("TipoCambioP")}
		sum := "0"
		if _, e := number(pr.TipoCambioP); e != nil {
			return nil, e
		}
		for _, d := range p.ChildrenNamed("pago20:DoctoRelacionado") {
			eq := d.Attr("EquivalenciaDR")
			v, e := op(d.Attr("ImpPagado"), eq, true)
			if e != nil {
				return nil, fmt.Errorf("pago %d: %w", pi, e)
			}
			sum = add(sum, v)
			// PHP procesa primero traslados y luego retenciones.
			for _, kind := range []string{"Traslado", "Retencion"} {
				group := "TrasladosDR"
				if kind == "Retencion" {
					group = "RetencionesDR"
				}
				for _, n := range d.SearchNodes("pago20:ImpuestosDR", "pago20:"+group, "pago20:"+kind+"DR") {
					t := TaxResult{Tipo: kind, Impuesto: n.Attr("ImpuestoDR"), Base: "0", Importe: "0"}
					if kind == "Traslado" {
						t.TipoFactor = n.Attr("TipoFactorDR")
						t.TasaOCuota = n.Attr("TasaOCuotaDR")
						t.Base, e = op(n.Attr("BaseDR"), eq, true)
						if e != nil {
							return nil, e
						}
					}
					t.Importe, e = op(n.Attr("ImporteDR"), eq, true)
					if e != nil {
						return nil, e
					}
					pr.Impuestos = aggregate(pr.Impuestos, t)
				}
			}
		}
		precision, ok := c.currencies[p.Attr("MonedaP")]
		if !ok {
			precision = 2
		}
		pr.MontoMinimo = fixed(mustNumber(sum), precision, false)
		pr.Monto = pr.MontoMinimo
		if p.HasAttr("Monto") {
			pr.Monto = p.Attr("Monto")
		}
		for i, t := range pr.Impuestos {
			t.Base = fixed(mustNumber(t.Base), c.precision, !c.truncated)
			t.Importe = fixed(mustNumber(t.Importe), c.precision, !c.truncated)
			pr.Impuestos[i] = t
			var e error
			t.Base, e = op(t.Base, pr.TipoCambioP, false)
			if e != nil {
				return nil, e
			}
			t.Importe, e = op(t.Importe, pr.TipoCambioP, false)
			if e != nil {
				return nil, e
			}
			taxes = aggregate(taxes, t)
		}
		converted, e := op(pr.Monto, pr.TipoCambioP, false)
		if e != nil {
			return nil, e
		}
		total = add(total, converted)
		result.Pagos = append(result.Pagos, pr)
	}
	result.Totales["MontoTotalPagos"] = fixed(mustNumber(total), 2, true)
	for _, t := range taxes {
		base, amount := fixed(mustNumber(t.Base), 2, true), fixed(mustNumber(t.Importe), 2, true)
		if t.Tipo == "Retencion" {
			name := map[string]string{"001": "ISR", "002": "IVA", "003": "IEPS"}[t.Impuesto]
			if name != "" {
				result.Totales["TotalRetenciones"+name] = amount
			}
			continue
		}
		if t.Impuesto != "002" {
			continue
		}
		if t.TipoFactor == "Exento" && t.TasaOCuota == "" {
			result.Totales["TotalTrasladosBaseIVAExento"] = base
		}
		if t.TipoFactor != "Tasa" {
			continue
		}
		rate, ok := map[string]string{"0.160000": "16", "0.080000": "8", "0.000000": "0"}[t.TasaOCuota]
		if ok {
			result.Totales["TotalTrasladosBaseIVA"+rate] = base
			result.Totales["TotalTrasladosImpuestoIVA"+rate] = amount
		}
	}
	return result, nil
}

// Write sustituye Totales e ImpuestosP y completa Monto cuando no estaba declarado.
func Write(pagos *Pagos, result *Result) error {
	if pagos == nil || pagos.Node == nil || result == nil {
		return fmt.Errorf("se requieren pagos y resultado")
	}
	payments := pagos.ChildrenNamed("pago20:Pago")
	if len(payments) != len(result.Pagos) {
		return fmt.Errorf("la cantidad de pagos no coincide con el resultado")
	}
	for _, n := range pagos.ChildrenNamed("pago20:Totales") {
		pagos.RemoveChild(n)
	}
	pagos.AddTotales(result.Totales)
	for i, n := range payments {
		p := &Pago{n}
		data := result.Pagos[i]
		if !p.HasAttr("Monto") {
			p.SetAttr("Monto", data.MontoMinimo)
		}
		for _, old := range p.ChildrenNamed("pago20:ImpuestosP") {
			p.RemoveChild(old)
		}
		for _, t := range data.Impuestos {
			if t.Tipo == "Retencion" {
				p.ImpuestosP().RetencionesP().AddRetencionP(Attrs{"ImpuestoP": t.Impuesto, "ImporteP": t.Importe})
				continue
			}
			attrs := Attrs{"BaseP": t.Base, "ImpuestoP": t.Impuesto, "TipoFactorP": t.TipoFactor}
			if t.TipoFactor != "Exento" {
				attrs["TasaOCuotaP"] = t.TasaOCuota
				attrs["ImporteP"] = t.Importe
			}
			p.ImpuestosP().TrasladosP().AddTrasladoP(attrs)
		}
	}
	return nil
}

// WriteTotales calcula y escribe tanto los totales como los impuestos de cada pago.
func (p *Pagos) WriteTotales(opts ...Option) (*Result, error) {
	r, e := Calculate(p, opts...)
	if e != nil {
		return nil, e
	}
	return r, Write(p, r)
}
