// Package calc is a self-contained arithmetic engine with no UI or framework
// dependencies. The frontend builds an expression string and asks the engine
// to evaluate it as a whole (only when the user presses "=" / Enter).
//
// Arithmetic uses math/big rationals, so results are exact — integers beyond
// 2^53 and sums like 0.1 + 0.2 come out right, unlike float64.
package calc

import (
	"fmt"
	"math/big"
	"strings"
	"unicode"
)

// Evaluate parses and evaluates a full arithmetic expression, returning the
// formatted result. It supports + - * / and parentheses with the usual
// precedence, unary signs, a postfix percent, and both "." and "," as the
// decimal separator.
func Evaluate(expression string) (string, error) {
	parser := newParser(expression)
	result, err := parser.parse()
	if err != nil {
		return "", err
	}
	return formatResult(result), nil
}

type parser struct {
	input []rune
	pos   int
}

func newParser(input string) *parser {
	input = strings.ReplaceAll(input, ",", ".")
	// Accept the unicode multiplication/division glyphs the UI renders on its
	// buttons so the frontend can send exactly what the user sees.
	input = strings.ReplaceAll(input, "×", "*")
	input = strings.ReplaceAll(input, "÷", "/")
	input = strings.ReplaceAll(input, "−", "-")
	return &parser{input: []rune(input)}
}

func (p *parser) parse() (*big.Rat, error) {
	p.skipSpaces()
	if p.done() {
		return nil, fmt.Errorf("digite uma conta")
	}

	result, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	p.skipSpaces()
	if !p.done() {
		return nil, fmt.Errorf("operador inesperado")
	}

	return result, nil
}

func (p *parser) parseExpression() (*big.Rat, error) {
	left, err := p.parseTerm()
	if err != nil {
		return nil, err
	}

	for {
		p.skipSpaces()
		switch p.peek() {
		case '+':
			p.pos++
			right, err := p.parseTerm()
			if err != nil {
				return nil, err
			}
			left = new(big.Rat).Add(left, right)
		case '-':
			p.pos++
			right, err := p.parseTerm()
			if err != nil {
				return nil, err
			}
			left = new(big.Rat).Sub(left, right)
		default:
			return left, nil
		}
	}
}

func (p *parser) parseTerm() (*big.Rat, error) {
	left, err := p.parseFactor()
	if err != nil {
		return nil, err
	}

	for {
		p.skipSpaces()
		switch p.peek() {
		case '*':
			p.pos++
			right, err := p.parseFactor()
			if err != nil {
				return nil, err
			}
			left = new(big.Rat).Mul(left, right)
		case '/':
			p.pos++
			right, err := p.parseFactor()
			if err != nil {
				return nil, err
			}
			if right.Sign() == 0 {
				return nil, fmt.Errorf("divisao por zero")
			}
			left = new(big.Rat).Quo(left, right)
		default:
			return left, nil
		}
	}
}

func (p *parser) parseFactor() (*big.Rat, error) {
	p.skipSpaces()

	switch p.peek() {
	case '+':
		p.pos++
		return p.parseFactor()
	case '-':
		p.pos++
		value, err := p.parseFactor()
		if err != nil {
			return nil, err
		}
		return new(big.Rat).Neg(value), nil
	default:
		value, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		// Postfix percent: "50%" is 0.5, "200 * 50%" is 100. Applied at the
		// factor level so it binds tighter than + - * /.
		for {
			p.skipSpaces()
			if p.peek() != '%' {
				break
			}
			p.pos++
			value = new(big.Rat).Quo(value, big.NewRat(100, 1))
		}
		return value, nil
	}
}

func (p *parser) parsePrimary() (*big.Rat, error) {
	p.skipSpaces()
	if p.peek() == '(' {
		p.pos++
		value, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		p.skipSpaces()
		if p.peek() != ')' {
			return nil, fmt.Errorf("parenteses nao fechado")
		}
		p.pos++
		return value, nil
	}
	return p.parseNumber()
}

func (p *parser) parseNumber() (*big.Rat, error) {
	start := p.pos
	dotCount := 0

	for !p.done() {
		current := p.peek()
		if current == '.' {
			dotCount++
			if dotCount > 1 {
				return nil, fmt.Errorf("numero invalido")
			}
			p.pos++
			continue
		}
		if !unicode.IsDigit(current) {
			break
		}
		p.pos++
	}

	if start == p.pos {
		return nil, fmt.Errorf("numero esperado")
	}

	value, ok := new(big.Rat).SetString(string(p.input[start:p.pos]))
	if !ok {
		return nil, fmt.Errorf("numero invalido")
	}

	return value, nil
}

func (p *parser) skipSpaces() {
	for !p.done() && unicode.IsSpace(p.peek()) {
		p.pos++
	}
}

func (p *parser) peek() rune {
	if p.done() {
		return 0
	}
	return p.input[p.pos]
}

func (p *parser) done() bool {
	return p.pos >= len(p.input)
}

// formatResult renders exact integers in full (arbitrary precision) and other
// values with up to 10 decimal places, trailing zeros trimmed.
func formatResult(value *big.Rat) string {
	if value.IsInt() {
		return value.Num().String()
	}

	result := value.FloatString(10)
	result = strings.TrimRight(result, "0")
	result = strings.TrimRight(result, ".")
	if result == "" || result == "-0" {
		return "0"
	}

	return result
}
