//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package place provides a generic interface to zettel places.
package place

import (
	"sort"
	"strconv"

	"zettelstore.de/z/domain/meta"
)

// EnsureSorter makes sure that there is a sorter object.
func EnsureSorter(sorter *Sorter) *Sorter {
	if sorter == nil {
		sorter = new(Sorter)
	}
	return sorter
}

// ApplySorter applies the given sorter to the slide of meta data.
func ApplySorter(metaList []*meta.Meta, s *Sorter) []*meta.Meta {
	if len(metaList) == 0 {
		return metaList
	}

	if s == nil {
		sort.Slice(
			metaList,
			func(i, j int) bool {
				return metaList[i].Zid > metaList[j].Zid
			})
		return metaList
	}
	var sorter sortFunc
	if s.Order == "" {
		sorter = getSortFunc(meta.MetaKeyID, true, metaList)
	} else {
		sorter = getSortFunc(s.Order, s.Descending, metaList)
	}
	sort.Slice(metaList, sorter)
	if s.Offset > 0 {
		if s.Offset > len(metaList) {
			return nil
		}
		metaList = metaList[s.Offset:]
	}
	if s.Limit > 0 && s.Limit < len(metaList) {
		metaList = metaList[:s.Limit]
	}
	return metaList
}

type sortFunc func(i, j int) bool

func getSortFunc(key string, descending bool, ml []*meta.Meta) sortFunc {
	keyType := meta.KeyType(key)
	if key == meta.MetaKeyID || keyType == meta.MetaTypeCredential {
		if descending {
			return func(i, j int) bool { return ml[i].Zid > ml[j].Zid }
		}
		return func(i, j int) bool { return ml[i].Zid < ml[j].Zid }
	} else if keyType == meta.MetaTypeBool {
		if descending {
			return func(i, j int) bool {
				left := ml[i].GetBool(key)
				if left == ml[j].GetBool(key) {
					return i > j
				}
				return left
			}
		}
		return func(i, j int) bool {
			right := ml[j].GetBool(key)
			if ml[i].GetBool(key) == right {
				return i < j
			}
			return right
		}
	} else if keyType == meta.MetaTypeNumber {
		if descending {
			return func(i, j int) bool {
				iVal, iOk := getNum(ml[i], key)
				jVal, jOk := getNum(ml[j], key)
				return (iOk && (!jOk || iVal > jVal)) || !jOk
			}
		}
		return func(i, j int) bool {
			iVal, iOk := getNum(ml[i], key)
			jVal, jOk := getNum(ml[j], key)
			return (iOk && (!jOk || iVal < jVal)) || !jOk
		}
	}

	if descending {
		return func(i, j int) bool {
			iVal, iOk := ml[i].Get(key)
			jVal, jOk := ml[j].Get(key)
			return (iOk && (!jOk || iVal > jVal)) || !jOk
		}
	}
	return func(i, j int) bool {
		iVal, iOk := ml[i].Get(key)
		jVal, jOk := ml[j].Get(key)
		return (iOk && (!jOk || iVal < jVal)) || !jOk
	}
}

func getNum(m *meta.Meta, key string) (int, bool) {
	if s, ok := m.Get(key); ok {
		if i, err := strconv.Atoi(s); err == nil {
			return i, true
		}
	}
	return 0, false
}
