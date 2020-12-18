//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package meta provides the domain specific type 'meta'.
package meta

import (
	"strings"
	"testing"

	"zettelstore.de/z/domain/id"
)

const testID = id.Zid(98765432101234)

func newMeta(title string, tags []string, syntax string) *Meta {
	m := New(testID)
	if title != "" {
		m.Set(KeyTitle, title)
	}
	if tags != nil {
		m.Set(KeyTags, strings.Join(tags, " "))
	}
	if syntax != "" {
		m.Set(KeySyntax, syntax)
	}
	return m
}

func TestKeyIsValid(t *testing.T) {
	validKeys := []string{"0", "a", "0-", "title", "title-----", strings.Repeat("r", 255)}
	for _, key := range validKeys {
		if !KeyIsValid(key) {
			t.Errorf("Key %q wrongly identified as invalid key", key)
		}
	}
	invalidKeys := []string{"", "-", "-a", "Title", "a_b", strings.Repeat("e", 256)}
	for _, key := range invalidKeys {
		if KeyIsValid(key) {
			t.Errorf("Key %q wrongly identified as valid key", key)
		}
	}
}

func TestTitleHeader(t *testing.T) {
	m := New(testID)
	if got, ok := m.Get(KeyTitle); ok || got != "" {
		t.Errorf("Title is not empty, but %q", got)
	}
	addToMeta(m, KeyTitle, " ")
	if got, ok := m.Get(KeyTitle); ok || got != "" {
		t.Errorf("Title is not empty, but %q", got)
	}
	const st = "A simple text"
	addToMeta(m, KeyTitle, " "+st+"  ")
	if got, ok := m.Get(KeyTitle); !ok || got != st {
		t.Errorf("Title is not %q, but %q", st, got)
	}
	addToMeta(m, KeyTitle, "  "+st+"\t")
	const exp = st + " " + st
	if got, ok := m.Get(KeyTitle); !ok || got != exp {
		t.Errorf("Title is not %q, but %q", exp, got)
	}

	m = New(testID)
	const at = "A Title"
	addToMeta(m, KeyTitle, at)
	addToMeta(m, KeyTitle, " ")
	if got, ok := m.Get(KeyTitle); !ok || got != at {
		t.Errorf("Title is not %q, but %q", at, got)
	}
}

func checkSet(t *testing.T, exp []string, m *Meta, key string) {
	t.Helper()
	got, _ := m.GetList(key)
	for i, tag := range exp {
		if i < len(got) {
			if tag != got[i] {
				t.Errorf("Pos=%d, expected %q, got %q", i, exp[i], got[i])
			}
		} else {
			t.Errorf("Expected %q, but is missing", exp[i])
		}
	}
	if len(exp) < len(got) {
		t.Errorf("Extra tags: %q", got[len(exp):])
	}
}

func TestTagsHeader(t *testing.T) {
	m := New(testID)
	checkSet(t, []string{}, m, KeyTags)

	addToMeta(m, KeyTags, "")
	checkSet(t, []string{}, m, KeyTags)

	addToMeta(m, KeyTags, "  #t1 #t2  #t3 #t4  ")
	checkSet(t, []string{"#t1", "#t2", "#t3", "#t4"}, m, KeyTags)

	addToMeta(m, KeyTags, "#t5")
	checkSet(t, []string{"#t1", "#t2", "#t3", "#t4", "#t5"}, m, KeyTags)

	addToMeta(m, KeyTags, "t6")
	checkSet(t, []string{"#t1", "#t2", "#t3", "#t4", "#t5"}, m, KeyTags)
}

func TestSyntax(t *testing.T) {
	m := New(testID)
	if got, ok := m.Get(KeySyntax); ok || got != "" {
		t.Errorf("Syntax is not %q, but %q", "", got)
	}
	addToMeta(m, KeySyntax, " ")
	if got, _ := m.Get(KeySyntax); got != "" {
		t.Errorf("Syntax is not %q, but %q", "", got)
	}
	addToMeta(m, KeySyntax, "MarkDown")
	const exp = "markdown"
	if got, ok := m.Get(KeySyntax); !ok || got != exp {
		t.Errorf("Syntax is not %q, but %q", exp, got)
	}
	addToMeta(m, KeySyntax, " ")
	if got, _ := m.Get(KeySyntax); got != "" {
		t.Errorf("Syntax is not %q, but %q", "", got)
	}
}

func checkHeader(t *testing.T, exp map[string]string, gotP []Pair) {
	t.Helper()
	got := make(map[string]string, len(gotP))
	for _, p := range gotP {
		got[p.Key] = p.Value
		if _, ok := exp[p.Key]; !ok {
			t.Errorf("Key %q is not expected, but has value %q", p.Key, p.Value)
		}
	}
	for k, v := range exp {
		if gv, ok := got[k]; !ok || v != gv {
			if ok {
				t.Errorf("Key %q is not %q, but %q", k, v, got[k])
			} else {
				t.Errorf("Key %q missing, should have value %q", k, v)
			}
		}
	}
}

func TestDefaultHeader(t *testing.T) {
	m := New(testID)
	addToMeta(m, "h1", "d1")
	addToMeta(m, "H2", "D2")
	addToMeta(m, "H1", "D1.1")
	exp := map[string]string{"h1": "d1 D1.1", "h2": "D2"}
	checkHeader(t, exp, m.Pairs())
	addToMeta(m, "", "d0")
	checkHeader(t, exp, m.Pairs())
	addToMeta(m, "h3", "")
	exp["h3"] = ""
	checkHeader(t, exp, m.Pairs())
	addToMeta(m, "h3", "  ")
	checkHeader(t, exp, m.Pairs())
	addToMeta(m, "h4", " ")
	exp["h4"] = ""
	checkHeader(t, exp, m.Pairs())
}

func TestDelete(t *testing.T) {
	m := New(testID)
	m.Set("key", "val")
	if got, ok := m.Get("key"); !ok || got != "val" {
		t.Errorf("Value != %q, got: %v/%q", "val", ok, got)
	}
	m.Set("key", "")
	if got, ok := m.Get("key"); !ok || got != "" {
		t.Errorf("Value != %q, got: %v/%q", "", ok, got)
	}
	m.Delete("key")
	if got, ok := m.Get("key"); ok || got != "" {
		t.Errorf("Value != %q, got: %v/%q", "", ok, got)
	}
}
