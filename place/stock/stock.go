//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package stock allows to get zettel without reading it from a place.
package stock

import (
	"context"
	"sync"

	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/place"
)

// Place is a place that is used by a stock.
type Place interface {
	// RegisterChangeObserver registers an observer that will be notified
	// if all or one zettel are found to be changed.
	RegisterChangeObserver(ob place.ObserverFunc)

	// GetZettel retrieves a specific zettel.
	GetZettel(ctx context.Context, zid id.Zid) (domain.Zettel, error)
}

// Stock allow to get subscribed zettel without reading it from a place.
type Stock interface {
	Subscribe(zid id.Zid) error
	GetZettel(zid id.Zid) domain.Zettel
	GetMeta(zid id.Zid) *meta.Meta
}

// NewStock creates a new stock that operates on the given place.
func NewStock(place Place) Stock {
	//RegisterChangeObserver(func(domain.Zid))
	stock := &defaultStock{
		place: place,
		subs:  make(map[id.Zid]domain.Zettel),
	}
	place.RegisterChangeObserver(stock.observe)
	return stock
}

type defaultStock struct {
	place  Place
	subs   map[id.Zid]domain.Zettel
	mxSubs sync.RWMutex
}

// observe tracks all changes the place signals.
func (s *defaultStock) observe(all bool, zid id.Zid) {
	if !all {
		s.mxSubs.RLock()
		defer s.mxSubs.RUnlock()
		if _, found := s.subs[zid]; found {
			go func() {
				s.mxSubs.Lock()
				defer s.mxSubs.Unlock()
				s.update(zid)
			}()
		}
		return
	}

	go func() {
		s.mxSubs.Lock()
		defer s.mxSubs.Unlock()
		for zid := range s.subs {
			s.update(zid)
		}
	}()
}

func (s *defaultStock) update(zid id.Zid) {
	if zettel, err := s.place.GetZettel(context.Background(), zid); err == nil {
		s.subs[zid] = zettel
		return
	}
}

// Subscribe adds a zettel to the stock.
func (s *defaultStock) Subscribe(zid id.Zid) error {
	s.mxSubs.Lock()
	defer s.mxSubs.Unlock()
	if _, found := s.subs[zid]; found {
		return nil
	}
	zettel, err := s.place.GetZettel(context.Background(), zid)
	if err != nil {
		return err
	}
	s.subs[zid] = zettel
	return nil
}

// GetZettel returns the zettel with the given zid, if in stock, else an empty zettel
func (s *defaultStock) GetZettel(zid id.Zid) domain.Zettel {
	s.mxSubs.RLock()
	defer s.mxSubs.RUnlock()
	return s.subs[zid]
}

// GetZettel returns the zettel Meta with the given zid, if in stock, else nil.
func (s *defaultStock) GetMeta(zid id.Zid) *meta.Meta {
	s.mxSubs.RLock()
	zettel := s.subs[zid]
	s.mxSubs.RUnlock()
	return zettel.Meta
}
