//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package cmd

import (
	"flag"
	"fmt"

	"zettelstore.de/z/config/startup"
)

// ---------- Subcommand: config ---------------------------------------------

func cmdConfig(string, *flag.FlagSet) (int, error) {
	fmtVersion()
	fmt.Println("Stores")
	fmt.Printf("  Read-only mode    = %v\n", startup.IsReadOnlyMode())
	fmt.Println("Web")
	fmt.Printf("  Listen address    = %q\n", startup.ListenAddress())
	fmt.Printf("  URL prefix        = %q\n", startup.URLPrefix())
	if startup.WithAuth() {
		fmt.Println("Auth")
		fmt.Printf("  Owner             = %v\n", startup.Owner().Format())
		fmt.Printf("  Secure cookie     = %v\n", startup.SecureCookie())
		fmt.Printf("  Persistent cookie = %v\n", startup.PersistentCookie())
		htmlLifetime, apiLifetime := startup.TokenLifetime()
		fmt.Printf("  HTML lifetime     = %v\n", htmlLifetime)
		fmt.Printf("  API lifetime      = %v\n", apiLifetime)
	}

	return 0, nil
}
