package calc

import "testing"

func TestEvaluate(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"soma", "2 + 3", "5"},
		{"precedencia", "2 + 3 * 4", "14"},
		{"parenteses", "(2 + 3) * 4", "20"},
		{"unario", "-5 + 2", "-3"},
		{"unario duplo", "--3", "3"},
		{"decimal ponto", "1.5 + 1.5", "3"},
		{"decimal virgula", "1,5 + 1,5", "3"},
		{"glifos da ui", "6 ÷ 2 × 3", "9"},
		{"menos unicode", "10 − 4", "6"},
		{"divisao exata", "10 / 4", "2.5"},
		{"zero limpo", "0.1 + 0.2", "0.3"},
		{"percent simples", "50%", "0.5"},
		{"percent multiplicacao", "200 * 50%", "100"},
		{"percent com glifo", "200 × 15%", "30"},
		{"percent negativo", "-10%", "-0.1"},
		{"inteiro grande acima de 2^53", "9007199254740992 + 1", "9007199254740993"},
		{"inteiro gigante", "100000000000000000000 * 2", "200000000000000000000"},
		{"soma exata sem ruido de float", "0.1 + 0.2 + 0.3", "0.6"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Evaluate(tc.in)
			if err != nil {
				t.Fatalf("Evaluate(%q) erro inesperado: %v", tc.in, err)
			}
			if got != tc.want {
				t.Errorf("Evaluate(%q) = %q, quer %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestEvaluateErrors(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"vazio", "   "},
		{"divisao por zero", "1 / 0"},
		{"parentese aberto", "(2 + 3"},
		{"operador solto", "2 +"},
		{"lixo no fim", "2 3"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Evaluate(tc.in); err == nil {
				t.Errorf("Evaluate(%q) esperava erro, obteve nil", tc.in)
			}
		})
	}
}
