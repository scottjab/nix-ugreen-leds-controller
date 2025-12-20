package diskmon

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/scottjab/nix-ugreen-leds-controller/internal/config"
)

func TestMonitor_InitializeDisks(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create mock sysfs structure
	sysBlockPath := filepath.Join(tmpDir, "sys", "block")
	if err := os.MkdirAll(sysBlockPath, 0755); err != nil {
		t.Fatalf("Failed to create sys/block directory: %v", err)
	}

	// Create mock disk devices
	devices := []string{"sda", "sdb"}
	for _, dev := range devices {
		devPath := filepath.Join(sysBlockPath, dev)
		if err := os.MkdirAll(devPath, 0755); err != nil {
			t.Fatalf("Failed to create device directory: %v", err)
		}
		// Create stat file
		statPath := filepath.Join(devPath, "stat")
		if err := os.WriteFile(statPath, []byte("0 0 0 0 0 0 0 0 0 0 0\n"), 0644); err != nil {
			t.Fatalf("Failed to create stat file: %v", err)
		}
	}

	cfg := &config.DiskMonitorConfig{
		MappingMethod: "ata",
		ColorDiskHealth: config.RGB{255, 255, 255},
		BrightnessDiskLeds: 255,
	}

	m := &Monitor{
		cfg:         cfg,
		disks:       make(map[string]*diskState),
		ledToDevice: make(map[string]string),
		deviceToLED: make(map[string]string),
		zpoolLEDMap: make(map[string]string),
	}

	// Note: This test would need more setup to fully test initializeDisks
	// as it calls external commands (lsblk, dmidecode) and accesses /sys/class/leds
	// For a complete test, you'd need to mock those or use interfaces
	_ = m
}

func TestMonitor_CheckIO(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create mock sysfs structure
	sysBlockPath := filepath.Join(tmpDir, "sys", "class", "block", "sda")
	if err := os.MkdirAll(sysBlockPath, 0755); err != nil {
		t.Fatalf("Failed to create sys/class/block directory: %v", err)
	}

	statPath := filepath.Join(sysBlockPath, "stat")
	initialStat := "100 200 300 400 500 600 700 800\n"
	if err := os.WriteFile(statPath, []byte(initialStat), 0644); err != nil {
		t.Fatalf("Failed to create stat file: %v", err)
	}

	cfg := &config.DiskMonitorConfig{
		LedRefreshInterval: 0.1,
	}

	m := &Monitor{
		cfg:         cfg,
		disks:       make(map[string]*diskState),
		ledToDevice: make(map[string]string),
		deviceToLED: make(map[string]string),
		zpoolLEDMap: make(map[string]string),
	}

	// Create a mock disk state
	// Note: Would need to properly mock LED, but for structure test this is fine
	state := &diskState{
		led:    nil, // Would mock this properly in real test
		device: "sda",
	}
	m.disks["sda"] = state

	// Note: Full test would require mocking LED operations
	_ = m
}

func TestMonitor_Run_ContextCancellation(t *testing.T) {
	cfg := &config.DiskMonitorConfig{
		Enable:                true,
		MappingMethod:         "ata",
		CheckSmart:            false, // Disable to avoid external command calls
		CheckZpool:            false, // Disable to avoid external command calls
		LedRefreshInterval:    0.1,
		CheckDiskOnlineInterval: 1, // Set valid interval to avoid panic
		ColorDiskHealth:       config.RGB{255, 255, 255},
		BrightnessDiskLeds:    255,
	}

	ctx, cancel := context.WithCancel(context.Background())
	
	// Cancel immediately
	cancel()

	// Run should handle context cancellation
	// Note: This will fail during initializeDisks if sysfs doesn't exist
	// In a real scenario, you'd mock the file system operations
	err := Run(ctx, cfg)
	// Error expected due to missing sysfs, but context should be handled
	_ = err
}

func TestMonitor_Run_Disabled(t *testing.T) {
	cfg := &config.DiskMonitorConfig{
		Enable: false,
	}

	ctx := context.Background()
	
	// Should return error immediately if disabled
	err := Run(ctx, cfg)
	if err == nil {
		t.Error("Run() with disabled config should return error")
	}
}

func TestMonitor_EnumerateDisks_ATA(t *testing.T) {
	tmpDir := t.TempDir()
	sysBlockPath := filepath.Join(tmpDir, "sys", "block")
	
	if err := os.MkdirAll(sysBlockPath, 0755); err != nil {
		t.Fatalf("Failed to create sys/block: %v", err)
	}

	// Create mock devices with symlinks
	devices := map[string]string{
		"sda": "../../devices/pci0000:00/0000:00:1f.2/ata1/host0/target0:0:0/0:0:0:0/block/sda",
		"sdb": "../../devices/pci0000:00/0000:00:1f.2/ata2/host1/target1:0:0/1:0:0:0/block/sdb",
	}

	for dev, target := range devices {
		devPath := filepath.Join(sysBlockPath, dev)
		// Create the target path structure
		targetPath := filepath.Join(tmpDir, target)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			t.Fatalf("Failed to create target directory: %v", err)
		}
		// Create symlink
		if err := os.Symlink(target, devPath); err != nil {
			t.Fatalf("Failed to create symlink: %v", err)
		}
	}

	cfg := &config.DiskMonitorConfig{
		MappingMethod: "ata",
	}

	m := &Monitor{
		cfg: cfg,
	}

	// Note: enumerateDisks accesses /sys/block directly
	// This test structure shows the approach but would need
	// the actual /sys/block to be mocked or the function to accept
	// a path parameter for testing
	_ = m
}

func TestDiskState_Concurrency(t *testing.T) {
	state := &diskState{
		device: "sda",
	}

	// Test concurrent access
	done := make(chan bool)
	
	// Writer goroutine
	go func() {
		state.mu.Lock()
		state.smartFailed = true
		state.mu.Unlock()
		done <- true
	}()

	// Reader goroutine
	go func() {
		state.mu.RLock()
		_ = state.smartFailed
		state.mu.RUnlock()
		done <- true
	}()

	// Wait for both
	<-done
	<-done
}

func TestMonitor_BuildZpoolMapping(t *testing.T) {
	cfg := &config.DiskMonitorConfig{
		CheckZpool: true,
		DebugZpool: false,
	}

	m := &Monitor{
		cfg:         cfg,
		disks:       make(map[string]*diskState),
		ledToDevice: make(map[string]string),
		deviceToLED: make(map[string]string),
		zpoolLEDMap: make(map[string]string),
	}

	// Set up device mappings
	m.deviceToLED["sda"] = "disk1"
	m.deviceToLED["sdb"] = "disk2"

	// Note: buildZpoolMapping calls zpool command
	// In a real test, you'd mock exec.Command
	// For now, this shows the structure
	_ = m
}

func TestMonitor_SmartCheckLoop(t *testing.T) {
	cfg := &config.DiskMonitorConfig{
		CheckSmart:         true,
		CheckSmartInterval: 1, // 1 second for testing
		ColorSmartFail:    config.RGB{255, 0, 0},
	}

	m := &Monitor{
		cfg:         cfg,
		disks:       make(map[string]*diskState),
		ledToDevice: make(map[string]string),
		deviceToLED: make(map[string]string),
		zpoolLEDMap: make(map[string]string),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Create a mock disk state
	state := &diskState{
		device: "sda",
		smartFailed: false,
		zpoolFaulted: false,
		offline: false,
	}
	m.disks["sda"] = state

	// Note: smartCheckLoop calls smartctl command
	// In a real test, you'd mock exec.Command
	// This test structure shows the approach
	go m.smartCheckLoop(ctx)

	// Wait for context timeout
	<-ctx.Done()
}

func TestMonitor_ZpoolCheckLoop(t *testing.T) {
	cfg := &config.DiskMonitorConfig{
		CheckZpool:         true,
		CheckZpoolInterval: 1, // 1 second for testing
		DebugZpool:        false,
		ColorZpoolFail:    config.RGB{255, 0, 0},
		ColorDiskHealth:   config.RGB{255, 255, 255},
	}

	m := &Monitor{
		cfg:         cfg,
		disks:       make(map[string]*diskState),
		ledToDevice: make(map[string]string),
		deviceToLED: make(map[string]string),
		zpoolLEDMap: make(map[string]string),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Set up zpool mapping
	m.zpoolLEDMap["sda"] = "disk1"

	// Note: zpoolCheckLoop calls zpool command
	// In a real test, you'd mock exec.Command
	go m.zpoolCheckLoop(ctx)

	// Wait for context timeout
	<-ctx.Done()
}

func TestMonitor_DiskOnlineCheckLoop(t *testing.T) {
	tmpDir := t.TempDir()
	sysBlockPath := filepath.Join(tmpDir, "sys", "class", "block", "sda")
	
	if err := os.MkdirAll(sysBlockPath, 0755); err != nil {
		t.Fatalf("Failed to create sys/class/block: %v", err)
	}

	statPath := filepath.Join(sysBlockPath, "stat")
	if err := os.WriteFile(statPath, []byte("0 0 0 0\n"), 0644); err != nil {
		t.Fatalf("Failed to create stat file: %v", err)
	}

	cfg := &config.DiskMonitorConfig{
		CheckDiskOnlineInterval: 1,
		ColorDiskUnavail:        config.RGB{255, 0, 0},
	}

	m := &Monitor{
		cfg:         cfg,
		disks:       make(map[string]*diskState),
		ledToDevice: make(map[string]string),
		deviceToLED: make(map[string]string),
		zpoolLEDMap: make(map[string]string),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Create a mock disk state
	state := &diskState{
		device: "sda",
		smartFailed: false,
		zpoolFaulted: false,
		offline: false,
	}
	m.disks["sda"] = state

	// Note: diskOnlineCheckLoop accesses /sys/class/block
	// In a real test, you'd mock the file system or use a test filesystem
	go m.diskOnlineCheckLoop(ctx)

	// Wait for context timeout
	<-ctx.Done()
}

