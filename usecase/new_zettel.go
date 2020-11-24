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
)

// NewZettel is the data for this use case.
type NewZettel struct{}

// NewNewZettel creates a new use case.
func NewNewZettel() NewZettel {
	return NewZettel{}
}

// Run executes the use case.
func (uc NewZettel) Run(origZettel domain.Zettel) domain.Zettel {
	meta := origZettel.Meta.Clone()
	if role, ok := meta.Get(domain.MetaKeyRole); ok && role == domain.MetaValueRoleNewTemplate {
		const prefix = "new-"
		for _, pair := range meta.PairsRest() {
			if key := pair.Key; len(key) > len(prefix) && key[0:len(prefix)] == prefix {
				meta.Set(key[len(prefix):], pair.Value)
				meta.Delete(key)
			}
		}
	}
	return domain.Zettel{Meta: meta, Content: origZettel.Content}
}
