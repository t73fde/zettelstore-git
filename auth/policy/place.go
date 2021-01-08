//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package policy provides some interfaces and implementation for authorizsation policies.
package policy

import (
	"context"

	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/place"
	"zettelstore.de/z/web/session"
)

// PlaceWithPolicy wraps the given place inside a policy place.
func PlaceWithPolicy(
	place place.Place,
	simpleMode bool,
	withAuth func() bool,
	isReadOnlyMode bool,
	expertMode func() bool,
	isOwner func(id.Zid) bool,
	getVisibility func(*meta.Meta) meta.Visibility,
) (place.Place, Policy) {
	pol := newPolicy(simpleMode, withAuth, isReadOnlyMode, expertMode, isOwner, getVisibility)
	return newPlace(place, pol), pol
}

// polPlace implements a policy place.
type polPlace struct {
	place  place.Place
	policy Policy
}

// newPlace creates a new policy place.
func newPlace(place place.Place, policy Policy) place.Place {
	return &polPlace{
		place:  place,
		policy: policy,
	}
}

func (pp *polPlace) Location() string {
	return pp.place.Location()
}

// Start the place. Now all other functions of the place are allowed.
// Starting an already started place is not allowed.
func (pp *polPlace) Start(ctx context.Context) error {
	return pp.place.Start(ctx)
}

// Stop the started place. Now only the Start() function is allowed.
func (pp *polPlace) Stop(ctx context.Context) error {
	return pp.place.Stop(ctx)
}

// RegisterChangeObserver registers an observer that will be notified
// if a zettel was found to be changed.
func (pp *polPlace) RegisterChangeObserver(f place.ObserverFunc) {
	pp.place.RegisterChangeObserver(f)
}

func (pp *polPlace) CanCreateZettel(ctx context.Context) bool {
	return pp.place.CanCreateZettel(ctx)
}

func (pp *polPlace) CreateZettel(
	ctx context.Context, zettel domain.Zettel) (id.Zid, error) {
	user := session.GetUser(ctx)
	if pp.policy.CanCreate(user, zettel.Meta) {
		return pp.place.CreateZettel(ctx, zettel)
	}
	return id.Invalid, place.NewErrNotAllowed("Create", user, id.Invalid)
}

func (pp *polPlace) GetZettel(ctx context.Context, zid id.Zid) (domain.Zettel, error) {
	zettel, err := pp.place.GetZettel(ctx, zid)
	if err != nil {
		return domain.Zettel{}, err
	}
	user := session.GetUser(ctx)
	if pp.policy.CanRead(user, zettel.Meta) {
		return zettel, nil
	}
	return domain.Zettel{}, place.NewErrNotAllowed("GetZettel", user, zid)
}

// GetMeta retrieves just the meta data of a specific zettel.
func (pp *polPlace) GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error) {
	m, err := pp.place.GetMeta(ctx, zid)
	if err != nil {
		return nil, err
	}
	user := session.GetUser(ctx)
	if pp.policy.CanRead(user, m) {
		return m, nil
	}
	return nil, place.NewErrNotAllowed("GetMeta", user, zid)
}

// SelectMeta returns all zettel meta data that match the selection
// criteria. The result is ordered by descending zettel id.
func (pp *polPlace) SelectMeta(
	ctx context.Context, f *place.Filter, s *place.Sorter) ([]*meta.Meta, error) {
	user := session.GetUser(ctx)
	f = place.EnsureFilter(f)
	canRead := pp.policy.CanRead
	if sel := f.Select; sel != nil {
		f.Select = func(m *meta.Meta) bool {
			return canRead(user, m) && sel(m)
		}
	} else {
		f.Select = func(m *meta.Meta) bool {
			return canRead(user, m)
		}
	}
	result, err := pp.place.SelectMeta(ctx, f, s)
	return result, err
}

func (pp *polPlace) CanUpdateZettel(ctx context.Context, zettel domain.Zettel) bool {
	return pp.place.CanUpdateZettel(ctx, zettel)
}

func (pp *polPlace) UpdateZettel(ctx context.Context, zettel domain.Zettel) error {
	zid := zettel.Meta.Zid
	user := session.GetUser(ctx)
	if !zid.IsValid() {
		return &place.ErrInvalidID{Zid: zid}
	}
	// Write existing zettel
	oldMeta, err := pp.place.GetMeta(ctx, zid)
	if err != nil {
		return err
	}
	if pp.policy.CanWrite(user, oldMeta, zettel.Meta) {
		return pp.place.UpdateZettel(ctx, zettel)
	}
	return place.NewErrNotAllowed("Write", user, zid)
}

func (pp *polPlace) AllowRenameZettel(ctx context.Context, zid id.Zid) bool {
	return pp.place.AllowRenameZettel(ctx, zid)
}

// Rename changes the current zid to a new zid.
func (pp *polPlace) RenameZettel(ctx context.Context, curZid, newZid id.Zid) error {
	meta, err := pp.place.GetMeta(ctx, curZid)
	if err != nil {
		return err
	}
	user := session.GetUser(ctx)
	if pp.policy.CanRename(user, meta) {
		return pp.place.RenameZettel(ctx, curZid, newZid)
	}
	return place.NewErrNotAllowed("Rename", user, curZid)
}

func (pp *polPlace) CanDeleteZettel(ctx context.Context, zid id.Zid) bool {
	return pp.place.CanDeleteZettel(ctx, zid)
}

// DeleteZettel removes the zettel from the place.
func (pp *polPlace) DeleteZettel(ctx context.Context, zid id.Zid) error {
	meta, err := pp.place.GetMeta(ctx, zid)
	if err != nil {
		return err
	}
	user := session.GetUser(ctx)
	if pp.policy.CanDelete(user, meta) {
		return pp.place.DeleteZettel(ctx, zid)
	}
	return place.NewErrNotAllowed("Delete", user, zid)
}

func (pp *polPlace) Reload(ctx context.Context) error {
	user := session.GetUser(ctx)
	if pp.policy.CanReload(user) {
		return pp.place.Reload(ctx)
	}
	return place.NewErrNotAllowed("Reload", user, id.Invalid)
}
func (pp *polPlace) ReadStats(st *place.Stats) {
	pp.place.ReadStats(st)
}
