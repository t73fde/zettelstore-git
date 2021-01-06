//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package memplace stores zettel volatile in main memory.
package memplace

import (
	"context"
	"net/url"
	"sync"
	"time"

	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/place"
	"zettelstore.de/z/place/manager"
)

func init() {
	manager.Register(
		"mem",
		func(u *url.URL, next place.Place) (place.Place, error) {
			return &memPlace{u: u, next: next}, nil
		})
}

type memPlace struct {
	u         *url.URL
	next      place.Place
	zettel    map[id.Zid]domain.Zettel
	mx        sync.RWMutex
	observers []place.ObserverFunc
}

func (mp *memPlace) notifyChanged(reason place.ChangeReason, zid id.Zid) {
	for _, ob := range mp.observers {
		ob(reason, zid)
	}
}

func (mp *memPlace) Next() place.Place { return nil }

func (mp *memPlace) Location() string {
	return mp.u.String()
}

func (mp *memPlace) Start(ctx context.Context) error {
	mp.mx.Lock()
	defer mp.mx.Unlock()
	mp.zettel = make(map[id.Zid]domain.Zettel)
	return nil
}

func (mp *memPlace) Stop(ctx context.Context) error {
	mp.mx.Lock()
	defer mp.mx.Unlock()
	mp.zettel = nil
	return nil
}

func (mp *memPlace) RegisterChangeObserver(f place.ObserverFunc) {
	mp.mx.Lock()
	mp.observers = append(mp.observers, f)
	mp.mx.Unlock()
}

func (mp *memPlace) CanCreateZettel(ctx context.Context) bool { return true }

func (mp *memPlace) CreateZettel(ctx context.Context, zettel domain.Zettel) (id.Zid, error) {
	mp.mx.Lock()
	defer mp.mx.Unlock()

	meta := zettel.Meta.Clone()
	meta.Zid = mp.calcNewZid()
	zettel.Meta = meta
	mp.zettel[meta.Zid] = zettel
	mp.notifyChanged(place.OnCreate, meta.Zid)
	return meta.Zid, nil
}

func (mp *memPlace) calcNewZid() id.Zid {
	zid := id.New(false)
	if _, ok := mp.zettel[zid]; !ok {
		return zid
	}
	for {
		zid = id.New(true)
		if _, ok := mp.zettel[zid]; !ok {
			return zid
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (mp *memPlace) GetZettel(ctx context.Context, zid id.Zid) (domain.Zettel, error) {
	mp.mx.RLock()
	zettel, ok := mp.zettel[zid]
	mp.mx.RUnlock()
	if !ok {
		return domain.Zettel{}, place.ErrNotFound
	}
	return zettel, nil
}

func (mp *memPlace) GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error) {
	mp.mx.RLock()
	zettel, ok := mp.zettel[zid]
	mp.mx.RUnlock()
	if !ok {
		return nil, place.ErrNotFound
	}
	return zettel.Meta, nil
}

func (mp *memPlace) SelectMeta(
	ctx context.Context, f *place.Filter, s *place.Sorter) ([]*meta.Meta, error) {
	mp.mx.RLock()
	filterFunc := place.CreateFilterFunc(f)
	result := make([]*meta.Meta, 0)
	for _, zettel := range mp.zettel {
		if filterFunc(zettel.Meta) {
			result = append(result, zettel.Meta)
		}
	}
	mp.mx.RUnlock()
	if mp.next != nil {
		other, err := mp.next.SelectMeta(ctx, f, nil)
		if err != nil {
			return nil, err
		}
		return place.MergeSorted(place.ApplySorter(result, nil), other, s), nil
	}
	return place.ApplySorter(result, s), nil
}

func (mp *memPlace) CanUpdateZettel(ctx context.Context, zettel domain.Zettel) bool {
	return true
}

func (mp *memPlace) UpdateZettel(ctx context.Context, zettel domain.Zettel) error {
	mp.mx.Lock()
	defer mp.mx.Unlock()

	meta := zettel.Meta.Clone()
	if !meta.Zid.IsValid() {
		return &place.ErrInvalidID{Zid: meta.Zid}
	}
	zettel.Meta = meta
	mp.zettel[meta.Zid] = zettel
	mp.notifyChanged(place.OnUpdate, meta.Zid)
	return nil
}

func (mp *memPlace) CanRenameZettel(ctx context.Context, zid id.Zid) bool { return true }

func (mp *memPlace) RenameZettel(ctx context.Context, curZid, newZid id.Zid) error {
	mp.mx.Lock()
	defer mp.mx.Unlock()

	zettel, ok := mp.zettel[curZid]
	if !ok {
		if mp.next != nil {
			return mp.next.RenameZettel(ctx, curZid, newZid)
		}
		return nil
	}

	// Check that there is no zettel with newZid, neither local nor in the next place
	if _, ok = mp.zettel[newZid]; ok {
		return &place.ErrInvalidID{Zid: newZid}
	}
	if mp.next != nil {
		if _, err := mp.next.GetMeta(ctx, newZid); err == nil {
			return &place.ErrInvalidID{Zid: newZid}
		}
	}

	meta := zettel.Meta.Clone()
	meta.Zid = newZid
	zettel.Meta = meta
	mp.zettel[newZid] = zettel
	delete(mp.zettel, curZid)
	mp.notifyChanged(place.OnDelete, curZid)
	mp.notifyChanged(place.OnCreate, newZid)
	return nil
}

func (mp *memPlace) CanDeleteZettel(ctx context.Context, zid id.Zid) bool {
	mp.mx.RLock()
	_, ok := mp.zettel[zid]
	mp.mx.Unlock()
	return ok
}

func (mp *memPlace) DeleteZettel(ctx context.Context, zid id.Zid) error {
	mp.mx.Lock()
	defer mp.mx.Unlock()

	if _, ok := mp.zettel[zid]; !ok {
		return place.ErrNotFound
	}
	delete(mp.zettel, zid)
	mp.notifyChanged(place.OnDelete, zid)
	return nil
}

func (mp *memPlace) Reload(ctx context.Context) error { return nil }
