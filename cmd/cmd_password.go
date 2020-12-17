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
	"os"

	"golang.org/x/crypto/ssh/terminal"

	"zettelstore.de/z/auth/cred"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

// ---------- Subcommand: password -------------------------------------------

func cmdPassword(name string, fs *flag.FlagSet) (int, error) {
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "User name and user zettel identification missing")
		return 2, nil
	}
	if fs.NArg() == 1 {
		fmt.Fprintln(os.Stderr, "User zettel identification missing")
		return 2, nil
	}

	sid := fs.Arg(1)
	zid, err := id.ParseZettelID(sid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Given zettel identification is not valid: %q\n", sid)
		return 2, err
	}

	password, err := getPassword("Password")
	if err != nil {
		return 2, err
	}
	passwordAgain, err := getPassword("   Again")
	if err != nil {
		return 2, err
	}
	if string(password) != string(passwordAgain) {
		fmt.Fprintln(os.Stderr, "Passwords differ!")
		return 2, nil
	}

	ident := fs.Arg(0)
	hashedPassword, err := cred.HashCredential(zid, ident, password)
	if err != nil {
		return 2, err
	}
	fmt.Printf("%v: %s\n%v: %s\n",
		meta.MetaKeyCredential, hashedPassword,
		meta.MetaKeyUserID, ident,
	)
	return 0, nil
}

func getPassword(prompt string) (string, error) {
	fmt.Fprintf(os.Stderr, "%s: ", prompt)
	password, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	return string(password), err
}
