//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is free software: you can redistribute it and/or modify it under
// the terms of the GNU Affero General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option)
// any later version.
//
// Zettelstore is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
// FITNESS FOR A PARTICULAR PURPOSE. See the GNU Affero General Public License
// for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with Zettelstore. If not, see <http://www.gnu.org/licenses/>.
//-----------------------------------------------------------------------------

// Package zettelmark provides a parser for zettelmarkup.
package zettelmark

import (
	"fmt"
	"strings"

	"zettelstore.de/z/ast"
	"zettelstore.de/z/input"
)

// parseInlineSlice parses a sequence of Inlines until EOS.
func (cp *zmkP) parseInlineSlice() ast.InlineSlice {
	inp := cp.inp
	var ins ast.InlineSlice
	for inp.Ch != input.EOS {
		in := cp.parseInline()
		if in == nil {
			return ins
		}
		ins = append(ins, in)
	}
	return ins
}

func (cp *zmkP) parseInline() ast.InlineNode {
	inp := cp.inp
	pos := inp.Pos
	if cp.nestingLevel <= maxNestingLevel {
		cp.nestingLevel++
		defer func() { cp.nestingLevel-- }()

		var in ast.InlineNode
		success := false
		switch inp.Ch {
		case input.EOS:
			return nil
		case '\n', '\r':
			return cp.parseSoftBreak()
		case ' ', '\t':
			return cp.parseSpace()
		case '[':
			inp.Next()
			switch inp.Ch {
			case '[':
				in, success = cp.parseLink()
			case '@':
				in, success = cp.parseCite()
			case '^':
				in, success = cp.parseFootnote()
			case '!':
				in, success = cp.parseMark()
			}
		case '{':
			inp.Next()
			switch inp.Ch {
			case '{':
				in, success = cp.parseImage()
			}
		case '#':
			return cp.parseTag()
		case '%':
			in, success = cp.parseComment()
		case '/', '*', '_', '~', '\'', '^', ',', '<', '"', ';', ':':
			in, success = cp.parseFormat()
		case '+', '`', '=', runeModGrave:
			in, success = cp.parseLiteral()
		case '\\':
			return cp.parseBackslash()
		case '-':
			in, success = cp.parseNdash()
		case '&':
			in, success = cp.parseEntity()
		}
		if success {
			return in
		}
	}
	inp.SetPos(pos)
	return cp.parseText()
}

func (cp *zmkP) parseText() *ast.TextNode {
	inp := cp.inp
	pos := inp.Pos
	if inp.Ch == '\\' {
		return cp.parseTextBackslash()
	}
	for {
		inp.Next()
		switch inp.Ch {
		// The following case must contain all runes that occur in parseInline!
		// Plus the closing brackets ] and } and ) and the middle |
		case input.EOS, '\n', '\r', ' ', '\t', '[', ']', '{', '}', '(', ')', '|', '#', '%', '/', '*', '_', '~', '\'', '^', ',', '<', '"', ';', ':', '+', '`', runeModGrave, '=', '\\', '-', '&':
			return &ast.TextNode{Text: inp.Src[pos:inp.Pos]}
		}
	}
}

func (cp *zmkP) parseTextBackslash() *ast.TextNode {
	cp.inp.Next()
	return cp.parseBackslashRest()
}

func (cp *zmkP) parseBackslash() ast.InlineNode {
	inp := cp.inp
	inp.Next()
	switch inp.Ch {
	case '\n', '\r':
		inp.EatEOL()
		return &ast.BreakNode{Hard: true}
	default:
		return cp.parseBackslashRest()
	}
}

func (cp *zmkP) parseBackslashRest() *ast.TextNode {
	inp := cp.inp
	switch inp.Ch {
	case input.EOS, '\n', '\r':
		return &ast.TextNode{Text: "\\"}
	case ' ':
		inp.Next()
		return &ast.TextNode{Text: "\u00a0"}
	}
	pos := inp.Pos
	inp.Next()
	return &ast.TextNode{Text: inp.Src[pos:inp.Pos]}
}

func (cp *zmkP) parseSpace() *ast.SpaceNode {
	inp := cp.inp
	pos := inp.Pos
	for {
		inp.Next()
		switch inp.Ch {
		case ' ', '\t':
		default:
			return &ast.SpaceNode{Lexeme: inp.Src[pos:inp.Pos]}
		}
	}
}

func (cp *zmkP) parseSoftBreak() *ast.BreakNode {
	cp.inp.EatEOL()
	return &ast.BreakNode{}
}

func (cp *zmkP) parseLink() (*ast.LinkNode, bool) {
	if ref, ins, ok := cp.parseReference(']'); ok {
		attrs := cp.parseAttributes(false)
		if len(ref) > 0 {
			onlyRef := false
			r := ast.ParseReference(ref)
			if ins == nil {
				ins = ast.InlineSlice{&ast.TextNode{Text: ref}}
				onlyRef = true
			}
			return &ast.LinkNode{
				Ref:     r,
				Inlines: ins,
				OnlyRef: onlyRef,
				Attrs:   attrs,
			}, true
		}
	}
	return nil, false
}

func (cp *zmkP) parseReference(closeCh rune) (ref string, ins ast.InlineSlice, ok bool) {
	inp := cp.inp
	inp.Next()
	for inp.Ch == ' ' {
		inp.Next()
	}
	hasSpace := false
	pos := inp.Pos
loop:
	for {
		switch inp.Ch {
		case input.EOS:
			return "", nil, false
		case '\n', '\r', ' ':
			hasSpace = true
		case '|', closeCh:
			break loop
		}
		inp.Next()
	}
	if inp.Ch == '|' { // First part must be inline text
		if pos == inp.Pos { // [[| or {{|
			return "", nil, false
		}
		inp.SetPos(pos)
	loop1:
		for {
			switch inp.Ch {
			case input.EOS, '|':
				break loop1
			}
			in := cp.parseInline()
			ins = append(ins, in)
		}
		inp.Next()
		pos = inp.Pos
	} else if hasSpace {
		return "", nil, false
	}

	inp.SetPos(pos)
	for inp.Ch == ' ' {
		inp.Next()
		pos = inp.Pos
	}
loop2:
	for {
		switch inp.Ch {
		case input.EOS, '\n', '\r', ' ':
			return "", nil, false
		case closeCh:
			break loop2
		}
		inp.Next()
	}
	ref = inp.Src[pos:inp.Pos]
	inp.Next()
	if inp.Ch != closeCh {
		return "", nil, false
	}
	inp.Next()
	return ref, ins, true
}

func (cp *zmkP) parseCite() (*ast.CiteNode, bool) {
	inp := cp.inp
	inp.Next()
	switch inp.Ch {
	case ' ', ',', '|', ']', '\n', '\r':
		return nil, false
	}
	pos := inp.Pos
loop:
	for {
		switch inp.Ch {
		case input.EOS:
			return nil, false
		case ' ', ',', '|', ']', '\n', '\r':
			break loop
		}
		inp.Next()
	}
	posL := inp.Pos
	switch inp.Ch {
	case ' ', ',', '|':
		inp.Next()
	}
	ins, ok := cp.parseLinkLikeRest()
	if !ok {
		return nil, false
	}
	attrs := cp.parseAttributes(false)
	return &ast.CiteNode{Key: inp.Src[pos:posL], Inlines: ins, Attrs: attrs}, true
}

func (cp *zmkP) parseFootnote() (*ast.FootnoteNode, bool) {
	cp.inp.Next()
	ins, ok := cp.parseLinkLikeRest()
	if !ok {
		return nil, false
	}
	attrs := cp.parseAttributes(false)
	return &ast.FootnoteNode{Inlines: ins, Attrs: attrs}, true
}

func (cp *zmkP) parseLinkLikeRest() (ast.InlineSlice, bool) {
	inp := cp.inp
	for inp.Ch == ' ' {
		inp.Next()
	}
	var ins ast.InlineSlice
	for inp.Ch != ']' {
		in := cp.parseInline()
		if in == nil {
			return nil, false
		}
		ins = append(ins, in)
		if _, ok := in.(*ast.BreakNode); ok {
			ch := cp.inp.Ch
			switch ch {
			case input.EOS, '\n', '\r':
				return nil, false
			}
		}
	}
	inp.Next()
	return ins, true
}

func (cp *zmkP) parseImage() (ast.InlineNode, bool) {
	if ref, ins, ok := cp.parseReference('}'); ok {
		attrs := cp.parseAttributes(false)
		if len(ref) > 0 {
			r := ast.ParseReference(ref)
			return &ast.ImageNode{Ref: r, Inlines: ins, Attrs: attrs}, true
		}
	}
	return nil, false
}

func (cp *zmkP) parseMark() (*ast.MarkNode, bool) {
	inp := cp.inp
	inp.Next()
	pos := inp.Pos
	for inp.Ch != ']' {
		if !isNameRune(inp.Ch) {
			return nil, false
		}
		inp.Next()
	}
	mn := &ast.MarkNode{Text: inp.Src[pos:inp.Pos]}
	inp.Next()
	return mn, true
}

func (cp *zmkP) parseTag() ast.InlineNode {
	inp := cp.inp
	posH := inp.Pos
	inp.Next()
	pos := inp.Pos
	for isNameRune(inp.Ch) {
		inp.Next()
	}
	if pos == inp.Pos || inp.Ch == '#' {
		return &ast.TextNode{Text: inp.Src[posH:inp.Pos]}
	}
	return &ast.TagNode{Tag: inp.Src[pos:inp.Pos]}
}

func (cp *zmkP) parseComment() (res *ast.LiteralNode, success bool) {
	inp := cp.inp
	inp.Next()
	if inp.Ch != '%' {
		return nil, false
	}
	for inp.Ch == '%' {
		inp.Next()
	}
	for inp.Ch == ' ' {
		inp.Next()
	}
	pos := inp.Pos
	for {
		switch inp.Ch {
		case input.EOS, '\n', '\r':
			return &ast.LiteralNode{Code: ast.LiteralComment, Text: inp.Src[pos:inp.Pos]}, true
		}
		inp.Next()
	}
}

var mapRuneFormat = map[rune]ast.FormatCode{
	'/':  ast.FormatItalic,
	'*':  ast.FormatBold,
	'_':  ast.FormatUnder,
	'~':  ast.FormatStrike,
	'\'': ast.FormatMonospace,
	'^':  ast.FormatSuper,
	',':  ast.FormatSub,
	'<':  ast.FormatQuotation,
	'"':  ast.FormatQuote,
	';':  ast.FormatSmall,
	':':  ast.FormatSpan,
}

func (cp *zmkP) parseFormat() (res ast.InlineNode, success bool) {
	inp := cp.inp
	fch := inp.Ch
	code, ok := mapRuneFormat[fch]
	if !ok {
		panic(fmt.Sprintf("%q is not a formatting char", fch))
	}
	inp.Next() // read 2nd formatting character
	if inp.Ch != fch {
		return nil, false
	}
	fn := &ast.FormatNode{Code: code}
	inp.Next()
	for {
		if inp.Ch == input.EOS {
			return nil, false
		}
		if inp.Ch == fch {
			inp.Next()
			if inp.Ch == fch {
				inp.Next()
				fn.Attrs = cp.parseAttributes(false)
				return fn, true
			}
			fn.Inlines = append(fn.Inlines, &ast.TextNode{Text: string(fch)})
		} else if in := cp.parseInline(); in != nil {
			if _, ok := in.(*ast.BreakNode); ok {
				switch inp.Ch {
				case input.EOS, '\n', '\r':
					return nil, false
				}
			}
			fn.Inlines = append(fn.Inlines, in)
		}
	}
}

var mapRuneLiteral = map[rune]ast.LiteralCode{
	'`':          ast.LiteralProg,
	runeModGrave: ast.LiteralProg,
	'+':          ast.LiteralKeyb,
	'=':          ast.LiteralOutput,
}

func (cp *zmkP) parseLiteral() (res ast.InlineNode, success bool) {
	inp := cp.inp
	fch := inp.Ch
	code, ok := mapRuneLiteral[fch]
	if !ok {
		panic(fmt.Sprintf("%q is not a formatting char", fch))
	}
	inp.Next() // read 2nd formatting character
	if inp.Ch != fch {
		return nil, false
	}
	fn := &ast.LiteralNode{Code: code}
	inp.Next()
	var sb strings.Builder
	for {
		if inp.Ch == input.EOS {
			return nil, false
		}
		if inp.Ch == fch {
			if inp.Peek() == fch {
				inp.Next()
				inp.Next()
				fn.Attrs = cp.parseAttributes(false)
				fn.Text = sb.String()
				return fn, true
			}
			sb.WriteRune(fch)
			inp.Next()
		} else {
			tn := cp.parseText()
			sb.WriteString(tn.Text)
		}
	}
}

func (cp *zmkP) parseNdash() (res *ast.TextNode, success bool) {
	inp := cp.inp
	if inp.Peek() != inp.Ch {
		return nil, false
	}
	inp.Next()
	inp.Next()
	return &ast.TextNode{Text: "\u2013"}, true
}

func (cp *zmkP) parseEntity() (res *ast.TextNode, success bool) {
	if text, ok := cp.inp.ScanEntity(); ok {
		return &ast.TextNode{Text: text}, true
	}
	return nil, false
}
