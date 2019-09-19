package css

import (
	"bytes"
	"container/list"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"text/scanner"
)

type tokenType int

const (
	tokenFirstToken tokenType = iota - 1
	tokenBlockStart
	tokenBlockEnd
	tokenRuleName
	tokenValue
	tokenSelector
	tokenStyleSeparator
	tokenStatementEnd
)

var (
	rComments = regexp.MustCompile(`\/\*[^*]*\*+([^\/*][^*]*\*+)*\/`)
	rLicenses = regexp.MustCompile(`\/\*![^*]*\*+([^\/*][^*]*\*+)*\/`)
)

// Rule is a string type that represents a CSS rule.
type Rule string

type TokenEntry struct {
	value string
	pos   scanner.Position
}

type tokenizer struct {
	s *scanner.Scanner
}

// Type returns the rule type, which can be a class, id or a tag.
func (rule Rule) Type() string {
	if strings.HasPrefix(string(rule), ".") {
		return "class"
	}
	if strings.HasPrefix(string(rule), "#") {
		return "id"
	}
	return "tag"
}

func (e TokenEntry) typ() tokenType {
	return newTokenType(e.value)
}

func (t *tokenizer) next() (TokenEntry, error) {
	token := t.s.Scan()
	if token == scanner.EOF {
		return TokenEntry{}, errors.New("EOF")
	}
	value := t.s.TokenText()
	pos := t.s.Pos()
	if newTokenType(value).String() == "STYLE_SEPARATOR" {
		t.s.IsIdentRune = func(ch rune, i int) bool { // property value can contain spaces
			if ch == -1 || ch == '\n' || ch == '\t' || ch == ':' || ch == ';' {
				return false
			}
			return true
		}
	} else {
		t.s.IsIdentRune = func(ch rune, i int) bool { // other tokens can't contain spaces
			if ch == -1 || ch == '.' || ch == '#' || ch == '\n' || ch == ' ' || ch == '\t' || ch == ':' || ch == ';' {
				return false
			}
			return true
		}
	}
	return TokenEntry{
		value: value,
		pos:   pos,
	}, nil
}

func (t tokenType) String() string {
	switch t {
	case tokenBlockStart:
		return "BLOCK_START"
	case tokenBlockEnd:
		return "BLOCK_END"
	case tokenStyleSeparator:
		return "STYLE_SEPARATOR"
	case tokenStatementEnd:
		return "STATEMENT_END"
	case tokenSelector:
		return "SELECTOR"
	}
	return "VALUE"
}

func newTokenType(typ string) tokenType {
	switch typ {
	case "{":
		return tokenBlockStart
	case "}":
		return tokenBlockEnd
	case ":":
		return tokenStyleSeparator
	case ";":
		return tokenStatementEnd
	case ".", "#":
		return tokenSelector
	}
	return tokenValue
}

func newTokenizer(r io.Reader) *tokenizer {
	s := &scanner.Scanner{}
	s.Init(r)
	return &tokenizer{
		s: s,
	}
}

func buildList(r io.Reader) *list.List {
	l := list.New()
	t := newTokenizer(r)
	for {
		token, err := t.next()
		if err != nil {
			break
		}
		l.PushBack(token)
	}
	return l
}

// TODO: rules can be comma separated
func parse(l *list.List) (map[Rule]map[string]string, error) {

	var (
		// Information about the current block that is parsed.
		rule     []string
		style    string
		value    string
		selector string

		isBlock bool

		// Parsed styles.
		css    = make(map[Rule]map[string]string)
		styles = make(map[string]string)

		// Previous token for the state machine.
		prevToken = tokenType(tokenFirstToken)
	)

	for e := l.Front(); e != nil; e = l.Front() {
		token := e.Value.(TokenEntry)
		l.Remove(e)
		switch token.typ() {
		case tokenValue:
			//fmt.Printf("typ: %v, value: %q, prevToken: %v\n", token.typ(), token.value, prevToken)
			switch prevToken {
			case tokenFirstToken, tokenBlockEnd:
				rule = append(rule, token.value)
			case tokenSelector:
				rule = append(rule, selector+token.value)
			case tokenBlockStart, tokenStatementEnd:
				style = token.value
			case tokenStyleSeparator:
				value = token.value
			case tokenValue:
				rule = append(rule, token.value)
			default:
				return css, fmt.Errorf("line %d: invalid syntax", token.pos.Line)
			}
		case tokenSelector:
			selector = token.value
		case tokenBlockStart:
			if prevToken != tokenValue {
				return css, fmt.Errorf("line %d: block is missing rule identifier", token.pos.Line)
			}
			isBlock = true
		case tokenStatementEnd:
			//fmt.Printf("prevToken: %v, style: %v, value: %v\n", prevToken, style, value)
			if prevToken != tokenValue || style == "" || value == "" {
				return css, fmt.Errorf("line %d: expected style before semicolon", token.pos.Line)
			}
			styles[style] = value
		case tokenBlockEnd:
			if !isBlock {
				return css, fmt.Errorf("line %d: rule block ends without a beginning", token.pos.Line)
			}
			for i := range rule {
				oldRule, ok := css[Rule(rule[i])]
				if ok {
					// merge rules
					for style, value := range oldRule {
						if _, ok := styles[style]; !ok {
							styles[style] = value
						}
					}
				}
				css[Rule(rule[i])] = styles

			}
			styles = map[string]string{}
			style, value = "", ""
			isBlock = false
		}
		prevToken = token.typ()
	}
	return css, nil
}

// Unmarshal will take a byte slice, containing sylesheet rules and return
// a map of a rules map.
func Unmarshal(b []byte) (map[Rule]map[string]string, error) {
	return parse(Tokenize(b))
}

// CSSStyle returns an error-checked parsed style, or an error if the
// style is unknown. Most of the styles are not supported yet.
func CSSStyle(name string, styles map[string]string) (Style, error) {
	value := styles[name]
	styleFn, ok := StylesTable[name]
	if !ok {
		return Style{}, errors.New("unknown style")
	}
	return styleFn(value)
}

// Tokenize builds a token list from css bytes
func Tokenize(b []byte) *list.List {
	return buildList(bytes.NewReader(b))
}

// Rules will return all the rules, including duplicated rules.
func Rules(tokens *list.List) []Rule {
	var (
		selector  = ""
		rule      = []string{}
		rules     = []Rule{}
		prevToken = tokenType(tokenFirstToken)
	)

	e := tokens.Front()
	for e != nil {
		tok := e.Value.(TokenEntry)

		switch tok.typ() {
		case tokenValue:
			switch prevToken {
			case tokenBlockEnd, tokenFirstToken, tokenValue:
				rule = append(rule, tok.value)
			case tokenSelector:
				rule = append(rule, selector+tok.value)
			}
		case tokenSelector:
			selector = "."
		case tokenBlockStart:
			rules = append(rules, Rule(strings.Join(rule, " ")))
			rule = []string{}
		}

		prevToken = tok.typ()
		e = e.Next()
	}

	return rules
}

// Comments returns all css comments
func Comments(b []byte) []string {
	return rComments.FindAllString(string(b), -1)
}

// Licenses returns all css License. Will return empty string if no
// license exists .
func Licenses(b []byte) []string {
	return rLicenses.FindAllString(string(b), -1)
}

// BlockCount returns the number code blocks in the css
func BlockCount(tokens *list.List) int {
	count := 0

	e := tokens.Front()
	for e != nil {
		tok := e.Value.(TokenEntry)
		if tok.typ() == tokenBlockStart {
			count++
		}
		e = e.Next()
	}

	return count
}

// Names returns all class, id and element identifiers including
// duplicants
func Names(tokens *list.List) []string {
	var (
		names = []string{}
		prev  = TokenEntry{}
	)

	e := tokens.Front()
	for e != nil {
		tok := e.Value.(TokenEntry)

		if prev.typ() == tokenSelector {
			names = append(names, prev.value+tok.value)
		} else if prev.typ() == tokenBlockEnd &&
			tok.typ() != tokenSelector {
			names = append(names, tok.value)
		}

		prev = tok
		e = e.Next()
	}

	return names
}
