package hrns

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type jsonDupParser struct {
	src  string
	pos  int
	dups []string
}

func duplicateJSONKeys(src string) ([]string, error) {
	p := &jsonDupParser{src: src}
	if err := p.parseValue(nil); err != nil {
		return nil, err
	}
	p.skipWS()
	if p.pos != len(p.src) {
		return nil, fmt.Errorf("trailing content at byte %d", p.pos)
	}
	return p.dups, nil
}

func (p *jsonDupParser) parseValue(path []string) error {
	p.skipWS()
	if p.pos >= len(p.src) {
		return fmt.Errorf("unexpected end")
	}
	switch p.src[p.pos] {
	case '{':
		return p.parseObject(path)
	case '[':
		return p.parseArray(path)
	case '"':
		_, err := p.parseString()
		return err
	case 't':
		return p.consume("true")
	case 'f':
		return p.consume("false")
	case 'n':
		return p.consume("null")
	default:
		return p.parseNumber()
	}
}

func (p *jsonDupParser) parseObject(path []string) error {
	p.pos++
	seen := map[string]struct{}{}
	p.skipWS()
	if p.peek('}') {
		p.pos++
		return nil
	}
	for {
		p.skipWS()
		key, err := p.parseString()
		if err != nil {
			return err
		}
		if _, ok := seen[key]; ok {
			p.dups = append(p.dups, strings.Join(append(path, key), "."))
		}
		seen[key] = struct{}{}
		p.skipWS()
		if !p.peek(':') {
			return fmt.Errorf("expected ':' at byte %d", p.pos)
		}
		p.pos++
		if err := p.parseValue(append(path, key)); err != nil {
			return err
		}
		p.skipWS()
		if p.peek('}') {
			p.pos++
			return nil
		}
		if !p.peek(',') {
			return fmt.Errorf("expected ',' or '}' at byte %d", p.pos)
		}
		p.pos++
	}
}

func (p *jsonDupParser) parseArray(path []string) error {
	p.pos++
	p.skipWS()
	if p.peek(']') {
		p.pos++
		return nil
	}
	for idx := 0; ; idx++ {
		if err := p.parseValue(append(path, strconv.Itoa(idx))); err != nil {
			return err
		}
		p.skipWS()
		if p.peek(']') {
			p.pos++
			return nil
		}
		if !p.peek(',') {
			return fmt.Errorf("expected ',' or ']' at byte %d", p.pos)
		}
		p.pos++
	}
}

func (p *jsonDupParser) parseString() (string, error) {
	if !p.peek('"') {
		return "", fmt.Errorf("expected string at byte %d", p.pos)
	}
	p.pos++
	var b strings.Builder
	for p.pos < len(p.src) {
		ch := p.src[p.pos]
		p.pos++
		if ch == '"' {
			return b.String(), nil
		}
		if ch != '\\' {
			b.WriteByte(ch)
			continue
		}
		if p.pos >= len(p.src) {
			return "", fmt.Errorf("invalid escape at byte %d", p.pos)
		}
		esc := p.src[p.pos]
		p.pos++
		switch esc {
		case '"', '\\', '/':
			b.WriteByte(esc)
		case 'b':
			b.WriteByte('\b')
		case 'f':
			b.WriteByte('\f')
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		case 't':
			b.WriteByte('\t')
		case 'u':
			if p.pos+4 > len(p.src) {
				return "", fmt.Errorf("short unicode escape at byte %d", p.pos)
			}
			b.WriteString(`\u` + p.src[p.pos:p.pos+4])
			p.pos += 4
		default:
			return "", fmt.Errorf("invalid escape at byte %d", p.pos)
		}
	}
	return "", fmt.Errorf("unterminated string")
}

func (p *jsonDupParser) parseNumber() error {
	start := p.pos
	for p.pos < len(p.src) && strings.ContainsRune("-+0123456789.eE", rune(p.src[p.pos])) {
		p.pos++
	}
	if start == p.pos {
		return fmt.Errorf("expected value at byte %d", p.pos)
	}
	return nil
}

func (p *jsonDupParser) consume(value string) error {
	if !strings.HasPrefix(p.src[p.pos:], value) {
		return fmt.Errorf("expected %s at byte %d", value, p.pos)
	}
	p.pos += len(value)
	return nil
}

func (p *jsonDupParser) skipWS() {
	for p.pos < len(p.src) && unicode.IsSpace(rune(p.src[p.pos])) {
		p.pos++
	}
}

func (p *jsonDupParser) peek(ch byte) bool {
	return p.pos < len(p.src) && p.src[p.pos] == ch
}
