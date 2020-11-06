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

// Package zmkenc encodes the abstract syntax tree back into Zettelmarkup.
package zmkenc

import (
	"fmt"
	"io"
	"sort"

	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/encoder"
)

func init() {
	encoder.Register("zmk", encoder.Info{
		Create: func() encoder.Encoder { return &zmkEncoder{} },
	})
}

type zmkEncoder struct{}

// SetOption sets an option for this encoder.
func (ze *zmkEncoder) SetOption(option encoder.Option) {}

// WriteZettel writes the encoded zettel to the writer.
func (ze *zmkEncoder) WriteZettel(w io.Writer, zettel *ast.Zettel) (int, error) {
	v := newVisitor(w, ze)
	zettel.Meta.WriteAsHeader(&v.b)
	v.acceptBlockSlice(zettel.Ast)
	length, err := v.b.Flush()
	return length, err
}

// WriteMeta encodes meta data as zmk.
func (ze *zmkEncoder) WriteMeta(w io.Writer, meta *domain.Meta) (int, error) {
	return meta.Write(w)
}

func (ze *zmkEncoder) WriteContent(w io.Writer, zettel *ast.Zettel) (int, error) {
	return ze.WriteBlocks(w, zettel.Ast)
}

// WriteBlocks writes the content of a block slice to the writer.
func (ze *zmkEncoder) WriteBlocks(w io.Writer, bs ast.BlockSlice) (int, error) {
	v := newVisitor(w, ze)
	v.acceptBlockSlice(bs)
	length, err := v.b.Flush()
	return length, err
}

// WriteInlines writes an inline slice to the writer
func (ze *zmkEncoder) WriteInlines(w io.Writer, is ast.InlineSlice) (int, error) {
	v := newVisitor(w, ze)
	v.acceptInlineSlice(is)
	length, err := v.b.Flush()
	return length, err
}

// visitor writes the abstract syntax tree to an io.Writer.
type visitor struct {
	b      encoder.BufWriter
	prefix []byte
	enc    *zmkEncoder
}

func newVisitor(w io.Writer, enc *zmkEncoder) *visitor {
	return &visitor{
		b:   encoder.NewBufWriter(w),
		enc: enc,
	}
}

// VisitPara emits HTML code for a paragraph: <p>...</p>
func (v *visitor) VisitPara(pn *ast.ParaNode) {
	v.acceptInlineSlice(pn.Inlines)
	v.b.WriteByte('\n')
	if len(v.prefix) == 0 {
		v.b.WriteByte('\n')
	}
}

// VisitVerbatim emits HTML code for verbatim lines.
func (v *visitor) VisitVerbatim(vn *ast.VerbatimNode) {
	// TODO: scan cn.Lines to find embedded "`"s at beginning
	v.b.WriteString("```")
	v.visitAttributes(vn.Attrs)
	v.b.WriteByte('\n')
	for _, line := range vn.Lines {
		v.b.WriteStrings(line, "\n")
	}
	v.b.WriteString("```\n")
}

var regionCode = map[ast.RegionCode]string{
	ast.RegionSpan:  ":::",
	ast.RegionQuote: "<<<",
	ast.RegionVerse: "\"\"\"",
}

// VisitRegion writes HTML code for block regions.
func (v *visitor) VisitRegion(rn *ast.RegionNode) {
	// Scan rn.Blocks for embedded regions to adjust length of regionCode
	code, ok := regionCode[rn.Code]
	if !ok {
		panic(fmt.Sprintf("Unknown region code %d", rn.Code))
	}
	v.b.WriteString(code)
	v.visitAttributes(rn.Attrs)
	v.b.WriteByte('\n')
	v.acceptBlockSlice(rn.Blocks)
	v.b.WriteString(code)
	if len(rn.Inlines) > 0 {
		v.b.WriteByte(' ')
		v.acceptInlineSlice(rn.Inlines)
	}
	v.b.WriteByte('\n')
}

// VisitHeading writes the HTML code for a heading.
func (v *visitor) VisitHeading(hn *ast.HeadingNode) {
	for i := 0; i <= hn.Level; i++ {
		v.b.WriteByte('=')
	}
	v.b.WriteByte(' ')
	v.acceptInlineSlice(hn.Inlines)
	v.visitAttributes(hn.Attrs)
	v.b.WriteByte('\n')
}

// VisitHRule writes HTML code for a horizontal rule: <hr>.
func (v *visitor) VisitHRule(hn *ast.HRuleNode) {
	v.b.WriteString("---")
	v.visitAttributes(hn.Attrs)
	v.b.WriteByte('\n')
}

var listCode = map[ast.NestedListCode]byte{
	ast.NestedListOrdered:   '#',
	ast.NestedListUnordered: '*',
	ast.NestedListQuote:     '>',
}

// VisitNestedList writes HTML code for lists and blockquotes.
func (v *visitor) VisitNestedList(ln *ast.NestedListNode) {
	v.prefix = append(v.prefix, listCode[ln.Code])
	for _, item := range ln.Items {
		v.b.Write(v.prefix)
		v.b.WriteByte(' ')
		for i, in := range item {
			if i > 0 {
				if _, ok := in.(*ast.ParaNode); ok {
					v.b.WriteByte('\n')
					for j := 0; j <= len(v.prefix); j++ {
						v.b.WriteByte(' ')
					}
				}
			}
			in.Accept(v)
		}
	}
	v.prefix = v.prefix[:len(v.prefix)-1]
	v.b.WriteByte('\n')
}

// VisitDescriptionList emits a HTML description list.
func (v *visitor) VisitDescriptionList(dn *ast.DescriptionListNode) {
	for _, descr := range dn.Descriptions {
		v.b.WriteString("; ")
		v.acceptInlineSlice(descr.Term)
		v.b.WriteByte('\n')

		for _, b := range descr.Descriptions {
			v.b.WriteString(": ")
			for _, dn := range b {
				dn.Accept(v)
			}
			v.b.WriteByte('\n')
		}
	}
}

var alignCode = map[ast.Alignment]string{
	ast.AlignDefault: "",
	ast.AlignLeft:    ">",
	ast.AlignCenter:  ":",
	ast.AlignRight:   ">",
}

// VisitTable emits a HTML table.
func (v *visitor) VisitTable(tn *ast.TableNode) {
	if len(tn.Header) > 0 {
		for _, cell := range tn.Header {
			v.b.WriteString("|=")
			v.acceptInlineSlice(cell.Inlines)
			v.b.WriteString(alignCode[cell.Align])
		}
		v.b.WriteByte('\n')
	}
	for _, row := range tn.Rows {
		for _, cell := range row {
			v.b.WriteStrings("|", alignCode[cell.Align])
			v.acceptInlineSlice(cell.Inlines)
		}
		v.b.WriteByte('\n')
	}
	v.b.WriteByte('\n')
}

// VisitBLOB writes the binary object as a value.
func (v *visitor) VisitBLOB(bn *ast.BLOBNode) {
	v.b.WriteStrings("%% Unable to display BLOB with title '", bn.Title, "' and syntax '", bn.Syntax, "'\n")
}

var escapeSeqs = map[string]bool{
	"\\":   true,
	"//":   true,
	"**":   true,
	"__":   true,
	"~~":   true,
	"^^":   true,
	",,":   true,
	"<<":   true,
	"\"\"": true,
	";;":   true,
	"::":   true,
	"''":   true,
	"``":   true,
	"++":   true,
	"==":   true,
}

// VisitText writes text content.
func (v *visitor) VisitText(tn *ast.TextNode) {
	last := 0
	for i := 0; i < len(tn.Text); i++ {
		if b := tn.Text[i]; b == '\\' {
			v.b.WriteString(tn.Text[last:i])
			v.b.WriteBytes('\\', b)
			last = i + 1
			continue
		}
		if i < len(tn.Text)-1 {
			s := tn.Text[i : i+2]
			if _, ok := escapeSeqs[s]; ok {
				v.b.WriteString(tn.Text[last:i])
				for j := 0; j < len(s); j++ {
					v.b.WriteBytes('\\', s[j])
				}
				last = i + 1
				continue
			}
		}
	}
	v.b.WriteString(tn.Text[last:])
}

// VisitTag writes tag content.
func (v *visitor) VisitTag(tn *ast.TagNode) {
	v.b.WriteStrings("#", tn.Tag)
}

// VisitSpace emits a white space.
func (v *visitor) VisitSpace(sn *ast.SpaceNode) {
	v.b.WriteString(sn.Lexeme)
}

// VisitBreak writes HTML code for line breaks.
func (v *visitor) VisitBreak(bn *ast.BreakNode) {
	if bn.Hard {
		v.b.WriteString("\\\n")
	} else {
		v.b.WriteByte('\n')
	}
	if prefixLen := len(v.prefix); prefixLen > 0 {
		for i := 0; i <= prefixLen; i++ {
			v.b.WriteByte(' ')
		}
	}
}

// VisitLink writes HTML code for links.
func (v *visitor) VisitLink(ln *ast.LinkNode) {
	v.b.WriteString("[[")
	v.acceptInlineSlice(ln.Inlines)
	v.b.WriteByte('|')
	v.b.WriteStrings(ln.Ref.String(), "]]")
}

// VisitImage writes HTML code for images.
func (v *visitor) VisitImage(in *ast.ImageNode) {
	if in.Ref != nil {
		v.b.WriteString("{{")
		if len(in.Inlines) > 0 {
			v.acceptInlineSlice(in.Inlines)
			v.b.WriteByte('|')
		}
		v.b.WriteStrings(in.Ref.String(), "}}")
	}
}

// VisitCite writes code for citations.
func (v *visitor) VisitCite(cn *ast.CiteNode) {
	v.b.WriteStrings("[@", cn.Key)
	if len(cn.Inlines) > 0 {
		v.b.WriteString(", ")
		v.acceptInlineSlice(cn.Inlines)
	}
	v.b.WriteByte(']')
	v.visitAttributes(cn.Attrs)
}

// VisitFootnote write HTML code for a footnote.
func (v *visitor) VisitFootnote(fn *ast.FootnoteNode) {
	v.b.WriteString("[^")
	v.acceptInlineSlice(fn.Inlines)
	v.b.WriteByte(']')
	v.visitAttributes(fn.Attrs)
}

// VisitMark writes HTML code to mark a position.
func (v *visitor) VisitMark(mn *ast.MarkNode) {
	v.b.WriteStrings("[!", mn.Text, "]")
}

var formatCode = map[ast.FormatCode][]byte{
	ast.FormatItalic:    []byte("//"),
	ast.FormatEmph:      []byte("//"),
	ast.FormatBold:      []byte("**"),
	ast.FormatStrong:    []byte("**"),
	ast.FormatUnder:     []byte("__"),
	ast.FormatInsert:    []byte("__"),
	ast.FormatStrike:    []byte("~~"),
	ast.FormatDelete:    []byte("~~"),
	ast.FormatSuper:     []byte("^^"),
	ast.FormatSub:       []byte(",,"),
	ast.FormatQuotation: []byte("<<"),
	ast.FormatQuote:     []byte("\"\""),
	ast.FormatSmall:     []byte(";;"),
	ast.FormatSpan:      []byte("::"),
	ast.FormatMonospace: []byte("''"),
}

// VisitFormat write HTML code for formatting text.
func (v *visitor) VisitFormat(fn *ast.FormatNode) {
	code, ok := formatCode[fn.Code]
	if !ok {
		panic(fmt.Sprintf("Unknown format code %d", fn.Code))
	}
	attrs := fn.Attrs
	switch fn.Code {
	case ast.FormatEmph, ast.FormatStrong, ast.FormatInsert, ast.FormatDelete:
		attrs = attrs.Clone()
		attrs.Set("-", "")
	}

	v.b.Write(code)
	v.acceptInlineSlice(fn.Inlines)
	v.b.Write(code)
	v.visitAttributes(attrs)
}

// VisitLiteral write Zettelmarkup for inline literal text.
func (v *visitor) VisitLiteral(ln *ast.LiteralNode) {
	switch ln.Code {
	case ast.LiteralProg:
		v.writeLiteral('`', ln.Attrs, ln.Text)
	case ast.LiteralKeyb:
		v.writeLiteral('+', ln.Attrs, ln.Text)
	case ast.LiteralOutput:
		v.writeLiteral('=', ln.Attrs, ln.Text)
	case ast.LiteralComment:
		v.b.WriteStrings("%% ", ln.Text)
	case ast.LiteralHTML:
		v.b.WriteString("``")
		v.writeEscaped(ln.Text, '`')
		v.b.WriteString("``{=html,.warning}")
	default:
		panic(fmt.Sprintf("Unknown literal code %v", ln.Code))
	}
}

func (v *visitor) writeLiteral(code byte, attrs *ast.Attributes, text string) {
	v.b.WriteBytes(code, code)
	v.writeEscaped(text, code)
	v.b.WriteBytes(code, code)
	v.visitAttributes(attrs)
}

func (v *visitor) acceptBlockSlice(bns ast.BlockSlice) {
	for _, bn := range bns {
		bn.Accept(v)
	}
}
func (v *visitor) acceptInlineSlice(ins ast.InlineSlice) {
	for _, in := range ins {
		in.Accept(v)
	}
}

// visitAttributes write HTML attributes
func (v *visitor) visitAttributes(a *ast.Attributes) {
	if a == nil || len(a.Attrs) == 0 {
		return
	}
	keys := make([]string, 0, len(a.Attrs))
	for k := range a.Attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	v.b.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			v.b.WriteByte(' ')
		}
		if k == "-" {
			v.b.WriteByte('-')
			continue
		}
		v.b.WriteString(k)
		if vl := a.Attrs[k]; len(vl) > 0 {
			v.b.WriteStrings("=\"", vl)
			v.b.WriteByte('"')
		}
	}
	v.b.WriteByte('}')
}

func (v *visitor) writeEscaped(s string, toEscape byte) {
	last := 0
	for i := 0; i < len(s); i++ {
		if b := s[i]; b == toEscape || b == '\\' {
			v.b.WriteString(s[last:i])
			v.b.WriteBytes('\\', b)
			last = i + 1
		}
	}
	v.b.WriteString(s[last:])
}
