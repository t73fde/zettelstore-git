//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package usecase provides (business) use cases for the zettelstore.
package usecase

import (
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/meta"
)

// NewZettel is the data for this use case.
type NewZettel struct{}

// NewNewZettel creates a new use case.
func NewNewZettel() NewZettel {
	return NewZettel{}
}

// Run executes the use case.
func (uc NewZettel) Run(origZettel domain.Zettel) domain.Zettel {
	m := origZettel.Meta.Clone()
	if role, ok := m.Get(meta.MetaKeyRole); ok && role == meta.MetaValueRoleNewTemplate {
		const prefix = "new-"
		for _, pair := range m.PairsRest() {
			if key := pair.Key; len(key) > len(prefix) && key[0:len(prefix)] == prefix {
				m.Set(key[len(prefix):], pair.Value)
				m.Delete(key)
			}
		}
	}
	return domain.Zettel{Meta: m, Content: origZettel.Content}
}
