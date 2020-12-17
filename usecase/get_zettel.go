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
	"zettelstore.de/z/domain/id"
)

// GetZettelPort is the interface used by this use case.
type GetZettelPort interface {
	// GetZettel retrieves a specific zettel.
	GetZettel(ctx context.Context, zid id.ZettelID) (domain.Zettel, error)
}

// GetZettel is the data for this use case.
type GetZettel struct {
	port GetZettelPort
}

// NewGetZettel creates a new use case.
func NewGetZettel(port GetZettelPort) GetZettel {
	return GetZettel{port: port}
}

// Run executes the use case.
func (uc GetZettel) Run(ctx context.Context, zid id.ZettelID) (domain.Zettel, error) {
	return uc.port.GetZettel(ctx, zid)
}
