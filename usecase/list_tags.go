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
	"context"

	"zettelstore.de/z/domain"
	"zettelstore.de/z/place"
)

// ListTagsPort is the interface used by this use case.
type ListTagsPort interface {
	// SelectMeta returns all zettel meta data that match the selection
	// criteria. The result is ordered by descending zettel id.
	SelectMeta(ctx context.Context, f *place.Filter, s *place.Sorter) ([]*domain.Meta, error)
}

// ListTags is the data for this use case.
type ListTags struct {
	port ListTagsPort
}

// NewListTags creates a new use case.
func NewListTags(port ListTagsPort) ListTags {
	return ListTags{port: port}
}

// TagData associates tags with a list of all zettel meta that use this tag
type TagData map[string][]*domain.Meta

// Run executes the use case.
func (uc ListTags) Run(ctx context.Context, minCount int) (TagData, error) {
	metas, err := uc.port.SelectMeta(ctx, nil, nil)
	if err != nil {
		return nil, err
	}
	result := make(TagData)
	for _, meta := range metas {
		if tl, ok := meta.GetList(domain.MetaKeyTags); ok && len(tl) > 0 {
			for _, t := range tl {
				result[t] = append(result[t], meta)
			}
		}
	}
	if minCount > 1 {
		for t, ms := range result {
			if len(ms) < minCount {
				delete(result, t)
			}
		}
	}
	return result, nil
}
