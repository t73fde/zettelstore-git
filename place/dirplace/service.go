//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package dirplace provides a directory-based zettel place.
package dirplace

import (
	"io/ioutil"
	"os"

	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/input"
	"zettelstore.de/z/place/dirplace/directory"
)

func fileService(num uint32, cmds <-chan fileCmd) {
	for cmd := range cmds {
		cmd.run()
	}
}

type fileCmd interface {
	run()
}

// COMMAND: getMeta ----------------------------------------
//
// Retrieves the meta data from a zettel.

type fileGetMeta struct {
	entry *directory.Entry
	rc    chan<- resGetMeta
}
type resGetMeta struct {
	meta *meta.Meta
	err  error
}

func (cmd *fileGetMeta) run() {
	var m *meta.Meta
	var err error
	switch cmd.entry.MetaSpec {
	case directory.MetaSpecFile:
		m, err = parseMetaFile(cmd.entry.Zid, cmd.entry.MetaPath)
	case directory.MetaSpecHeader:
		m, _, err = parseMetaContentFile(cmd.entry.Zid, cmd.entry.ContentPath)
	default:
		m = cmd.entry.CalcDefaultMeta()
	}
	if err == nil {
		cleanupMeta(m, cmd.entry)
	}
	cmd.rc <- resGetMeta{m, err}
}

// COMMAND: getMetaContent ----------------------------------------
//
// Retrieves the meta data and the content of a zettel.

type fileGetMetaContent struct {
	entry *directory.Entry
	rc    chan<- resGetMetaContent
}
type resGetMetaContent struct {
	meta    *meta.Meta
	content string
	err     error
}

func (cmd *fileGetMetaContent) run() {
	var m *meta.Meta
	var content string
	var err error

	switch cmd.entry.MetaSpec {
	case directory.MetaSpecFile:
		m, err = parseMetaFile(cmd.entry.Zid, cmd.entry.MetaPath)
		content, err = readFileContent(cmd.entry.ContentPath)
	case directory.MetaSpecHeader:
		m, content, err = parseMetaContentFile(cmd.entry.Zid, cmd.entry.ContentPath)
	default:
		m = cmd.entry.CalcDefaultMeta()
		content, err = readFileContent(cmd.entry.ContentPath)
	}
	if err == nil {
		cleanupMeta(m, cmd.entry)
	}
	cmd.rc <- resGetMetaContent{m, content, err}
}

// COMMAND: setZettel ----------------------------------------
//
// Writes a new or exsting zettel.

type fileSetZettel struct {
	entry  *directory.Entry
	zettel domain.Zettel
	rc     chan<- resSetZettel
}
type resSetZettel = error

func (cmd *fileSetZettel) run() {
	var f *os.File
	var err error

	switch cmd.entry.MetaSpec {
	case directory.MetaSpecFile:
		f, err = openFileWrite(cmd.entry.MetaPath)
		if err == nil {
			_, err = cmd.zettel.Meta.Write(f)
			if err1 := f.Close(); err == nil {
				err = err1
			}

			if err == nil {
				err = writeFileContent(cmd.entry.ContentPath, cmd.zettel.Content.AsString())
			}
		}

	case directory.MetaSpecHeader:
		f, err = openFileWrite(cmd.entry.ContentPath)
		if err == nil {
			_, err = cmd.zettel.Meta.WriteAsHeader(f)
			if err == nil {
				_, err = f.WriteString(cmd.zettel.Content.AsString())
				if err1 := f.Close(); err == nil {
					err = err1
				}
			}
		}

	case directory.MetaSpecNone:
		// TODO: if meta has some additional infos: write meta to new .meta;
		// update entry in dir

		err = writeFileContent(cmd.entry.ContentPath, cmd.zettel.Content.AsString())

	case directory.MetaSpecUnknown:
		panic("TODO: ???")
	}
	cmd.rc <- err
}

// COMMAND: renameZettel ----------------------------------------
//
// Gives an existing zettel a new id.

type fileRenameZettel struct {
	curEntry *directory.Entry
	newEntry *directory.Entry
	rc       chan<- resRenameZettel
}

type resRenameZettel = error

func (cmd *fileRenameZettel) run() {
	var err error

	switch cmd.curEntry.MetaSpec {
	case directory.MetaSpecFile:
		err1 := os.Rename(cmd.curEntry.MetaPath, cmd.newEntry.MetaPath)
		err = os.Rename(cmd.curEntry.ContentPath, cmd.newEntry.ContentPath)
		if err == nil {
			err = err1
		}
	case directory.MetaSpecHeader, directory.MetaSpecNone:
		err = os.Rename(cmd.curEntry.ContentPath, cmd.newEntry.ContentPath)
	case directory.MetaSpecUnknown:
		panic("TODO: ???")
	}
	cmd.rc <- err
}

// COMMAND: deleteZettel ----------------------------------------
//
// Deletes an existing zettel.

type fileDeleteZettel struct {
	entry *directory.Entry
	rc    chan<- resDeleteZettel
}
type resDeleteZettel = error

func (cmd *fileDeleteZettel) run() {
	var err error

	switch cmd.entry.MetaSpec {
	case directory.MetaSpecFile:
		err1 := os.Remove(cmd.entry.MetaPath)
		err = os.Remove(cmd.entry.ContentPath)
		if err == nil {
			err = err1
		}
	case directory.MetaSpecHeader:
		err = os.Remove(cmd.entry.ContentPath)
	case directory.MetaSpecNone:
		err = os.Remove(cmd.entry.ContentPath)
	case directory.MetaSpecUnknown:
		panic("TODO: ???")
	}
	cmd.rc <- err
}

// Utility functions ----------------------------------------

func readFileContent(path string) (string, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func parseMetaFile(zid id.Zid, path string) (*meta.Meta, error) {
	src, err := readFileContent(path)
	if err != nil {
		return nil, err
	}
	inp := input.NewInput(src)
	return meta.NewFromInput(zid, inp), nil
}

func parseMetaContentFile(zid id.Zid, path string) (*meta.Meta, string, error) {
	src, err := readFileContent(path)
	if err != nil {
		return nil, "", err
	}
	inp := input.NewInput(src)
	meta := meta.NewFromInput(zid, inp)
	return meta, src[inp.Pos:], nil
}

func cleanupMeta(m *meta.Meta, entry *directory.Entry) {
	if title, ok := m.Get(meta.KeyTitle); !ok || title == "" {
		m.Set(meta.KeyTitle, entry.Zid.Format())
	}

	switch entry.MetaSpec {
	case directory.MetaSpecFile:
		if syntax, ok := m.Get(meta.KeySyntax); !ok || syntax == "" {
			dm := entry.CalcDefaultMeta()
			syntax, ok = dm.Get(meta.KeySyntax)
			if !ok {
				panic("Default meta must contain syntax")
			}
			m.Set(meta.KeySyntax, syntax)
		}
	}

	if entry.Duplicates {
		m.Set(meta.KeyDuplicates, meta.ValueTrue)
	}
}

func openFileWrite(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
}

func writeFileContent(path string, content string) error {
	f, err := openFileWrite(path)
	if err == nil {
		_, err = f.WriteString(content)
		if err1 := f.Close(); err == nil {
			err = err1
		}
	}
	return err
}
