package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/scottjab/nix-ugreen-leds-controller/internal/config"
	"github.com/scottjab/nix-ugreen-leds-controller/internal/diskmon"
	"github.com/scottjab/nix-ugreen-leds-controller/internal/netmon"
)

var (
	configFile = flag.String("config", "/etc/ugreen-leds.conf", "Path to configuration file")
)

func main() {
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Printf("Warning: Failed to load config file %s: %v. Using defaults.", *configFile, err)
		cfg = &config.Config{}
		cfg.SetDefaults()
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received shutdown signal, cleaning up...")
		cancel()
	}()

	// Ensure kernel modules are loaded
	if err := ensureKernelModules(); err != nil {
		log.Fatalf("Failed to ensure kernel modules: %v", err)
	}

	var wg sync.WaitGroup

	// Start disk monitor if enabled
	if cfg.DiskMonitor.Enable {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := diskmon.Run(ctx, &cfg.DiskMonitor); err != nil {
				log.Printf("Disk monitor error: %v", err)
			}
		}()
	}

	// Start network monitors for each interface
	if cfg.NetworkMonitor.Enable {
		for _, iface := range cfg.NetworkMonitor.Interfaces {
			wg.Add(1)
			go func(interfaceName string) {
				defer wg.Done()
				if err := netmon.Run(ctx, &cfg.NetworkMonitor, interfaceName); err != nil {
					log.Printf("Network monitor error for %s: %v", interfaceName, err)
				}
			}(iface)
		}
	}

	// Wait for all monitors to finish
	wg.Wait()
	log.Println("Service stopped")
}

func ensureKernelModules() error {
	modules := []string{"ledtrig_oneshot", "ledtrig_netdev"}
	for _, mod := range modules {
		if err := loadKernelModule(mod); err != nil {
			return fmt.Errorf("failed to load %s: %w", mod, err)
		}
	}
	return nil
}

func loadKernelModule(name string) error {
	// Check if module is already loaded
	f, err := os.Open("/proc/modules")
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := os.ReadFile("/proc/modules")
	if err != nil {
		return err
	}

	if strings.Contains(string(data), name+" ") {
		return nil // Already loaded
	}

	// Try to load the module
	// Note: This requires CAP_SYS_MODULE capability
	// In practice, modules should be loaded by systemd-modules-load.service
	log.Printf("Warning: Module %s not loaded. Ensure it's loaded via systemd-modules-load.service", name)
	return nil
}

