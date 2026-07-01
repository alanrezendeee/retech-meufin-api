package health

import "testing"

func ptrF(v float64) *float64 { return &v }

func TestParseResultNumeric(t *testing.T) {
	cases := []struct {
		in   string
		want *float64
	}{
		{"1,23", ptrF(1.23)},
		{"1.23", ptrF(1.23)},
		{"<0,1", ptrF(0.1)},
		{">200", ptrF(200)},
		{"  ≥ 5,5 ", ptrF(5.5)},
		{"1.234,56", ptrF(1234.56)}, // separador de milhar pt-BR
		{"98", ptrF(98)},
		{"Não reagente", nil},
		{"Positivo", nil},
		{"", nil},
		{"---", nil},
	}
	for _, c := range cases {
		got := ParseResultNumeric(c.in)
		switch {
		case got == nil && c.want == nil:
			// ok
		case got == nil || c.want == nil:
			t.Errorf("ParseResultNumeric(%q) = %v, quer %v", c.in, deref(got), deref(c.want))
		case *got != *c.want:
			t.Errorf("ParseResultNumeric(%q) = %v, quer %v", c.in, *got, *c.want)
		}
	}
}

func deref(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}

func TestComputeInterpretation(t *testing.T) {
	cases := []struct {
		name       string
		value      *float64
		min, max   *float64
		wantResult *string
	}{
		{"abaixo do minimo -> low", ptrF(3), ptrF(5), ptrF(10), strp("low")},
		{"dentro da faixa -> normal", ptrF(7), ptrF(5), ptrF(10), strp("normal")},
		{"acima do maximo -> high", ptrF(12), ptrF(5), ptrF(10), strp("high")},
		{"sem referencia -> normal", ptrF(7), nil, nil, strp("normal")},
		{"apenas min, abaixo -> low", ptrF(2), ptrF(5), nil, strp("low")},
		{"apenas max, acima -> high", ptrF(20), nil, ptrF(10), strp("high")},
		{"sem valor -> nil", nil, ptrF(5), ptrF(10), nil},
	}
	for _, c := range cases {
		got := ComputeInterpretation(c.value, c.min, c.max)
		switch {
		case got == nil && c.wantResult == nil:
			// ok
		case got == nil || c.wantResult == nil:
			t.Errorf("%s: ComputeInterpretation = %v, quer %v", c.name, derefS(got), derefS(c.wantResult))
		case *got != *c.wantResult:
			t.Errorf("%s: ComputeInterpretation = %q, quer %q", c.name, *got, *c.wantResult)
		}
	}
}

func strp(s string) *string { return &s }

func derefS(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}
