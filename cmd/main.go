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
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strings"

	"zettelstore.de/z/config/runtime"
	"zettelstore.de/z/config/startup"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/input"
	"zettelstore.de/z/place"
	"zettelstore.de/z/place/manager"
	"zettelstore.de/z/place/progplace"
)

const (
	defConfigfile = ".zscfg"
)

func init() {
	RegisterCommand(Command{
		Name: "help",
		Func: func(*flag.FlagSet) (int, error) {
			fmt.Println("Available commands:")
			for _, name := range List() {
				fmt.Printf("- %q\n", name)
			}
			return 0, nil
		},
	})
	RegisterCommand(Command{
		Name: "version",
		Func: func(*flag.FlagSet) (int, error) {
			fmtVersion()
			return 0, nil
		},
	})
	RegisterCommand(Command{
		Name:   "run",
		Func:   runFunc,
		Places: true,
		Flags:  flgRun,
	})
	RegisterCommand(Command{
		Name:   "run-simple",
		Func:   runSimpleFunc,
		Places: true,
		Simple: true,
		Flags:  flgSimpleRun,
	})
	RegisterCommand(Command{
		Name:  "config",
		Func:  cmdConfig,
		Flags: flgRun,
	})
	RegisterCommand(Command{
		Name: "file",
		Func: cmdFile,
		Flags: func(fs *flag.FlagSet) {
			fs.String("t", "html", "target output format")
		},
	})
	RegisterCommand(Command{
		Name: "password",
		Func: cmdPassword,
	})
}

func fmtVersion() {
	version := startup.GetVersion()
	fmt.Printf("%v (%v/%v) running on %v (%v/%v)\n",
		version.Prog, version.Build, version.GoVersion,
		version.Hostname, version.Os, version.Arch)
}

func getConfig(fs *flag.FlagSet) (cfg *meta.Meta) {
	var configFile string
	if configFlag := fs.Lookup("c"); configFlag != nil {
		configFile = configFlag.Value.String()
	} else {
		configFile = defConfigfile
	}
	content, err := ioutil.ReadFile(configFile)
	if err != nil {
		cfg = meta.New(id.Invalid)
	} else {
		cfg = meta.NewFromInput(id.Invalid, input.NewInput(string(content)))
	}
	fs.Visit(func(flg *flag.Flag) {
		switch flg.Name {
		case "p":
			cfg.Set(startup.KeyListenAddress, "127.0.0.1:"+flg.Value.String())
		case "d":
			val := flg.Value.String()
			if strings.HasPrefix(val, "/") {
				val = "dir://" + val
			} else {
				val = "dir:" + val
			}
			cfg.Set(startup.KeyPlaceOneURI, val)
		case "r":
			cfg.Set(startup.KeyReadOnlyMode, flg.Value.String())
		case "v":
			cfg.Set(startup.KeyVerbose, flg.Value.String())
		}
	})
	return cfg
}

func setupOperations(cfg *meta.Meta, withPlaces bool, simple bool) error {
	var place place.Place
	if withPlaces {
		p, err := manager.New(getPlaces(cfg))
		if err != nil {
			return err
		}
		place = p
	}

	err := startup.SetupStartup(cfg, place, simple)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to connect to specified places")
		return err
	}
	if withPlaces {
		if err := startup.Place().Start(context.Background()); err != nil {
			fmt.Fprintln(os.Stderr, "Unable to start zettel place")
			return err
		}
		runtime.SetupConfiguration(startup.Place())
		progplace.Setup(cfg, startup.Place())
	}
	return nil
}

func getPlaces(cfg *meta.Meta) []string {
	hasConst := false
	var result []string = nil
	for cnt := 1; ; cnt++ {
		key := fmt.Sprintf("place-%v-uri", cnt)
		uri, ok := cfg.Get(key)
		if !ok || uri == "" {
			if cnt > 1 {
				break
			}
			uri = "dir:./zettel"
		}
		if uri == "const:" {
			hasConst = true
		}
		if cfg.GetBool(startup.KeyReadOnlyMode) {
			if u, err := url.Parse(uri); err == nil {
				// TODO: the following is wrong under some circumstances:
				// 1. query parameter "readonly" is already set
				// 2. fragment is set
				if len(u.Query()) == 0 {
					uri += "?readonly"
				} else {
					uri += "&readonly"
				}
			}
		}
		result = append(result, uri)
	}
	if !hasConst {
		result = append(result, "const:")
	}
	return result
}
func cleanupOperations(withPlaces bool) error {
	if withPlaces {
		if err := startup.Place().Stop(context.Background()); err != nil {
			fmt.Fprintln(os.Stderr, "Unable to start zettel place")
			return err
		}
	}
	return nil
}

func executeCommand(name string, args ...string) {
	command, ok := Get(name)
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown command %q\n", name)
		os.Exit(1)
	}
	fs := command.GetFlags()
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: unable to parse flags: %v %v\n", name, args, err)
		os.Exit(1)
	}
	cfg := getConfig(fs)
	if err := setupOperations(cfg, command.Places, command.Simple); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", name, err)
		os.Exit(2)
	}

	exitCode, err := command.Func(fs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", name, err)
	}
	if err := cleanupOperations(command.Places); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", name, err)
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

// Main is the real entrypoint of the zettelstore.
func Main(progName, buildVersion string) {
	startup.SetupVersion(progName, buildVersion)
	if len(os.Args) <= 1 {
		runSimple()
	} else {
		executeCommand(os.Args[1], os.Args[2:]...)
	}
}
