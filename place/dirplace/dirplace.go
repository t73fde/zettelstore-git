//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package dirplace provides a directory-based zettel place.
package dirplace

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"zettelstore.de/z/config/runtime"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/place"
	"zettelstore.de/z/place/dirplace/directory"
	"zettelstore.de/z/place/manager"
)

func init() {
	manager.Register("dir", func(u *url.URL, next place.Place) (place.Place, error) {
		path := getDirPath(u)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil, err
		}
		dp := dirPlace{
			u:        u,
			readonly: getQueryBool(u, "readonly"),
			next:     next,
			dir:      path,
			dirRescan: time.Duration(
				getQueryInt(u, "rescan", 60, 600, 30*24*60*60)) * time.Second,
			fSrvs: uint32(getQueryInt(u, "worker", 1, 17, 1499)),
		}
		return &dp, nil
	})
}

func getDirPath(u *url.URL) string {
	if u.Opaque != "" {
		return filepath.Clean(u.Opaque)
	}
	return filepath.Clean(u.Path)
}

func getQueryBool(u *url.URL, key string) bool {
	_, ok := u.Query()[key]
	return ok
}

func getQueryInt(u *url.URL, key string, min, def, max int) int {
	sVal := u.Query().Get(key)
	if sVal == "" {
		return def
	}
	iVal, err := strconv.Atoi(sVal)
	if err != nil {
		return def
	}
	if iVal < min {
		return min
	}
	if iVal > max {
		return max
	}
	return iVal
}

// dirPlace uses a directory to store zettel as files.
type dirPlace struct {
	u          *url.URL
	readonly   bool
	next       place.Place
	observers  []place.ObserverFunc
	mxObserver sync.RWMutex
	dir        string
	dirRescan  time.Duration
	dirSrv     *directory.Service
	fSrvs      uint32
	fCmds      []chan fileCmd
	mxCmds     sync.RWMutex
}

func (dp *dirPlace) isStopped() bool {
	return dp.dirSrv == nil
}

func (dp *dirPlace) Next() place.Place { return dp.next }

func (dp *dirPlace) Location() string {
	return dp.u.String()
}

func (dp *dirPlace) Start(ctx context.Context) error {
	if !dp.isStopped() {
		panic("Calling dirplace.Start() twice.")
	}
	return dp.localStart(ctx)
}

func (dp *dirPlace) localStart(ctx context.Context) error {
	dp.mxCmds.Lock()
	dp.fCmds = make([]chan fileCmd, 0, dp.fSrvs)
	for i := uint32(0); i < dp.fSrvs; i++ {
		cc := make(chan fileCmd)
		go fileService(i, cc)
		dp.fCmds = append(dp.fCmds, cc)
	}
	dp.dirSrv = directory.NewService(dp.dir, dp.dirRescan)
	dp.mxCmds.Unlock()
	dp.dirSrv.Subscribe(dp.notifyChanged)
	dp.dirSrv.Start()
	return nil
}

func (dp *dirPlace) notifyChanged(reason place.ChangeReason, zid id.Zid) {
	dp.mxObserver.RLock()
	observers := dp.observers
	dp.mxObserver.RUnlock()
	for _, ob := range observers {
		ob(reason, zid)
	}
}

func (dp *dirPlace) getFileChan(zid id.Zid) chan fileCmd {
	// Based on https://en.wikipedia.org/wiki/Fowler%E2%80%93Noll%E2%80%93Vo_hash_function
	var sum uint32 = 2166136261 ^ uint32(zid)
	sum *= 16777619
	sum ^= uint32(zid >> 32)
	sum *= 16777619

	dp.mxCmds.RLock()
	defer dp.mxCmds.RUnlock()
	return dp.fCmds[sum%dp.fSrvs]
}

func (dp *dirPlace) Stop(ctx context.Context) error {
	if dp.isStopped() {
		return place.ErrStopped
	}
	if err := dp.localStop(ctx); err != nil {
		return err
	}
	return nil
}

func (dp *dirPlace) localStop(ctx context.Context) error {

	dirSrv := dp.dirSrv
	dp.dirSrv = nil
	dirSrv.Stop()
	for _, c := range dp.fCmds {
		close(c)
	}
	return nil
}

// RegisterChangeObserver registers an observer that will be notified
// if a zettel was found to be changed.
// possibly changed.
func (dp *dirPlace) RegisterChangeObserver(f place.ObserverFunc) {
	dp.mxObserver.Lock()
	dp.observers = append(dp.observers, f)
	dp.mxObserver.Unlock()
}

func (dp *dirPlace) CanCreateZettel(ctx context.Context) bool {
	return !dp.isStopped() && !dp.readonly
}

func (dp *dirPlace) CreateZettel(
	ctx context.Context, zettel domain.Zettel) (id.Zid, error) {
	if dp.isStopped() {
		return id.Invalid, place.ErrStopped
	}
	if dp.readonly {
		return id.Invalid, place.ErrReadOnly
	}

	meta := zettel.Meta
	entry := dp.dirSrv.GetNew()
	meta.Zid = entry.Zid
	dp.updateEntryFromMeta(&entry, meta)

	err := setZettel(dp, &entry, zettel)
	if err == nil {
		dp.dirSrv.UpdateEntry(&entry)
		dp.notifyChanged(place.OnCreate, meta.Zid)
	}
	return meta.Zid, err
}

// GetZettel reads the zettel from a file.
func (dp *dirPlace) GetZettel(ctx context.Context, zid id.Zid) (domain.Zettel, error) {
	if dp.isStopped() {
		return domain.Zettel{}, place.ErrStopped
	}

	entry := dp.dirSrv.GetEntry(zid)
	if !entry.IsValid() {
		if dp.next != nil {
			return dp.next.GetZettel(ctx, zid)
		}
		return domain.Zettel{}, place.ErrNotFound
	}

	m, c, err := getMetaContent(dp, &entry, zid)
	if err != nil {
		return domain.Zettel{}, err
	}
	dp.cleanupMeta(ctx, m)
	zettel := domain.Zettel{Meta: m, Content: domain.NewContent(c)}
	return zettel, nil
}

// GetMeta retrieves just the meta data of a specific zettel.
func (dp *dirPlace) GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error) {
	if dp.isStopped() {
		return nil, place.ErrStopped
	}
	entry := dp.dirSrv.GetEntry(zid)
	if !entry.IsValid() {
		if dp.next != nil {
			return dp.next.GetMeta(ctx, zid)
		}
		return nil, place.ErrNotFound
	}

	m, err := getMeta(dp, &entry, zid)
	if err != nil {
		return nil, err
	}
	dp.cleanupMeta(ctx, m)
	return m, nil
}

// SelectMeta returns all zettel meta data that match the selection
// criteria. The result is ordered by descending zettel id.
func (dp *dirPlace) SelectMeta(
	ctx context.Context, f *place.Filter, s *place.Sorter) (res []*meta.Meta, err error) {
	if dp.isStopped() {
		return nil, place.ErrStopped
	}

	hasMatch := place.CreateFilterFunc(f)
	entries := dp.dirSrv.GetEntries()
	res = make([]*meta.Meta, 0, len(entries))
	for _, entry := range entries {
		// TODO: execute requests in parallel
		m, err := getMeta(dp, &entry, entry.Zid)
		if err != nil {
			continue
		}
		dp.cleanupMeta(ctx, m)

		if hasMatch(m) {
			res = append(res, m)
		}
	}
	if err != nil {
		return nil, err
	}
	if dp.next != nil {
		other, err := dp.next.SelectMeta(ctx, f, nil)
		if err != nil {
			return nil, err
		}
		return place.MergeSorted(place.ApplySorter(res, nil), other, s), err
	}
	return place.ApplySorter(res, s), nil
}

func (dp *dirPlace) CanUpdateZettel(ctx context.Context, zettel domain.Zettel) bool {
	return !dp.isStopped() && !dp.readonly
}

func (dp *dirPlace) UpdateZettel(ctx context.Context, zettel domain.Zettel) error {
	if dp.isStopped() {
		return place.ErrStopped
	}
	if dp.readonly {
		return place.ErrReadOnly
	}

	meta := zettel.Meta
	if !meta.Zid.IsValid() {
		return &place.ErrInvalidID{Zid: meta.Zid}
	}
	entry := dp.dirSrv.GetEntry(meta.Zid)
	if !entry.IsValid() {
		// Existing zettel, but new in this place.
		entry.Zid = meta.Zid
		dp.updateEntryFromMeta(&entry, meta)
	} else if entry.MetaSpec == directory.MetaSpecNone {
		if defaultMeta := entry.CalcDefaultMeta(); !meta.Equal(defaultMeta) {
			dp.updateEntryFromMeta(&entry, meta)
			dp.dirSrv.UpdateEntry(&entry)
		}
	}
	dp.notifyChanged(place.OnUpdate, meta.Zid)
	return setZettel(dp, &entry, zettel)
}

func (dp *dirPlace) updateEntryFromMeta(entry *directory.Entry, meta *meta.Meta) {
	entry.MetaSpec, entry.ContentExt = calcSpecExt(meta)
	basePath := filepath.Join(dp.dir, entry.Zid.String())
	if entry.MetaSpec == directory.MetaSpecFile {
		entry.MetaPath = basePath + ".meta"
	}
	entry.ContentPath = basePath + "." + entry.ContentExt
	entry.Duplicates = false
}

func calcSpecExt(m *meta.Meta) (directory.MetaSpec, string) {
	if m.YamlSep {
		return directory.MetaSpecHeader, "zettel"
	}
	syntax := m.GetDefault(meta.KeySyntax, "bin")
	switch syntax {
	case meta.ValueSyntaxMeta, meta.ValueSyntaxZmk:
		return directory.MetaSpecHeader, "zettel"
	}
	for _, s := range runtime.GetZettelFileSyntax() {
		if s == syntax {
			return directory.MetaSpecHeader, "zettel"
		}
	}
	return directory.MetaSpecFile, syntax
}

func (dp *dirPlace) CanRenameZettel(ctx context.Context, zid id.Zid) bool {
	if dp.isStopped() || dp.readonly {
		return false
	}
	return true
}

// Rename changes the current zettel id to a new zettel id.
func (dp *dirPlace) RenameZettel(ctx context.Context, curZid, newZid id.Zid) error {
	if dp.isStopped() {
		return place.ErrStopped
	}
	if dp.readonly {
		return place.ErrReadOnly
	}
	if curZid == newZid {
		return nil
	}
	curEntry := dp.dirSrv.GetEntry(curZid)
	if !curEntry.IsValid() {
		if dp.next != nil {
			return dp.next.RenameZettel(ctx, curZid, newZid)
		}
		return nil
	}

	// Check whether zettel with new ID already exists in this place or in next places
	if _, err := dp.GetMeta(ctx, newZid); err == nil {
		return &place.ErrInvalidID{Zid: newZid}
	}

	oldMeta, oldContent, err := getMetaContent(dp, &curEntry, curZid)
	if err != nil {
		return err
	}

	newEntry := directory.Entry{
		Zid:         newZid,
		MetaSpec:    curEntry.MetaSpec,
		MetaPath:    renamePath(curEntry.MetaPath, curZid, newZid),
		ContentPath: renamePath(curEntry.ContentPath, curZid, newZid),
		ContentExt:  curEntry.ContentExt,
	}

	if err := dp.dirSrv.RenameEntry(&curEntry, &newEntry); err != nil {
		return err
	}
	oldMeta.Zid = newZid
	newZettel := domain.Zettel{Meta: oldMeta, Content: domain.NewContent(oldContent)}
	if err := setZettel(dp, &newEntry, newZettel); err != nil {
		// "Rollback" rename. No error checking...
		dp.dirSrv.RenameEntry(&newEntry, &curEntry)
		return err
	}
	if err := deleteZettel(dp, &curEntry, curZid); err != nil {
		return err
	}
	dp.notifyChanged(place.OnDelete, curZid)
	dp.notifyChanged(place.OnCreate, newZid)
	return nil
}

func (dp *dirPlace) CanDeleteZettel(ctx context.Context, zid id.Zid) bool {
	if dp.isStopped() || dp.readonly {
		return false
	}
	entry := dp.dirSrv.GetEntry(zid)
	return entry.IsValid() || (dp.next != nil && dp.next.CanDeleteZettel(ctx, zid))
}

// DeleteZettel removes the zettel from the place.
func (dp *dirPlace) DeleteZettel(ctx context.Context, zid id.Zid) error {
	if dp.isStopped() {
		return place.ErrStopped
	}
	if dp.readonly {
		return place.ErrReadOnly
	}

	entry := dp.dirSrv.GetEntry(zid)
	if !entry.IsValid() {
		dp.notifyChanged(place.OnDelete, zid)
		return nil
	}
	dp.dirSrv.DeleteEntry(zid)
	err := deleteZettel(dp, &entry, zid)
	dp.notifyChanged(place.OnDelete, zid)
	return err
}

// Reload clears all caches, reloads all internal data to reflect changes
// that were possibly undetected.
func (dp *dirPlace) Reload(ctx context.Context) error {
	if dp.isStopped() {
		return place.ErrStopped
	}

	// Brute force: stop everything, then start everything.
	// Could be done better in the future...
	err := dp.localStop(ctx)
	if err == nil {
		err = dp.localStart(ctx)
	}
	return err
}

func (dp *dirPlace) cleanupMeta(ctx context.Context, m *meta.Meta) {
	if role, ok := m.Get(meta.KeyRole); !ok || role == "" {
		m.Set(meta.KeyRole, runtime.GetDefaultRole())
	}
	if syntax, ok := m.Get(meta.KeySyntax); !ok || syntax == "" {
		m.Set(meta.KeySyntax, runtime.GetDefaultSyntax())
	}
}

func renamePath(path string, curID, newID id.Zid) string {
	dir, file := filepath.Split(path)
	if cur := curID.String(); strings.HasPrefix(file, cur) {
		file = newID.String() + file[len(cur):]
		return filepath.Join(dir, file)
	}
	return path
}
