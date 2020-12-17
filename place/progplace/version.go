//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package progplace provides zettel that inform the user about the internal Zettelstore state.
package progplace

import (
	"fmt"

	"zettelstore.de/z/config/startup"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

func getVersionMeta(zid id.ZettelID, title string) *meta.Meta {
	m := meta.NewMeta(zid)
	m.Set(meta.MetaKeyTitle, title)
	m.Set(meta.MetaKeyRole, "configuration")
	m.Set(meta.MetaKeySyntax, "zmk")
	m.Set(meta.MetaKeyVisibility, meta.MetaValueVisibilityExpert)
	m.Set(meta.MetaKeyReadOnly, "true")
	return m
}

func genVersionBuildM(zid id.ZettelID) *meta.Meta {
	m := getVersionMeta(zid, "Zettelstore Version")
	m.Set(meta.MetaKeyVisibility, meta.MetaValueVisibilityPublic)
	return m
}
func genVersionBuildC(*meta.Meta) string { return startup.GetVersion().Build }

func genVersionHostM(zid id.ZettelID) *meta.Meta {
	return getVersionMeta(zid, "Zettelstore Host")
}
func genVersionHostC(*meta.Meta) string { return startup.GetVersion().Hostname }

func genVersionOSM(zid id.ZettelID) *meta.Meta {
	return getVersionMeta(zid, "Zettelstore Operating System")
}
func genVersionOSC(*meta.Meta) string {
	v := startup.GetVersion()
	return fmt.Sprintf("%v/%v", v.Os, v.Arch)
}

func genVersionGoM(zid id.ZettelID) *meta.Meta {
	return getVersionMeta(zid, "Zettelstore Go Version")
}
func genVersionGoC(*meta.Meta) string { return startup.GetVersion().GoVersion }
