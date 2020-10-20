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

// Package textenc encodes the abstract syntax tree into its text.
package textenc

import (
	"io"

	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/encoder"
)

func init() {
	encoder.Register("text", encoder.Info{
		Create: func() encoder.Encoder { return &textEncoder{} },
	})
}

type textEncoder struct{}

// SetOption sets an option for this encoder
func (te *textEncoder) SetOption(option encoder.Option) {}

// WriteZettel does nothing.
func (te *textEncoder) WriteZettel(w io.Writer, zettel *ast.Zettel) (int, error) {
	v := newVisitor(w)
	te.WriteMeta(&v.b, zettel.Meta)
	v.acceptBlockSlice(zettel.Ast)
	length, err := v.b.Flush()
	return length, err
}

// WriteMeta encodes meta data as text.
func (te *textEncoder) WriteMeta(w io.Writer, meta *domain.Meta) (int, error) {
	b := encoder.NewBufWriter(w)
	for _, pair := range meta.Pairs() {
		b.WriteString(pair.Value)
		b.WriteByte('\n')
	}
	length, err := b.Flush()
	return length, err
}

func (te *textEncoder) WriteContent(w io.Writer, zettel *ast.Zettel) (int, error) {
	return te.WriteBlocks(w, zettel.Ast)
}

// WriteBlocks writes the content of a block slice to the writer.
func (te *textEncoder) WriteBlocks(w io.Writer, bs ast.BlockSlice) (int, error) {
	v := newVisitor(w)
	v.acceptBlockSlice(bs)
	length, err := v.b.Flush()
	return length, err
}

// WriteInlines writes an inline slice to the writer
func (te *textEncoder) WriteInlines(w io.Writer, is ast.InlineSlice) (int, error) {
	v := newVisitor(w)
	v.acceptInlineSlice(is)
	length, err := v.b.Flush()
	return length, err
}

// visitor writes the abstract syntax tree to an io.Writer.
type visitor struct {
	b encoder.BufWriter
}

func newVisitor(w io.Writer) *visitor {
	return &visitor{b: encoder.NewBufWriter(w)}
}

// VisitPara emits text code for a paragraph
func (v *visitor) VisitPara(pn *ast.ParaNode) {
	v.acceptInlineSlice(pn.Inlines)
}

// VisitVerbatim emits text for verbatim lines.
func (v *visitor) VisitVerbatim(vn *ast.VerbatimNode) {
	if vn.Code == ast.VerbatimComment {
		return
	}
	for i, line := range vn.Lines {
		if i > 0 {
			v.b.WriteByte('\n')
		}
		v.b.WriteString(line)
	}
}

// VisitRegion writes text code for block regions.
func (v *visitor) VisitRegion(rn *ast.RegionNode) {
	v.acceptBlockSlice(rn.Blocks)
	if len(rn.Inlines) > 0 {
		v.b.WriteByte('\n')
		v.acceptInlineSlice(rn.Inlines)
	}
}

// VisitHeading writes the text code for a heading.
func (v *visitor) VisitHeading(hn *ast.HeadingNode) {
	v.acceptInlineSlice(hn.Inlines)
}

// VisitHRule writes nothing for a horizontal rule.
func (v *visitor) VisitHRule(hn *ast.HRuleNode) {}

// VisitNestedList writes text code for lists and blockquotes.
func (v *visitor) VisitNestedList(ln *ast.NestedListNode) {
	for i, item := range ln.Items {
		if i > 0 {
			v.b.WriteByte('\n')
		}
		v.acceptItemSlice(item)
	}
}

// VisitDescriptionList emits a text for a description list.
func (v *visitor) VisitDescriptionList(dn *ast.DescriptionListNode) {
	for i, descr := range dn.Descriptions {
		if i > 0 {
			v.b.WriteByte('\n')
		}
		v.acceptInlineSlice(descr.Term)

		for _, b := range descr.Descriptions {
			v.b.WriteByte('\n')
			v.acceptDescriptionSlice(b)
		}
	}
}

// VisitTable emits a text table.
func (v *visitor) VisitTable(tn *ast.TableNode) {
	if len(tn.Header) > 0 {
		for i, cell := range tn.Header {
			if i > 0 {
				v.b.WriteByte(' ')
			}
			v.acceptInlineSlice(cell.Inlines)
		}
		v.b.WriteByte('\n')
	}
	for i, row := range tn.Rows {
		if i > 0 {
			v.b.WriteByte('\n')
		}
		for j, cell := range row {
			if j > 0 {
				v.b.WriteByte(' ')
			}
			v.acceptInlineSlice(cell.Inlines)
		}
	}
}

// VisitBLOB writes nothing, because it contains no text.
func (v *visitor) VisitBLOB(bn *ast.BLOBNode) {}

// VisitText writes text content.
func (v *visitor) VisitText(tn *ast.TextNode) {
	v.b.WriteString(tn.Text)
}

// VisitTag writes tag content.
func (v *visitor) VisitTag(tn *ast.TagNode) {
	v.b.WriteStrings("#", tn.Tag)
}

// VisitSpace emits a white space.
func (v *visitor) VisitSpace(sn *ast.SpaceNode) {
	v.b.WriteByte(' ')
}

// VisitBreak writes text code for line breaks.
func (v *visitor) VisitBreak(bn *ast.BreakNode) {
	if bn.Hard {
		v.b.WriteByte('\n')
	} else {
		v.b.WriteByte(' ')
	}
}

// VisitLink writes text code for links.
func (v *visitor) VisitLink(ln *ast.LinkNode) {
	v.acceptInlineSlice(ln.Inlines)
}

// VisitImage writes text code for images.
func (v *visitor) VisitImage(in *ast.ImageNode) {
	v.acceptInlineSlice(in.Inlines)
}

// VisitCite writes code for citations.
func (v *visitor) VisitCite(cn *ast.CiteNode) {
	v.acceptInlineSlice(cn.Inlines)
}

// VisitFootnote write text code for a footnote.
func (v *visitor) VisitFootnote(fn *ast.FootnoteNode) {
	v.b.WriteByte(' ')
	v.acceptInlineSlice(fn.Inlines)
}

// VisitMark writes nothing for a mark.
func (v *visitor) VisitMark(mn *ast.MarkNode) {}

// VisitFormat write text code for formatting text.
func (v *visitor) VisitFormat(fn *ast.FormatNode) {
	v.acceptInlineSlice(fn.Inlines)
}

// VisitLiteral write text code for literal inline text.
func (v *visitor) VisitLiteral(ln *ast.LiteralNode) {
	if ln.Code != ast.LiteralComment {
		v.b.WriteString(ln.Text)
	}
}

// VisitAttributes never writes any attribute data.
func (v *visitor) VisitAttributes(a *ast.Attributes) {}

func (v *visitor) acceptBlockSlice(bns ast.BlockSlice) {
	for i, bn := range bns {
		if i > 0 {
			v.b.WriteByte('\n')
		}
		bn.Accept(v)
	}
}
func (v *visitor) acceptItemSlice(ins ast.ItemSlice) {
	for i, in := range ins {
		if i > 0 {
			v.b.WriteByte('\n')
		}
		in.Accept(v)
	}
}
func (v *visitor) acceptDescriptionSlice(dns ast.DescriptionSlice) {
	for i, dn := range dns {
		if i > 0 {
			v.b.WriteByte('\n')
		}
		dn.Accept(v)
	}
}
func (v *visitor) acceptInlineSlice(ins ast.InlineSlice) {
	for _, in := range ins {
		in.Accept(v)
	}
}
