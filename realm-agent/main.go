package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/imnebula/flux-panel-realm/realm-agent/internal/agent"
	"github.com/imnebula/flux-panel-realm/realm-agent/internal/platform"
	"github.com/imnebula/flux-panel-realm/realm-agent/internal/realm"
	"github.com/imnebula/flux-panel-realm/realm-agent/internal/reporter"
)

// Version is set at build time via -ldflags "-X main.Version=x.y.z-realm"
var Version = "dev"

func main() {
	var cfg agent.Config
	flag.StringVar(&cfg.ServerAddr, "server-addr", "", "panel host:port")
	flag.StringVar(&cfg.Secret, "secret", "", "node secret")
	flag.StringVar(&cfg.AgentName, "agent-name", "flux-realm-agent", "agent display name")
	flag.StringVar(&cfg.AgentProcessName, "agent-process-name", "flux-realm-agent", "agent process name")
	flag.StringVar(&cfg.RealmProcessName, "realm-process-name", "flux-realm", "realm process name")
	flag.StringVar(&cfg.ServiceName, "service-name", "flux-realm-agent", "service name")
	flag.StringVar(&cfg.InstanceName, "instance", "default", "instance name")
	flag.StringVar(&cfg.InstallDir, "install-dir", "/opt/flux-realm-agent", "install dir")
	flag.StringVar(&cfg.ConfigDir, "config-dir", "/etc/flux-realm/instances/default", "realm config dir")
	flag.StringVar(&cfg.LogDir, "log-dir", "/var/log/flux-realm-agent", "log dir")
	flag.StringVar(&cfg.DataDir, "data-dir", "/var/lib/flux-realm-agent", "data dir")
	flag.StringVar(&cfg.PidFile, "pid-file", "", "realm pid file")
	flag.StringVar(&cfg.Mode, "mode", "single", "single|per-tunnel|per-forward")
	flag.StringVar(&cfg.RealmBinaryPath, "realm-binary", "", "realm binary path")
	flag.BoolVar(&cfg.Foreground, "foreground", false, "run foreground")
	flag.BoolVar(&cfg.PrintVersion, "version", false, "print version")
	flag.Parse()

	if cfg.PrintVersion {
		fmt.Printf("flux-realm-agent %s\n", Version)
		return
	}
	if cfg.ServerAddr == "" || cfg.Secret == "" {
		log.Fatal("--server-addr and --secret are required")
	}
	if cfg.PidFile == "" {
		cfg.PidFile = filepath.Join(cfg.DataDir, cfg.InstanceName+".pid")
	}
	if cfg.RealmBinaryPath == "" {
		cfg.RealmBinaryPath = filepath.Join(cfg.InstallDir, "realm")
	}
	for _, p := range []string{cfg.ConfigDir, cfg.LogDir, cfg.DataDir} {
		if err := os.MkdirAll(p, 0755); err != nil {
			log.Fatalf("create %s: %v", p, err)
		}
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	env := platform.Detect()
	manager := realm.NewManager(realm.ManagerConfig{
		BinaryPath:  cfg.RealmBinaryPath,
		ConfigDir:   cfg.ConfigDir,
		LogDir:      cfg.LogDir,
		PidFile:     cfg.PidFile,
		ProcessName: cfg.RealmProcessName,
		Instance:    cfg.InstanceName,
		Mode:        cfg.Mode,
	})

	r := reporter.New(reporter.Config{
		ServerAddr:       cfg.ServerAddr,
		Secret:           cfg.Secret,
		AgentVersion:     Version,
		AgentProcessName: cfg.AgentProcessName,
		AgentName:        cfg.AgentName,
		ServiceName:      cfg.ServiceName,
		InstanceName:     cfg.InstanceName,
		ConfigDir:        cfg.ConfigDir,
		RealmBinaryPath:  cfg.RealmBinaryPath,
		RealmProcessName: cfg.RealmProcessName,
		Mode:             cfg.Mode,
	}, env, manager)

	if err := r.Run(ctx); err != nil && ctx.Err() == nil {
		log.Fatal(err)
	}
}
