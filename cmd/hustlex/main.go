// Command hustlex is the hustleX desktop app.
//
//	hustlex                            open this machine (starting its daemon)
//	hustlex --remote user@host[:port]  open a server over SSH
//	hustlex --project DIR              also open DIR as a project
//	hustlex daemon                     run this machine's daemon (started by the app)
//	hustlex hook STATE                 report an agent's state (run by its hooks)
//	hustlex ls|spawn|send|wait|…       the command line (hustlex help)
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"

	"github.com/hustlex/hustlex/cli"
	"github.com/hustlex/hustlex/core/hosts"
	"github.com/hustlex/hustlex/daemon"
	"github.com/hustlex/hustlex/transport"
	hustlexui "github.com/hustlex/hustlex/ui"
)

func main() {
	if err := daemon.MoveOldHome(); err != nil {
		fmt.Fprintln(os.Stderr, "hustlex:", err)
	}
	if len(os.Args) > 1 && os.Args[1] == "daemon" {
		if err := daemon.Run(); err != nil {
			fmt.Fprintln(os.Stderr, "hustlex daemon:", err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "hook" {
		daemon.Hook(os.Args[2:])
		return
	}
	if cli.IsDockerHelper(os.Args[0]) {
		if err := cli.DockerHelper(os.Args[1:], os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, "hustlex:", err)
			os.Exit(1)
		}
		return
	}
	if cli.Is(os.Args[1:]) {
		if err := cli.Run(os.Args[1:], os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, "hustlex:", err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "display" {
		if err := daemon.Display(); err != nil {
			fmt.Fprintln(os.Stderr, "hustlex display:", err)
			os.Exit(1)
		}
		return
	}

	remote := flag.String("remote", "", "open the server at `user@host[:port]` over SSH")
	project := flag.String("project", "", "open `DIR` (a path on that host) as a project")
	flag.Parse()

	dir, err := transport.Dir()
	if err != nil {
		log.Fatal(err)
	}
	store, err := hosts.Open(filepath.Join(dir, "hosts.json"))
	if err != nil {
		log.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	start := hustlexui.LocalHost()
	if *remote != "" {
		start = hosts.Host{ID: "cli", Name: *remote, Address: *remote}
		for _, h := range store.List() {
			if h.Address == *remote {
				start = h
			}
		}
	}
	app := hustlexui.New(hustlexui.Options{
		Hosts:         store,
		Start:         start,
		DaemonCommand: []string{daemonExe(exe, dir), "daemon"},
		OpenProject:   *project,
		StateFile:     filepath.Join(dir, "app.json"),
		ResumeHost:    *remote == "",
	})
	mygo.App.WhenReady(func() { app.Open() })
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
