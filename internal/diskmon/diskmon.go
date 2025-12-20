package diskmon

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/scottjab/nix-ugreen-leds-controller/internal/config"
	"github.com/scottjab/nix-ugreen-leds-controller/internal/led"
)

type diskState struct {
	led           *led.LED
	device        string
	lastStat      string
	zpoolFaulted  bool
	smartFailed   bool
	offline       bool
	standby       bool
	mu            sync.RWMutex
}

type Monitor struct {
	cfg          *config.DiskMonitorConfig
	disks        map[string]*diskState // device -> state
	ledToDevice  map[string]string      // LED name -> device
	deviceToLED  map[string]string      // device -> LED name
	zpoolLEDMap  map[string]string      // zpool device -> LED name
	mu           sync.RWMutex
}

func Run(ctx context.Context, cfg *config.DiskMonitorConfig) error {
	m := &Monitor{
		cfg:         cfg,
		disks:       make(map[string]*diskState),
		ledToDevice: make(map[string]string),
		deviceToLED: make(map[string]string),
		zpoolLEDMap: make(map[string]string),
	}

	// Enumerate disks and initialize LEDs
	if err := m.initializeDisks(); err != nil {
		return fmt.Errorf("failed to initialize disks: %w", err)
	}

	// Build zpool mapping if enabled
	if cfg.CheckZpool {
		if err := m.buildZpoolMapping(); err != nil {
			log.Printf("Warning: Failed to build zpool mapping: %v", err)
		}
	}

	var wg sync.WaitGroup

	// Start SMART check loop
	if cfg.CheckSmart {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.smartCheckLoop(ctx)
		}()
	}

	// Start zpool check loop
	if cfg.CheckZpool {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.zpoolCheckLoop(ctx)
		}()
	}

	// Start disk online check loop
	wg.Add(1)
	go func() {
		defer wg.Done()
		m.diskOnlineCheckLoop(ctx)
	}()

	// Start I/O monitoring loop
	wg.Add(1)
	go func() {
		defer wg.Done()
		m.ioMonitorLoop(ctx)
	}()

	wg.Wait()
	return nil
}

func (m *Monitor) initializeDisks() error {
	ledMap := []string{"disk1", "disk2", "disk3", "disk4", "disk5", "disk6", "disk7", "disk8"}

	// Enumerate disks based on mapping method
	devMap, err := m.enumerateDisks()
	if err != nil {
		return err
	}

	// Get mapping array based on method
	var mapping []string
	switch m.cfg.MappingMethod {
	case "ata":
		mapping = []string{"ata1", "ata2", "ata3", "ata4", "ata5", "ata6", "ata7", "ata8"}
		// Adjust for specific models if dmidecode is available
		if productName := m.getProductName(); productName != "" {
			if strings.HasPrefix(productName, "DXP6800") {
				mapping = []string{"ata3", "ata4", "ata5", "ata6", "ata1", "ata2"}
			}
		}
	case "hctl":
		mapping = []string{"0:0:0:0", "1:0:0:0", "2:0:0:0", "3:0:0:0", "4:0:0:0", "5:0:0:0", "6:0:0:0", "7:0:0:0"}
		if productName := m.getProductName(); productName != "" {
			if strings.HasPrefix(productName, "DXP6800") {
				mapping = []string{"2:0:0:0", "3:0:0:0", "4:0:0:0", "5:0:0:0", "0:0:0:0", "1:0:0:0"}
			}
		}
	case "serial":
		// Serial mapping comes from environment variable DISK_SERIAL
		serialEnv := os.Getenv("DISK_SERIAL")
		if serialEnv != "" {
			mapping = strings.Fields(serialEnv)
		} else {
			return fmt.Errorf("serial mapping method requires DISK_SERIAL environment variable")
		}
	default:
		return fmt.Errorf("unsupported mapping method: %s", m.cfg.MappingMethod)
	}

	// Initialize LEDs
	for i, ledName := range ledMap {
		if i >= len(mapping) {
			break
		}

		l := led.NewLED(ledName)
		if !l.Exists() {
			continue
		}

		// Initialize LED
		if err := l.SetTrigger("oneshot"); err != nil {
			log.Printf("Warning: Failed to set trigger for %s: %v", ledName, err)
			continue
		}
		l.SetInvert(1)
		l.SetDelayOn(100)
		l.SetDelayOff(100)
		l.SetColor(m.cfg.ColorDiskHealth.R, m.cfg.ColorDiskHealth.G, m.cfg.ColorDiskHealth.B)
		l.SetBrightness(m.cfg.BrightnessDiskLeds)

		// Find corresponding device
		key := mapping[i]
		device, ok := devMap[key]
		if !ok {
			// No disk in this slot
			l.SetBrightness(0)
			l.SetTrigger("none")
			continue
		}

		// Check if device exists
		if _, err := os.Stat(filepath.Join("/sys/class/block", device, "stat")); err != nil {
			// Device doesn't exist
			l.SetBrightness(0)
			l.SetTrigger("none")
			continue
		}

		// Store mappings
		m.mu.Lock()
		m.ledToDevice[ledName] = device
		m.deviceToLED[device] = ledName
		m.disks[device] = &diskState{
			led:    l,
			device: device,
		}
		m.mu.Unlock()

		log.Printf("Mapped %s -> %s -> %s", m.cfg.MappingMethod, key, device)
	}

	return nil
}

func (m *Monitor) enumerateDisks() (map[string]string, error) {
	devMap := make(map[string]string)

	switch m.cfg.MappingMethod {
	case "ata":
		// List /sys/block and find ata devices
		entries, err := os.ReadDir("/sys/block")
		if err != nil {
			return nil, err
		}

		ataRegex := regexp.MustCompile(`ata\d+`)
		for _, entry := range entries {
			linkPath := filepath.Join("/sys/block", entry.Name())
			linkTarget, err := os.Readlink(linkPath)
			if err != nil {
				continue
			}

			matches := ataRegex.FindString(linkTarget)
			if matches != "" {
				devMap[matches] = entry.Name()
			}
		}

	case "hctl", "serial":
		// Use lsblk to enumerate
		cmd := exec.Command("lsblk", "-S", "-o", "name,"+m.cfg.MappingMethod+",tran")
		output, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("failed to run lsblk: %w", err)
		}

		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if !strings.Contains(line, "sata") {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				devMap[fields[1]] = fields[0]
			}
		}
	}

	return devMap, nil
}

func (m *Monitor) getProductName() string {
	cmd := exec.Command("dmidecode", "--string", "system-product-name")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func (m *Monitor) buildZpoolMapping() error {
	cmd := exec.Command("zpool", "status", "-L")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to run zpool status: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "sd") && !strings.HasPrefix(line, "dm") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		zpoolDev := fields[0]
		baseDev := regexp.MustCompile(`\d+$`).ReplaceAllString(zpoolDev, "")

		m.mu.RLock()
		ledName, ok := m.deviceToLED[baseDev]
		m.mu.RUnlock()

		if ok {
			m.mu.Lock()
			m.zpoolLEDMap[zpoolDev] = ledName
			if zpoolDev != baseDev {
				m.zpoolLEDMap[baseDev] = ledName
			}
			m.mu.Unlock()
			if m.cfg.DebugZpool {
				log.Printf("zpool device %s -> %s -> LED: %s", zpoolDev, baseDev, ledName)
			}
		}
	}

	return nil
}

func (m *Monitor) smartCheckLoop(ctx context.Context) {
	interval := m.cfg.CheckSmartInterval
	if interval <= 0 {
		interval = 360 // Default to 360 seconds if invalid
	}
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.checkSMART()
		}
	}
}

func (m *Monitor) checkSMART() {
	m.mu.RLock()
	disks := make([]*diskState, 0, len(m.disks))
	for _, state := range m.disks {
		disks = append(disks, state)
	}
	m.mu.RUnlock()

	for _, state := range disks {
		state.mu.RLock()
		ledColor := state.led
		isHealthy := !state.smartFailed && !state.zpoolFaulted && !state.offline
		device := state.device
		state.mu.RUnlock()

		if !isHealthy {
			continue // Skip if already in error state
		}

		// Run smartctl
		cmd := exec.Command("smartctl", "-H", "/dev/"+device, "-n", "standby,0")
		err := cmd.Run()
		ret := 0
		if err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				ret = exitError.ExitCode()
			}
		}

		// Check return code (bit 5 is standby, ignore it)
		if ret&^32 != 0 {
			state.mu.Lock()
			state.smartFailed = true
			state.mu.Unlock()

			ledColor.SetColor(m.cfg.ColorSmartFail.R, m.cfg.ColorSmartFail.G, m.cfg.ColorSmartFail.B)
			log.Printf("SMART Disk failure detected on /dev/%s at %s", device, time.Now().Format("2006-01-02 15:04:05"))
		}
	}
}

func (m *Monitor) zpoolCheckLoop(ctx context.Context) {
	interval := m.cfg.CheckZpoolInterval
	if interval <= 0 {
		interval = 5 // Default to 5 seconds if invalid
	}
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	faultedLogged := make(map[string]bool)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.checkZpool(faultedLogged)
		}
	}
}

func (m *Monitor) checkZpool(faultedLogged map[string]bool) {
	cmd := exec.Command("zpool", "status", "-L")
	output, err := cmd.Output()
	if err != nil {
		return
	}

	lines := strings.Split(string(output), "\n")
	seenDevices := make(map[string]bool)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "sd") && !strings.HasPrefix(line, "dm") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		zpoolDev := fields[0]
		state := strings.TrimSpace(fields[1])
		seenDevices[zpoolDev] = true

		// Find LED for this device
		m.mu.RLock()
		ledName, ok := m.zpoolLEDMap[zpoolDev]
		if !ok {
			baseDev := regexp.MustCompile(`\d+$`).ReplaceAllString(zpoolDev, "")
			ledName, ok = m.zpoolLEDMap[baseDev]
			if !ok {
				ledName, ok = m.deviceToLED[baseDev]
			}
		}
		m.mu.RUnlock()

		if !ok {
			if m.cfg.DebugZpool {
				log.Printf("WARNING: ZPOOL device /dev/%s not found in LED mapping", zpoolDev)
			}
			continue
		}

		l := led.NewLED(ledName)
		currentColor, _ := l.Read("color")

		switch state {
		case "OFFLINE", "FAULTED", "UNAVAIL", "REMOVED", "CORRUPT":
			// Set to failure color
			l.SetColor(m.cfg.ColorZpoolFail.R, m.cfg.ColorZpoolFail.G, m.cfg.ColorZpoolFail.B)

			// Log once per faulted device
			if !faultedLogged[zpoolDev] {
				if m.cfg.DebugZpool {
					log.Printf("ZPOOL Disk failure detected on /dev/%s (state: %s) -> LED: %s at %s", zpoolDev, state, ledName, time.Now().Format("2006-01-02 15:04:05"))
				} else {
					log.Printf("ZPOOL Disk failure detected on /dev/%s (state: %s) at %s", zpoolDev, state, time.Now().Format("2006-01-02 15:04:05"))
				}
				faultedLogged[zpoolDev] = true
			}

		case "ONLINE", "AVAIL", "DEGRADED":
			// Reset if it was previously faulted
			if currentColor == fmt.Sprintf("%d %d %d", m.cfg.ColorZpoolFail.R, m.cfg.ColorZpoolFail.G, m.cfg.ColorZpoolFail.B) {
				l.SetColor(m.cfg.ColorDiskHealth.R, m.cfg.ColorDiskHealth.G, m.cfg.ColorDiskHealth.B)
				if m.cfg.DebugZpool {
					log.Printf("ZPOOL Disk /dev/%s recovered (state: %s) at %s", zpoolDev, state, time.Now().Format("2006-01-02 15:04:05"))
				}
			}
			delete(faultedLogged, zpoolDev)
		}
	}
}

func (m *Monitor) diskOnlineCheckLoop(ctx context.Context) {
	interval := m.cfg.CheckDiskOnlineInterval
	if interval <= 0 {
		interval = 5 // Default to 5 seconds if invalid
	}
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.checkDiskOnline()
		}
	}
}

func (m *Monitor) checkDiskOnline() {
	m.mu.RLock()
	disks := make([]*diskState, 0, len(m.disks))
	for _, state := range m.disks {
		disks = append(disks, state)
	}
	m.mu.RUnlock()

	for _, state := range disks {
		state.mu.RLock()
		isHealthy := !state.smartFailed && !state.zpoolFaulted && !state.offline
		device := state.device
		ledColor := state.led
		state.mu.RUnlock()

		if !isHealthy {
			continue
		}

		// Check if device still exists
		if _, err := os.Stat(filepath.Join("/sys/class/block", device, "stat")); err != nil {
			state.mu.Lock()
			state.offline = true
			state.mu.Unlock()

			ledColor.SetColor(m.cfg.ColorDiskUnavail.R, m.cfg.ColorDiskUnavail.G, m.cfg.ColorDiskUnavail.B)
			log.Printf("Disk /dev/%s went offline at %s", device, time.Now().Format("2006-01-02 15:04:05"))
		}
	}
}

func (m *Monitor) ioMonitorLoop(ctx context.Context) {
	interval := m.cfg.LedRefreshInterval
	if interval <= 0 {
		interval = 0.1 // Default to 0.1 seconds if invalid
	}
	ticker := time.NewTicker(time.Duration(interval * float64(time.Second)))
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.checkIO()
		}
	}
}

func (m *Monitor) checkIO() {
	m.mu.RLock()
	disks := make([]*diskState, 0, len(m.disks))
	for _, state := range m.disks {
		disks = append(disks, state)
	}
	m.mu.RUnlock()

	for _, state := range disks {
		state.mu.RLock()
		isHealthy := !state.smartFailed && !state.zpoolFaulted && !state.offline
		device := state.device
		lastStat := state.lastStat
		state.mu.RUnlock()

		if !isHealthy {
			continue
		}

		// Read current stat
		statPath := filepath.Join("/sys/class/block", device, "stat")
		newStat, err := os.ReadFile(statPath)
		if err != nil {
			continue
		}

		newStatStr := string(newStat)
		if newStatStr != lastStat {
			// I/O activity detected
			state.mu.Lock()
			state.lastStat = newStatStr
			state.mu.Unlock()

			state.led.TriggerShot()
		}
	}
}

