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

// Package config provides functions to retrieve configuration data.
package config

import (
	"fmt"
	"hash/fnv"
	"os"
	"runtime"

	"zettelstore.de/z/domain"
	"zettelstore.de/z/store"
	"zettelstore.de/z/store/stock"
)

// Version describes all elements of a software version.
type Version struct {
	Prog      string // Name of the software
	Build     string // Representation of build process
	Hostname  string // Host name a reported by the kernel
	GoVersion string // Version of go
	Os        string // GOOS
	Arch      string // GOARCH
	// More to come
}

var version Version

// SetupVersion initializes the version data.
func SetupVersion(progName, buildVersion string) {
	version.Prog = progName
	if buildVersion == "" {
		version.Build = "unknown"
	} else {
		version.Build = buildVersion
	}
	if hn, err := os.Hostname(); err == nil {
		version.Hostname = hn
	} else {
		version.Hostname = "*unknown host*"
	}
	version.GoVersion = runtime.Version()
	version.Os = runtime.GOOS
	version.Arch = runtime.GOARCH
}

// GetVersion returns the current software version data.
func GetVersion() Version { return version }

// --- Startup config --------------------------------------------------------

var startupConfig *domain.Meta

// SetupStartup initializes the startup data.
func SetupStartup(cfg *domain.Meta) {
	if startupConfig != nil {
		panic("startupConfig already set")
	}
	cfg.Freeze()
	startupConfig = cfg
}

// IsReadOnly returns whether the system is in read-only mode or not.
func IsReadOnly() bool { return startupConfig.GetBool("readonly") }

// GetURLPrefix returns the configured prefix to be used when providing URL to
// the service.
func GetURLPrefix() string {
	return startupConfig.GetDefault("url-prefix", "/")
}

// GetSecret returns the interal application secret. It is typically used to
// encrypt session values.
func GetSecret() []byte {
	h := fnv.New128()
	if secret, ok := startupConfig.Get("secret"); ok {
		h.Write([]byte(secret))
	}
	h.Write([]byte(version.Prog))
	h.Write([]byte(version.Build))
	h.Write([]byte(version.Hostname))
	h.Write([]byte(version.GoVersion))
	h.Write([]byte(version.Os))
	h.Write([]byte(version.Arch))
	return h.Sum(nil)
}

// --- Configuration zettel --------------------------------------------------

var configStock stock.Stock

// SetupConfiguration enables the configuration data.
func SetupConfiguration(store store.Store) {
	if configStock != nil {
		panic("configStock already set")
	}
	configStock = stock.NewStock(store)
	if err := configStock.Subscribe(domain.ConfigurationID); err != nil {
		panic(err)
	}
}

// getConfigurationMeta returns the meta data of the configuration zettel.
func getConfigurationMeta() *domain.Meta {
	if configStock == nil {
		panic("configStock not set")
	}
	return configStock.GetMeta(domain.ConfigurationID)
}

// GetDefaultTitle returns the current value of the "default-title" key.
func GetDefaultTitle() string {
	if config := getConfigurationMeta(); config != nil {
		if title, ok := config.Get(domain.MetaKeyDefaultTitle); ok {
			return title
		}
	}
	return "Untitled"
}

// GetDefaultSyntax returns the current value of the "default-syntax" key.
func GetDefaultSyntax() string {
	if configStock != nil {
		if config := getConfigurationMeta(); config != nil {
			if syntax, ok := config.Get(domain.MetaKeyDefaultSyntax); ok {
				return syntax
			}
		}
	}
	return "zmk"
}

// GetDefaultRole returns the current value of the "default-role" key.
func GetDefaultRole() string {
	if configStock != nil {
		if config := getConfigurationMeta(); config != nil {
			if role, ok := config.Get(domain.MetaKeyDefaultRole); ok {
				return role
			}
		}
	}
	return "zettel"
}

// GetDefaultLang returns the current value of the "default-lang" key.
func GetDefaultLang() string {
	if configStock != nil {
		if config := getConfigurationMeta(); config != nil {
			if lang, ok := config.Get(domain.MetaKeyDefaultLang); ok {
				return lang
			}
		}
	}
	return "en"
}

// GetDefaultCopyright returns the current value of the "default-copyright" key.
func GetDefaultCopyright() string {
	if configStock != nil {
		if config := getConfigurationMeta(); config != nil {
			if copyright, ok := config.Get(domain.MetaKeyDefaultCopyright); ok {
				return copyright
			}
		}
		// TODO: get owner
	}
	return ""
}

// GetDefaultLicense returns the current value of the "default-license" key.
func GetDefaultLicense() string {
	if configStock != nil {
		if config := getConfigurationMeta(); config != nil {
			if license, ok := config.Get(domain.MetaKeyDefaultLicense); ok {
				return license
			}
		}
	}
	return ""
}

// GetSiteName returns the current value of the "site-name" key.
func GetSiteName() string {
	if config := getConfigurationMeta(); config != nil {
		if name, ok := config.Get(domain.MetaKeySiteName); ok {
			return name
		}
	}
	return "Zettelstore"
}

// GetYAMLHeader returns the current value of the "yaml-header" key.
func GetYAMLHeader() bool {
	if config := getConfigurationMeta(); config != nil {
		return config.GetBool(domain.MetaKeyYAMLHeader)
	}
	return false
}

// GetZettelFileSyntax returns the current value of the "zettel-file-syntax" key.
func GetZettelFileSyntax() []string {
	if config := getConfigurationMeta(); config != nil {
		return config.GetListOrNil(domain.MetaKeyZettelFileSyntax)
	}
	return nil
}

// GetIconMaterial returns the current value of the "icon-material" key.
func GetIconMaterial() string {
	if config := getConfigurationMeta(); config != nil {
		if html, ok := config.Get(domain.MetaKeyIconMaterial); ok {
			return html
		}
	}
	return fmt.Sprintf(
		"<img class=\"zs-text-icon\" src=\"%vc/%v\">",
		GetURLPrefix(),
		domain.MaterialIconID.Format())
}

// GetOwner returns the zid of the zettelkasten's owner.
// If there is no owner defined, the value ZettelID(0) is returned.
func GetOwner() domain.ZettelID {
	if config := getConfigurationMeta(); config != nil {
		if owner, ok := config.Get(domain.MetaKeyOwner); ok {
			if zid, err := domain.ParseZettelID(owner); err == nil {
				return zid
			}
		}
	}
	return domain.ZettelID(0)
}

var mapDefaultKeys = map[string]func() string{
	domain.MetaKeyCopyright: GetDefaultCopyright,
	domain.MetaKeyLang:      GetDefaultLang,
	domain.MetaKeyLicense:   GetDefaultLicense,
	domain.MetaKeyRole:      GetDefaultRole,
	domain.MetaKeySyntax:    GetDefaultSyntax,
	domain.MetaKeyTitle:     GetDefaultTitle,
}

// AddDefaultValues enriches the given meta data with its default values.
func AddDefaultValues(meta *domain.Meta) *domain.Meta {
	result := meta
	for k, f := range mapDefaultKeys {
		if _, ok := result.Get(k); !ok {
			if result == meta {
				result = meta.Clone()
			}
			if val := f(); len(val) > 0 || meta.Type(k) == domain.MetaTypeEmpty {
				result.Set(k, val)
			}
		}
	}
	if result != meta && meta.IsFrozen() {
		result.Freeze()
	}
	return result
}

// GetSyntax returns the value of the "syntax" key of the given meta. If there
// is no such value, GetDefaultLang is returned.
func GetSyntax(meta *domain.Meta) string {
	if syntax, ok := meta.Get(domain.MetaKeySyntax); ok && len(syntax) > 0 {
		return syntax
	}
	return GetDefaultSyntax()
}

// GetLang returns the value of the "lang" key of the given meta. If there is
// no such value, GetDefaultLang is returned.
func GetLang(meta *domain.Meta) string {
	if lang, ok := meta.Get(domain.MetaKeyLang); ok && len(lang) > 0 {
		return lang
	}
	return GetDefaultLang()
}
