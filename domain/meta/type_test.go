//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package meta_test provides tests for the domain specific type 'meta'.
package meta_test

import (
	"strconv"
	"testing"
	"time"

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

func TestNow(t *testing.T) {
	m := meta.New(id.Invalid)
	m.SetNow("key")
	val, ok := m.Get("key")
	if !ok {
		t.Error("Unable to get value of key")
	}
	if len(val) != 14 {
		t.Errorf("Value is not 14 digits long: %q", val)
	}
	if _, err := strconv.ParseInt(val, 10, 64); err != nil {
		t.Errorf("Unable to parse %q as an int64: %v", val, err)
	}
	if _, ok := m.GetTime("key"); !ok {
		t.Errorf("Unable to get time from value %q", val)
	}
}

func TestGetTime(t *testing.T) {
	testCases := []struct {
		value string
		valid bool
		exp   time.Time
	}{
		{"", false, time.Time{}},
		{"1", false, time.Time{}},
		{"00000000000000", false, time.Time{}},
		{"98765432109876", false, time.Time{}},
		{"20201221111905", true, time.Date(2020, time.December, 21, 11, 19, 5, 0, time.UTC)},
	}
	for i, tc := range testCases {
		got, ok := meta.TimeValue(tc.value)
		if ok != tc.valid {
			t.Errorf("%d: parsing of %q should be %v, but got %v", i, tc.value, tc.valid, ok)
			continue
		}
		if got != tc.exp {
			t.Errorf("%d: parsing of %q should return %v, but got %v", i, tc.value, tc.exp, got)
		}
	}
}
