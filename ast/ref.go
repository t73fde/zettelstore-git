//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package ast provides the abstract syntax tree.
package ast

import (
	"net/url"

	"zettelstore.de/z/domain/id"
)

// ParseReference parses a string and returns a reference.
func ParseReference(s string) *Reference {
	if len(s) == 0 {
		return &Reference{URL: nil, Value: s, State: RefStateInvalid}
	}
	u, err := url.Parse(s)
	if err != nil {
		return &Reference{URL: nil, Value: s, State: RefStateInvalid}
	}
	if len(u.Scheme)+len(u.Opaque)+len(u.Host) == 0 && u.User == nil {
		if _, err := id.Parse(u.Path); err == nil {
			return &Reference{URL: u, Value: s, State: RefStateZettel}
		}
		if u.Path == "" && u.Fragment != "" {
			return &Reference{URL: u, Value: s, State: RefStateZettelSelf}
		}
		if isLocalPath(u.Path) {
			return &Reference{URL: u, Value: s, State: RefStateLocal}
		}
	}
	return &Reference{URL: u, Value: s, State: RefStateExternal}
}

func isLocalPath(path string) bool {
	if len(path) > 0 && path[0] == '/' {
		return true
	}
	if len(path) > 1 && path[0] == '.' {
		if len(path) > 2 && path[1] == '.' && path[2] == '/' {
			return true
		}
		return path[1] == '/'
	}
	return false
}

// String returns the string representation of a reference.
func (r Reference) String() string {
	if r.URL != nil {
		return r.URL.String()
	}
	return r.Value
}

// IsValid returns true if reference is valid
func (r *Reference) IsValid() bool { return r.State != RefStateInvalid }

// IsZettel returns true if it is a referencen to a local zettel.
func (r *Reference) IsZettel() bool {
	switch r.State {
	case RefStateZettel, RefStateZettelSelf, RefStateZettelFound, RefStateZettelBroken:
		return true
	}
	return false
}

// IsLocal returns true if reference is local
func (r *Reference) IsLocal() bool { return r.State == RefStateLocal }

// IsExternal returns true if it is a referencen to external material.
func (r *Reference) IsExternal() bool { return r.State == RefStateExternal }
