package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/rstms/nbd/netboot"
	"github.com/sevlyar/go-daemon"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"
)

const Version = "0.0.2"

const serverName = "netbootd"
const SHUTDOWN_TIMEOUT = 5

var ConfigPattern = regexp.MustCompile(`^([a-zA-Z0-9_]+)=(.*)\s*$`)

var NetbootName string

var (
	signalFlag = flag.String("s", "", `send signal:
    stop - shutdown
    reload - reload config
    `)
	shutdown = make(chan struct{})
	reload   = make(chan struct{})
)

var HostCache *netboot.HostCache

func handleEndpoints(w http.ResponseWriter, r *http.Request) {

	log.Printf("%s %s %s (%d)\n", r.RemoteAddr, r.Method, r.RequestURI, r.ContentLength)
	switch r.Method {
	case "GET":
		switch r.URL.Path {
		case "/api/hosts/":
			netboot.ListHostsHandler(w, r)
			return
		default:
			if strings.HasPrefix(r.URL.Path, "/api/booted/") {
				netboot.HostBootedHandler(w, r, HostCache)
				return
			}
			if strings.HasPrefix(r.URL.Path, "/api/address/") {
				netboot.HostAddressQueryHandler(w, r, HostCache)
				return
			}
		}
	case "PUT":
		switch r.URL.Path {
		case "/api/host/":
			netboot.AddHostHandler(w, r, NetbootName, HostCache)
			return
		}
	case "DELETE":
		switch r.URL.Path {
		case "/api/host/":
			netboot.DeleteHostHandler(w, r, HostCache)
			return
		}
	case "POST":
		switch r.URL.Path {
		case "/api/tarball/":
			netboot.UploadPackageHandler(w, r)
			return
		}
	}
	http.Error(w, "WAT?", http.StatusNotFound)

}

func runServer(addr *string, port *int) {

	listen := fmt.Sprintf("%s:%d", *addr, *port)
	server := &http.Server{
		Addr:    listen,
		Handler: http.HandlerFunc(handleEndpoints),
	}

	HostCache = netboot.NewHostCache()

	go func() {
		log.Printf("%s v%s started as PID %d listening on %s\n", serverName, Version, os.Getpid(), listen)
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatalln("ListenAndServe failed: ", err)
		}
	}()

	<-shutdown

	log.Println("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), SHUTDOWN_TIMEOUT*time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		log.Fatalln("Server Shutdown failed: ", err)
	}
	log.Println("shutdown complete")
}

func stopHandler(sig os.Signal) error {
	log.Println("received stop signal")
	shutdown <- struct{}{}
	return daemon.ErrStop
}

func reloadHandler(sig os.Signal) error {
	log.Println("received reload signal")
	return nil
}

func main() {
	addr := flag.String("addr", "127.0.0.1", "listen address")
	port := flag.Int("port", 2014, "listen port")
	debugFlag := flag.Bool("debug", false, "run in foreground mode")
	flag.StringVar(&NetbootName, "name", "", "server name")
	flag.Parse()
	ReadConfigFile("/etc/nbd.conf")
	if NetbootName == "" {
		NetbootName = "netboot"
	}

	if !*debugFlag {
		daemonize(addr, port)
		os.Exit(0)
	}
	go runServer(addr, port)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM)
	<-sigs
	shutdown <- struct{}{}
	os.Exit(0)
}

func ReadConfigFile(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		match := ConfigPattern.FindStringSubmatch(line)
		if len(match) > 2 {
			key := match[1]
			value := match[2]
			switch key {
			case "netboot_name":
				if NetbootName == "" {
					NetbootName = value
				}
			}
		}
	}
}

func daemonize(addr *string, port *int) {

	daemon.AddCommand(daemon.StringFlag(signalFlag, "stop"), syscall.SIGTERM, stopHandler)
	daemon.AddCommand(daemon.StringFlag(signalFlag, "reload"), syscall.SIGHUP, reloadHandler)

	ctx := &daemon.Context{
		LogFileName: "/var/log/nbd.log",
		LogFilePerm: 0600,
		WorkDir:     "/",
		Umask:       007,
	}

	if len(daemon.ActiveFlags()) > 0 {
		d, err := ctx.Search()
		if err != nil {
			log.Fatalln("Unable to signal daemon: ", err)
		}
		daemon.SendCommands(d)
		return
	}

	child, err := ctx.Reborn()
	if err != nil {
		log.Fatalln("Fork failed: ", err)
	}

	if child != nil {
		return
	}
	defer ctx.Release()

	go runServer(addr, port)

	err = daemon.ServeSignals()
	if err != nil {
		log.Fatalln("Error: ServeSignals: ", err)
	}
}
