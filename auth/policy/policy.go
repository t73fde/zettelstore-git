//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
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
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

// Policy is an interface for checking access authorization.
type Policy interface {
	// User is allowed to reload a place.
	CanReload(user *meta.Meta) bool

	// User is allowed to create a new zettel.
	CanCreate(user *meta.Meta, newMeta *meta.Meta) bool

	// User is allowed to read zettel
	CanRead(user *meta.Meta, m *meta.Meta) bool

	// User is allowed to write zettel.
	CanWrite(user *meta.Meta, oldMeta, newMeta *meta.Meta) bool

	// User is allowed to rename zettel
	CanRename(user *meta.Meta, m *meta.Meta) bool

	// User is allowed to delete zettel
	CanDelete(user *meta.Meta, m *meta.Meta) bool
}

// newPolicy creates a policy based on given constraints.
func newPolicy(
	withAuth func() bool,
	isReadOnlyMode bool,
	expertMode func() bool,
	isOwner func(id.ZettelID) bool,
	getVisibility func(*meta.Meta) meta.Visibility,
) Policy {
	var pol Policy
	if isReadOnlyMode {
		pol = &roPolicy{}
	} else {
		pol = &defaultPolicy{}
	}
	if withAuth() {
		pol = &ownerPolicy{
			expertMode:    expertMode,
			isOwner:       isOwner,
			getVisibility: getVisibility,
			pre:           pol,
		}
	} else {
		pol = &anonPolicy{
			expertMode:    expertMode,
			getVisibility: getVisibility,
			pre:           pol,
		}
	}
	return &prePolicy{pol}
}

type prePolicy struct {
	post Policy
}

func (p *prePolicy) CanReload(user *meta.Meta) bool {
	return p.post.CanReload(user)
}

func (p *prePolicy) CanCreate(user *meta.Meta, newMeta *meta.Meta) bool {
	return newMeta != nil && p.post.CanCreate(user, newMeta)
}

func (p *prePolicy) CanRead(user *meta.Meta, m *meta.Meta) bool {
	return m != nil && p.post.CanRead(user, m)
}

func (p *prePolicy) CanWrite(user *meta.Meta, oldMeta, newMeta *meta.Meta) bool {
	return oldMeta != nil && newMeta != nil && oldMeta.Zid == newMeta.Zid &&
		p.post.CanWrite(user, oldMeta, newMeta)
}

func (p *prePolicy) CanRename(user *meta.Meta, m *meta.Meta) bool {
	return m != nil && p.post.CanRename(user, m)
}

func (p *prePolicy) CanDelete(user *meta.Meta, m *meta.Meta) bool {
	return m != nil && p.post.CanDelete(user, m)
}
