//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package api provides api handlers for web requests.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"zettelstore.de/z/ast"
	"zettelstore.de/z/config/runtime"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

type jsonIDURL struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}
type jsonZettel struct {
	ID       string            `json:"id"`
	URL      string            `json:"url"`
	Meta     map[string]string `json:"meta"`
	Encoding string            `json:"encoding"`
	Content  interface{}       `json:"content"`
}
type jsonMeta struct {
	ID   string            `json:"id"`
	URL  string            `json:"url"`
	Meta map[string]string `json:"meta"`
}
type jsonContent struct {
	ID       string      `json:"id"`
	URL      string      `json:"url"`
	Encoding string      `json:"encoding"`
	Content  interface{} `json:"content"`
}

func writeJSONZettel(w http.ResponseWriter, z *ast.ZettelNode, part string) error {
	var outData interface{}
	idData := jsonIDURL{
		ID:  z.Zid.Format(),
		URL: adapter.NewURLBuilder('z').SetZid(z.Zid).String(),
	}

	switch part {
	case "zettel":
		encoding, content := encodedContent(z.Zettel.Content)
		outData = jsonZettel{
			ID:       idData.ID,
			URL:      idData.URL,
			Meta:     z.InhMeta.Map(),
			Encoding: encoding,
			Content:  content,
		}
	case "meta":
		outData = jsonMeta{
			ID:   idData.ID,
			URL:  idData.URL,
			Meta: z.InhMeta.Map(),
		}
	case "content":
		encoding, content := encodedContent(z.Zettel.Content)
		outData = jsonContent{
			ID:       idData.ID,
			URL:      idData.URL,
			Encoding: encoding,
			Content:  content,
		}
	case "id":
		outData = idData
	default:
		panic(part)
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(outData)
}

func encodedContent(content domain.Content) (string, interface{}) {
	if content.IsBinary() {
		return "base64", content.AsBytes()
	}
	return "", content.AsString()
}

func writeDJSONZettel(
	ctx context.Context,
	w http.ResponseWriter,
	z *ast.ZettelNode,
	part string,
	getMeta usecase.GetMeta,
) (err error) {
	switch part {
	case "zettel":
		err = writeDJSONHeader(w, z.Zid)
		if err == nil {
			err = writeDJSONMeta(w, z)
		}
		if err == nil {
			err = writeDJSONContent(ctx, w, z, part, getMeta)
		}
	case "meta":
		err = writeDJSONHeader(w, z.Zid)
		if err == nil {
			err = writeDJSONMeta(w, z)
		}
	case "content":
		err = writeDJSONHeader(w, z.Zid)
		if err == nil {
			err = writeDJSONContent(ctx, w, z, part, getMeta)
		}
	case "id":
		writeDJSONHeader(w, z.Zid)
	default:
		panic(part)
	}
	if err == nil {
		_, err = w.Write(djsonFooter)
	}
	return err
}

var (
	djsonMetaHeader    = []byte(",\"meta\":")
	djsonContentHeader = []byte(",\"content\":")
	djsonHeader1       = []byte("{\"id\":\"")
	djsonHeader2       = []byte("\",\"url\":\"")
	djsonHeader3       = []byte("?_format=")
	djsonHeader4       = []byte("\"")
	djsonFooter        = []byte("}")
)

func writeDJSONHeader(w http.ResponseWriter, zid id.ZettelID) error {
	_, err := w.Write(djsonHeader1)
	if err == nil {
		_, err = w.Write(zid.FormatBytes())
	}
	if err == nil {
		_, err = w.Write(djsonHeader2)
	}
	if err == nil {
		_, err = w.Write([]byte(adapter.NewURLBuilder('z').SetZid(zid).String()))
	}
	if err == nil {
		_, err = w.Write(djsonHeader3)
		if err == nil {
			_, err = w.Write([]byte("djson"))
		}
	}
	if err == nil {
		_, err = w.Write(djsonHeader4)
	}
	return err
}

func writeDJSONMeta(w io.Writer, z *ast.ZettelNode) error {
	_, err := w.Write(djsonMetaHeader)
	if err == nil {
		err = writeMeta(w, z.InhMeta, "djson", &encoder.TitleOption{Inline: z.Title})
	}
	return err
}

func writeDJSONContent(
	ctx context.Context,
	w io.Writer,
	z *ast.ZettelNode,
	part string,
	getMeta usecase.GetMeta,
) (err error) {
	_, err = w.Write(djsonContentHeader)
	if err == nil {
		err = writeContent(w, z, "djson",
			&encoder.AdaptLinkOption{
				Adapter: adapter.MakeLinkAdapter(ctx, 'z', getMeta, part, "djson"),
			},
			&encoder.AdaptImageOption{Adapter: adapter.MakeImageAdapter()},
		)
	}
	return err
}

var (
	jsonListHeader = []byte("{\"list\":[")
	jsonListSep    = []byte{','}
	jsonListFooter = []byte("]}")
)

var setJSON = map[string]bool{"json": true}

func renderListMetaXJSON(
	ctx context.Context,
	w http.ResponseWriter,
	metaList []*meta.Meta,
	format string, part string,
	getMeta usecase.GetMeta,
	parseZettel usecase.ParseZettel,
) {
	var readZettel bool
	switch part {
	case "zettel", "content":
		readZettel = true
	case "meta", "id":
		readZettel = false
	default:
		http.Error(w, fmt.Sprintf("Unknown _part=%v parameter", part), http.StatusBadRequest)
		return
	}
	isJSON := setJSON[format]
	_, err := w.Write(jsonListHeader)
	for i, m := range metaList {
		if err != nil {
			break
		}
		if i > 0 {
			_, err = w.Write(jsonListSep)
		}
		if err != nil {
			break
		}
		var zn *ast.ZettelNode
		if readZettel {
			z, err1 := parseZettel.Run(ctx, m.Zid, "")
			if err1 != nil {
				err = err1
				break
			}
			zn = z
		} else {
			zn = &ast.ZettelNode{
				Zettel:  domain.Zettel{Meta: m, Content: ""},
				Zid:     m.Zid,
				InhMeta: runtime.AddDefaultValues(m),
				Title: parser.ParseTitle(
					m.GetDefault(meta.KeyTitle, runtime.GetDefaultTitle())),
				Ast: nil,
			}
		}
		if isJSON {
			err = writeJSONZettel(w, zn, part)
		} else {
			err = writeDJSONZettel(ctx, w, zn, part, getMeta)
		}
	}
	if err == nil {
		_, err = w.Write(jsonListFooter)
	}
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		log.Println(err)
	}
}

func writeContent(
	w io.Writer, zn *ast.ZettelNode, format string, options ...encoder.Option) error {
	enc := encoder.Create(format, options...)
	if enc == nil {
		return adapter.ErrNoSuchFormat
	}

	_, err := enc.WriteContent(w, zn)
	return err
}

func writeMeta(
	w io.Writer, m *meta.Meta, format string, options ...encoder.Option) error {
	enc := encoder.Create(format, options...)
	if enc == nil {
		return adapter.ErrNoSuchFormat
	}

	_, err := enc.WriteMeta(w, m)
	return err
}
