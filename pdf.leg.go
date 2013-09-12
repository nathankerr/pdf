
package pdf

// TODO:
// Whitespace should include \000, but including it makes the class not work properly

import (
	"fmt"
)


const (
	ruleFile = iota
	ruleHeader
	ruleComment
	ruleBody
	ruleIndirectObject
	ruleNonNegativeInteger
	rulePositiveInteger
	ruleRegular
	ruleWhitespace
	ruleDelimiter
)

type yyParser struct {
	File
	Buffer string
	Min, Max int
	rules [10]func() bool
	ResetBuffer	func(string) string
}

func (p *yyParser) Parse(ruleId int) (err error) {
	if p.rules[ruleId]() {
		return
	}
	return p.parseErr()
}

type errPos struct {
	Line, Pos int
}

func	(e *errPos) String() string {
	return fmt.Sprintf("%d:%d", e.Line, e.Pos)
}

type unexpectedCharError struct {
	After, At	errPos
	Char	byte
}

func (e *unexpectedCharError) Error() string {
	return fmt.Sprintf("%v: unexpected character '%c'", &e.At, e.Char)
}

type unexpectedEOFError struct {
	After errPos
}

func (e *unexpectedEOFError) Error() string {
	return fmt.Sprintf("%v: unexpected end of file", &e.After)
}

func (p *yyParser) parseErr() (err error) {
	var pos, after errPos
	pos.Line = 1
	for i, c := range p.Buffer[0:] {
		if c == '\n' {
			pos.Line++
			pos.Pos = 0
		} else {
			pos.Pos++
		}
		if i == p.Min {
			if p.Min != p.Max {
				after = pos
			} else {
				break
			}
		} else if i == p.Max {
			break
		}
	}
	if p.Max >= len(p.Buffer) {
		err = &unexpectedEOFError{after}
	} else {
		err = &unexpectedCharError{after, pos, p.Buffer[p.Max]}
	}
	return
}

func (p *yyParser) Init() {
	var position int

	actions := [...]func(string, int){
		/* 0 Header */
		func(yytext string, _ int) {
			 p.File = append(p.File, Header(yytext)) 
		},
		/* 1 Body */
		func(yytext string, _ int) {
			p.File = append(p.File, "body")
		},
		/* 2 IndirectObject */
		func(yytext string, _ int) {
			p.File = append(p.File, "indirect")
		},
		/* 3 NonNegativeInteger */
		func(yytext string, _ int) {
			p.File = append(p.File, "non-negative")
		},
		/* 4 PositiveInteger */
		func(yytext string, _ int) {
			p.File = append(p.File, "positive")
		},

	}

	type thunk struct {
		action uint8
		begin, end int
	}
	var thunkPosition, begin, end int
	thunks := make([]thunk, 32)
	doarg := func(action uint8, arg int) {
		if thunkPosition == len(thunks) {
			newThunks := make([]thunk, 2*len(thunks))
			copy(newThunks, thunks)
			thunks = newThunks
		}
		t := &thunks[thunkPosition]
		thunkPosition++
		t.action = action
		if arg != 0 {
			t.begin = arg // use begin to store an argument
		} else {
			t.begin = begin
		}
		t.end = end
	}
	do := func(action uint8) {
		doarg(action, 0)
	}

	p.ResetBuffer = func(s string) (old string) {
		if position < len(p.Buffer) {
			old = p.Buffer[position:]
		}
		p.Buffer = s
		thunkPosition = 0
		position = 0
		p.Min = 0
		p.Max = 0
		end = 0
		return
	}

	commit := func(thunkPosition0 int) bool {
		if thunkPosition0 == 0 {
			s := ""
			for _, t := range thunks[:thunkPosition] {
				b := t.begin
				if b >= 0 && b <= t.end {
					s = p.Buffer[b:t.end]
				}
				magic := b
				actions[t.action](s, magic)
			}
			p.Min = position
			thunkPosition = 0
			return true
		}
		return false
	}
	matchDot := func() bool {
		if position < len(p.Buffer) {
			position++
			return true
		} else if position >= p.Max {
			p.Max = position
		}
		return false
	}

	matchChar := func(c byte) bool {
		if (position < len(p.Buffer)) && (p.Buffer[position] == c) {
			position++
			return true
		} else if position >= p.Max {
			p.Max = position
		}
		return false
	}


	matchString := func(s string) bool {
		length := len(s)
		next := position + length
		if (next <= len(p.Buffer)) && p.Buffer[position] == s[0] && (p.Buffer[position:next] == s) {
			position = next
			return true
		} else if position >= p.Max {
			p.Max = position
		}
		return false
	}

	classes := [...][32]uint8{
	0:	{0, 0, 0, 0, 0, 0, 255, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	1:	{0, 0, 0, 0, 0, 0, 255, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	2:	{0, 0, 0, 0, 0, 0, 254, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	4:	{0, 0, 0, 0, 0, 0, 251, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	3:	{0, 54, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	matchClass := func(class uint) bool {
		if (position < len(p.Buffer)) &&
			((classes[class][p.Buffer[position]>>3] & (1 << (p.Buffer[position] & 7))) != 0) {
			position++
			return true
		} else if position >= p.Max {
			p.Max = position
		}
		return false
	}


	p.rules = [...]func() bool{

		/* 0 File <- (Header Whitespace+ Comment Body .+ !. commit) */
		func() bool {
			position0, thunkPosition0 := position, thunkPosition
			if !p.rules[ruleHeader]() {
				goto l0
			}
			if !p.rules[ruleWhitespace]() {
				goto l0
			}
		l1:
			{
				position2, thunkPosition2 := position, thunkPosition
				if !p.rules[ruleWhitespace]() {
					goto l2
				}
				goto l1
			l2:
				position, thunkPosition = position2, thunkPosition2
			}
			if !p.rules[ruleComment]() {
				goto l0
			}
			if !p.rules[ruleBody]() {
				goto l0
			}
			if !matchDot() {
				goto l0
			}
		l3:
			{
				position4, thunkPosition4 := position, thunkPosition
				if !matchDot() {
					goto l4
				}
				goto l3
			l4:
				position, thunkPosition = position4, thunkPosition4
			}
			{
				position5, thunkPosition5 := position, thunkPosition
				if !matchDot() {
					goto l5
				}
				goto l0
			l5:
				position, thunkPosition = position5, thunkPosition5
			}
			if !(commit(thunkPosition0)) {
				goto l0
			}
			return true
		l0:
			position, thunkPosition = position0, thunkPosition0
			return false
		},
		/* 1 Header <- ('%' < 'PDF-1.' [0-7] > { p.File = append(p.File, Header(yytext)) }) */
		func() bool {
			position0, thunkPosition0 := position, thunkPosition
			if !matchChar('%') {
				goto l6
			}
			begin = position
			if !matchString("PDF-1.") {
				goto l6
			}
			if !matchClass(0) {
				goto l6
			}
			end = position
			do(0)
			return true
		l6:
			position, thunkPosition = position0, thunkPosition0
			return false
		},
		/* 2 Comment <- ('%' (!'\n' .)+ '\n') */
		func() bool {
			position0, thunkPosition0 := position, thunkPosition
			if !matchChar('%') {
				goto l7
			}
			{
				position10, thunkPosition10 := position, thunkPosition
				if !matchChar('\n') {
					goto l10
				}
				goto l7
			l10:
				position, thunkPosition = position10, thunkPosition10
			}
			if !matchDot() {
				goto l7
			}
		l8:
			{
				position9, thunkPosition9 := position, thunkPosition
				{
					position11, thunkPosition11 := position, thunkPosition
					if !matchChar('\n') {
						goto l11
					}
					goto l9
				l11:
					position, thunkPosition = position11, thunkPosition11
				}
				if !matchDot() {
					goto l9
				}
				goto l8
			l9:
				position, thunkPosition = position9, thunkPosition9
			}
			if !matchChar('\n') {
				goto l7
			}
			return true
		l7:
			position, thunkPosition = position0, thunkPosition0
			return false
		},
		/* 3 Body <- (IndirectObject+ {p.File = append(p.File, "body")}) */
		func() bool {
			position0, thunkPosition0 := position, thunkPosition
			if !p.rules[ruleIndirectObject]() {
				goto l12
			}
		l13:
			{
				position14, thunkPosition14 := position, thunkPosition
				if !p.rules[ruleIndirectObject]() {
					goto l14
				}
				goto l13
			l14:
				position, thunkPosition = position14, thunkPosition14
			}
			do(1)
			return true
		l12:
			position, thunkPosition = position0, thunkPosition0
			return false
		},
		/* 4 IndirectObject <- (PositiveInteger Whitespace+ NonNegativeInteger Whitespace+ 'obj' {p.File = append(p.File, "indirect")}) */
		func() bool {
			position0, thunkPosition0 := position, thunkPosition
			if !p.rules[rulePositiveInteger]() {
				goto l15
			}
			if !p.rules[ruleWhitespace]() {
				goto l15
			}
		l16:
			{
				position17, thunkPosition17 := position, thunkPosition
				if !p.rules[ruleWhitespace]() {
					goto l17
				}
				goto l16
			l17:
				position, thunkPosition = position17, thunkPosition17
			}
			if !p.rules[ruleNonNegativeInteger]() {
				goto l15
			}
			if !p.rules[ruleWhitespace]() {
				goto l15
			}
		l18:
			{
				position19, thunkPosition19 := position, thunkPosition
				if !p.rules[ruleWhitespace]() {
					goto l19
				}
				goto l18
			l19:
				position, thunkPosition = position19, thunkPosition19
			}
			if !matchString("obj") {
				goto l15
			}
			do(2)
			return true
		l15:
			position, thunkPosition = position0, thunkPosition0
			return false
		},
		/* 5 NonNegativeInteger <- ([0-9] {p.File = append(p.File, "non-negative")}) */
		func() bool {
			position0, thunkPosition0 := position, thunkPosition
			if !matchClass(1) {
				goto l20
			}
			do(3)
			return true
		l20:
			position, thunkPosition = position0, thunkPosition0
			return false
		},
		/* 6 PositiveInteger <- ([1-9] {p.File = append(p.File, "positive")}) */
		func() bool {
			position0, thunkPosition0 := position, thunkPosition
			if !matchClass(2) {
				goto l21
			}
			do(4)
			return true
		l21:
			position, thunkPosition = position0, thunkPosition0
			return false
		},
		/* 7 Regular <- (!(Whitespace / Delimiter) .) */
		func() bool {
			position0, thunkPosition0 := position, thunkPosition
			{
				position23, thunkPosition23 := position, thunkPosition
				{
					position24, thunkPosition24 := position, thunkPosition
					if !p.rules[ruleWhitespace]() {
						goto l25
					}
					goto l24
				l25:
					position, thunkPosition = position24, thunkPosition24
					if !p.rules[ruleDelimiter]() {
						goto l23
					}
				}
			l24:
				goto l22
			l23:
				position, thunkPosition = position23, thunkPosition23
			}
			if !matchDot() {
				goto l22
			}
			return true
		l22:
			position, thunkPosition = position0, thunkPosition0
			return false
		},
		/* 8 Whitespace <- [\t\n\f\r ] */
		func() bool {
			position0, thunkPosition0 := position, thunkPosition
			if !matchClass(3) {
				goto l26
			}
			return true
		l26:
			position, thunkPosition = position0, thunkPosition0
			return false
		},
		/* 9 Delimiter <- [\050\051\060\133\173\175\057\045] */
		func() bool {
			position0, thunkPosition0 := position, thunkPosition
			if !matchClass(4) {
				goto l27
			}
			return true
		l27:
			position, thunkPosition = position0, thunkPosition0
			return false
		},
	}
}
