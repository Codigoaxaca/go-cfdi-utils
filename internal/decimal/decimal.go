// Package decimal contiene utilerías de redondeo y formato numérico
// compatibles con el comportamiento de PHP (round y number_format), donde el
// redondeo es "mitad hacia arriba" (alejándose de cero) sobre la representación
// decimal más corta del número, de forma que Round(1.005, 2) == 1.01.
package decimal

import (
	"math"
	"math/big"
	"strconv"
	"strings"
)

// Round redondea x a prec decimales, mitad alejándose de cero.
func Round(x float64, prec int) float64 {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return x
	}
	r, ok := new(big.Rat).SetString(strconv.FormatFloat(x, 'g', -1, 64))
	if !ok {
		return x
	}
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(abs(prec))), nil)
	scaleRat := new(big.Rat).SetInt(scale)
	if prec >= 0 {
		r.Mul(r, scaleRat)
	} else {
		r.Quo(r, scaleRat)
	}
	num, den := new(big.Int).Abs(r.Num()), r.Denom()
	// q = floor((2*num + den) / (2*den))
	twoNum := new(big.Int).Mul(num, big.NewInt(2))
	twoNum.Add(twoNum, den)
	q := new(big.Int).Quo(twoNum, new(big.Int).Mul(den, big.NewInt(2)))
	if r.Sign() < 0 {
		q.Neg(q)
	}
	res := new(big.Rat).SetInt(q)
	if prec >= 0 {
		res.Quo(res, scaleRat)
	} else {
		res.Mul(res, scaleRat)
	}
	f, _ := res.Float64()
	if f == 0 {
		return 0 // evita -0
	}
	return f
}

// Format redondea x a prec decimales y lo expresa con exactamente prec
// decimales, sin separador de miles (como number_format($x, $prec, '.', ”)).
func Format(x float64, prec int) string {
	if prec < 0 {
		prec = 0
	}
	s := strconv.FormatFloat(Round(x, prec), 'f', prec, 64)
	if strings.Trim(s, "-0.") == "" {
		s = strings.TrimPrefix(s, "-")
	}
	return s
}

// Parse convierte una cadena a número, devuelve 0 si no es válida.
// Se eliminan espacios y separadores de miles (comas).
func Parse(s string) float64 {
	s = strings.ReplaceAll(strings.TrimSpace(s), ",", "")
	if s == "" {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

// Equals compara dos números con una tolerancia de medio dígito en la
// precisión dada.
func Equals(a, b float64, prec int) bool {
	return math.Abs(Round(a, prec)-Round(b, prec)) < math.Pow10(-prec)/2
}

func abs(i int) int {
	if i < 0 {
		return -i
	}
	return i
}
