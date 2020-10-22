//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is free software: you can redistribute it and/or modify it under
// the terms of the GNU Affero General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option)
// any later version.
//
// Zettelstore is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
// FITNESS FOR A PARTICULAR PURPOSE. See the GNU Affero General Public License
// for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with Zettelstore. If not, see <http://www.gnu.org/licenses/>.
//-----------------------------------------------------------------------------

// Package usecase provides (business) use cases for the zettelstore.
package usecase

import (
	"context"

	"zettelstore.de/z/domain"
)

// DeleteZettelPort is the interface used by this use case.
type DeleteZettelPort interface {
	// DeleteZettel removes the zettel from the place.
	DeleteZettel(ctx context.Context, zid domain.ZettelID) error
}

// DeleteZettel is the data for this use case.
type DeleteZettel struct {
	port DeleteZettelPort
}

// NewDeleteZettel creates a new use case.
func NewDeleteZettel(port DeleteZettelPort) DeleteZettel {
	return DeleteZettel{port: port}
}

// Run executes the use case.
func (uc DeleteZettel) Run(ctx context.Context, zid domain.ZettelID) error {
	return uc.port.DeleteZettel(ctx, zid)
}
