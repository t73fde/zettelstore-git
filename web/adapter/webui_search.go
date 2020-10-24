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

// Package adapter provides handlers for web requests.
package adapter

import (
	"log"
	"net/http"
	"strconv"

	"zettelstore.de/z/config"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/place"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/session"
)

// MakeSearchHandler creates a new HTTP handler for the use case "search".
func MakeSearchHandler(te *TemplateEngine, search usecase.Search, getMeta usecase.GetMeta, getZettel usecase.GetZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		var filter *place.Filter
		var sorter *place.Sorter
		for key, values := range query {
			switch key {
			case "offset":
				if len(values) > 0 {
					if offset, err := strconv.Atoi(values[0]); err == nil {
						sorter = ensureSorter(sorter)
						sorter.Offset = offset
					}
				}
			case "limit":
				if len(values) > 0 {
					if limit, err := strconv.Atoi(values[0]); err == nil {
						sorter = ensureSorter(sorter)
						sorter.Limit = limit
					}
				}
			case "negate":
				filter = ensureFilter(filter)
				filter.Negate = true
			case "s":
				cleanedValues := make([]string, 0, len(values))
				for _, val := range values {
					if len(val) > 0 {
						cleanedValues = append(cleanedValues, val)
					}
				}
				if len(cleanedValues) > 0 {
					filter = ensureFilter(filter)
					filter.Expr[""] = cleanedValues
				}
			}
		}
		if filter == nil || len(filter.Expr) == 0 {
			http.Redirect(w, r, newURLBuilder('h').String(), http.StatusFound)
			return
		}

		metaList, err := search.Run(r.Context(), filter, sorter)
		if err != nil {
			checkUsecaseError(w, err)
			return
		}

		ctx := r.Context()
		user := session.GetUser(ctx)
		if format := getFormat(r, "html"); format != "html" {
			w.Header().Set("Content-Type", format2ContentType(format))
			part := getPart(r, "meta")
			switch format {
			case "json", "djson":
				renderListMetaJSON(ctx, w, metaList, format, part, getMeta, getZettel)
				return
			}
		}

		metas, err := buildHTMLMetaList(metaList)
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			log.Println(err)
			return
		}
		te.renderTemplate(ctx, w, domain.ListTemplateID, struct {
			baseData
			Metas []metaInfo
			Key   byte
		}{
			baseData: te.makeBaseData(ctx, config.GetDefaultLang(), config.GetSiteName(), user),
			Metas:    metas,
			Key:      'h',
		})
	}
}
