//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package htmlenc encodes the abstract syntax tree into HTML5.
package htmlenc

import (
	"fmt"
	"io"
	"strings"

	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/encoder"
)

func init() {
	encoder.Register("html", encoder.Info{
		Create: func() encoder.Encoder { return &htmlEncoder{} },
	})
}

type htmlEncoder struct {
	lang           string // default language
	xhtml          bool   // use XHTML syntax instead of HTML syntax
	markerExternal string // Marker after link to (external) material.
	newWindow      bool   // open link in new window
	adaptLink      func(*ast.LinkNode) ast.InlineNode
	adaptImage     func(*ast.ImageNode) ast.InlineNode
	adaptCite      func(*ast.CiteNode) ast.InlineNode
	ignoreMeta     map[string]bool
	footnotes      []*ast.FootnoteNode
}

func (he *htmlEncoder) SetOption(option encoder.Option) {
	switch opt := option.(type) {
	case *encoder.StringOption:
		switch opt.Key {
		case "lang":
			he.lang = opt.Value
		case domain.MetaKeyMarkerExternal:
			he.markerExternal = opt.Value
		}
	case *encoder.BoolOption:
		switch opt.Key {
		case "newwindow":
			he.newWindow = opt.Value
		case "xhtml":
			he.xhtml = opt.Value
		}
	case *encoder.StringsOption:
		switch opt.Key {
		case "no-meta":
			he.ignoreMeta = make(map[string]bool, len(opt.Value))
			for _, v := range opt.Value {
				he.ignoreMeta[v] = true
			}
		}
	case *encoder.AdaptLinkOption:
		he.adaptLink = opt.Adapter
	case *encoder.AdaptImageOption:
		he.adaptImage = opt.Adapter
	case *encoder.AdaptCiteOption:
		he.adaptCite = opt.Adapter
	default:
		var name string
		if option != nil {
			name = option.Name()
		}
		fmt.Println("HESO", option, name)
	}
}

// WriteZettel encodes a full zettel as HTML5.
func (he *htmlEncoder) WriteZettel(w io.Writer, zn *ast.ZettelNode, inhMeta bool) (int, error) {
	v := newVisitor(he, w)
	if !he.xhtml {
		v.b.WriteString("<!DOCTYPE html>\n")
	}
	v.b.WriteStrings("<html lang=\"", he.lang, "\">\n<head>\n<meta charset=\"utf-8\">\n")
	textEnc := encoder.Create("text")
	var sb strings.Builder
	textEnc.WriteInlines(&sb, zn.Title)
	v.b.WriteStrings("<title>", sb.String(), "</title>")
	if inhMeta {
		v.acceptMeta(zn.InhMeta, false)
	} else {
		v.acceptMeta(zn.Zettel.Meta, false)
	}
	v.b.WriteString("\n</head>\n<body>\n")
	v.acceptBlockSlice(zn.Ast)
	v.writeEndnotes()
	v.b.WriteString("</body>\n</html>")
	length, err := v.b.Flush()
	return length, err
}

// WriteMeta encodes meta data as HTML5.
func (he *htmlEncoder) WriteMeta(w io.Writer, meta *domain.Meta) (int, error) {
	v := newVisitor(he, w)
	v.acceptMeta(meta, true)
	length, err := v.b.Flush()
	return length, err
}

func (he *htmlEncoder) WriteContent(w io.Writer, zn *ast.ZettelNode) (int, error) {
	return he.WriteBlocks(w, zn.Ast)
}

// WriteBlocks encodes a block slice.
func (he *htmlEncoder) WriteBlocks(w io.Writer, bs ast.BlockSlice) (int, error) {
	v := newVisitor(he, w)
	v.acceptBlockSlice(bs)
	v.writeEndnotes()
	length, err := v.b.Flush()
	return length, err
}

// WriteInlines writes an inline slice to the writer
func (he *htmlEncoder) WriteInlines(w io.Writer, is ast.InlineSlice) (int, error) {
	v := newVisitor(he, w)
	v.acceptInlineSlice(is)
	length, err := v.b.Flush()
	return length, err
}
