package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseRGB(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected RGB
	}{
		{
			name:     "valid RGB",
			input:    "255 128 64",
			expected: RGB{R: 255, G: 128, B: 64},
		},
		{
			name:     "zero values",
			input:    "0 0 0",
			expected: RGB{R: 0, G: 0, B: 0},
		},
		{
			name:     "invalid format - too few values",
			input:    "255 128",
			expected: RGB{R: 255, G: 255, B: 255}, // default
		},
		{
			name:     "invalid format - too many values",
			input:    "255 128 64 32",
			expected: RGB{R: 255, G: 255, B: 255}, // default when len != 3
		},
		{
			name:     "empty string",
			input:    "",
			expected: RGB{R: 255, G: 255, B: 255}, // default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseRGB(tt.input)
			if result != tt.expected {
				t.Errorf("parseRGB(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRGBString(t *testing.T) {
	rgb := RGB{R: 255, G: 128, B: 64}
	expected := "255 128 64"
	if result := rgb.String(); result != expected {
		t.Errorf("RGB.String() = %q, want %q", result, expected)
	}
}

func TestSetDefaults(t *testing.T) {
	cfg := &Config{}
	cfg.setDefaults()

	// Check disk monitor defaults
	if !cfg.DiskMonitor.Enable {
		t.Error("DiskMonitor.Enable should be true by default")
	}
	if cfg.DiskMonitor.MappingMethod != "ata" {
		t.Errorf("DiskMonitor.MappingMethod = %q, want %q", cfg.DiskMonitor.MappingMethod, "ata")
	}
	if cfg.DiskMonitor.CheckSmartInterval != 360 {
		t.Errorf("DiskMonitor.CheckSmartInterval = %d, want %d", cfg.DiskMonitor.CheckSmartInterval, 360)
	}
	if cfg.DiskMonitor.ColorDiskHealth.R != 255 || cfg.DiskMonitor.ColorDiskHealth.G != 255 || cfg.DiskMonitor.ColorDiskHealth.B != 255 {
		t.Errorf("ColorDiskHealth = %v, want RGB{255, 255, 255}", cfg.DiskMonitor.ColorDiskHealth)
	}

	// Check network monitor defaults
	if cfg.NetworkMonitor.Enable {
		t.Error("NetworkMonitor.Enable should be false by default")
	}
	if len(cfg.NetworkMonitor.Interfaces) != 0 {
		t.Errorf("NetworkMonitor.Interfaces = %v, want []", cfg.NetworkMonitor.Interfaces)
	}
}

func TestLoadConfig_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nonexistent.conf")

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil", err)
	}
	if cfg == nil {
		t.Fatal("LoadConfig() returned nil config")
	}

	// Should have defaults
	if cfg.DiskMonitor.MappingMethod != "ata" {
		t.Errorf("MappingMethod = %q, want %q", cfg.DiskMonitor.MappingMethod, "ata")
	}
}

func TestLoadConfig_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.conf")

	configContent := `# Test config
DISK_MONITOR_ENABLE=true
MAPPING_METHOD=hctl
CHECK_SMART=false
CHECK_SMART_INTERVAL=180
LED_REFRESH_INTERVAL=0.5
CHECK_ZPOOL=true
COLOR_DISK_HEALTH="100 200 300"
BRIGHTNESS_DISK_LEDS=128

NETWORK_INTERFACES="eth0 eth1"
COLOR_NETDEV_NORMAL="255 0 0"
CHECK_GATEWAY_CONNECTIVITY=true
CHECK_LINK_SPEED=true
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil", err)
	}

	// Check disk monitor config
	if !cfg.DiskMonitor.Enable {
		t.Error("DiskMonitor.Enable should be true")
	}
	if cfg.DiskMonitor.MappingMethod != "hctl" {
		t.Errorf("MappingMethod = %q, want %q", cfg.DiskMonitor.MappingMethod, "hctl")
	}
	if cfg.DiskMonitor.CheckSmart {
		t.Error("CheckSmart should be false")
	}
	if cfg.DiskMonitor.CheckSmartInterval != 180 {
		t.Errorf("CheckSmartInterval = %d, want %d", cfg.DiskMonitor.CheckSmartInterval, 180)
	}
	if cfg.DiskMonitor.ColorDiskHealth.R != 100 || cfg.DiskMonitor.ColorDiskHealth.G != 200 || cfg.DiskMonitor.ColorDiskHealth.B != 300 {
		t.Errorf("ColorDiskHealth = %v, want RGB{100, 200, 300}", cfg.DiskMonitor.ColorDiskHealth)
	}

	// Check network monitor config
	if !cfg.NetworkMonitor.Enable {
		t.Error("NetworkMonitor.Enable should be true")
	}
	if len(cfg.NetworkMonitor.Interfaces) != 2 {
		t.Errorf("Interfaces = %v, want [eth0 eth1]", cfg.NetworkMonitor.Interfaces)
	}
	if cfg.NetworkMonitor.Interfaces[0] != "eth0" || cfg.NetworkMonitor.Interfaces[1] != "eth1" {
		t.Errorf("Interfaces = %v, want [eth0 eth1]", cfg.NetworkMonitor.Interfaces)
	}
	if cfg.NetworkMonitor.ColorNormal.R != 255 || cfg.NetworkMonitor.ColorNormal.G != 0 || cfg.NetworkMonitor.ColorNormal.B != 0 {
		t.Errorf("ColorNormal = %v, want RGB{255, 0, 0}", cfg.NetworkMonitor.ColorNormal)
	}
	if !cfg.NetworkMonitor.CheckGatewayConnectivity {
		t.Error("CheckGatewayConnectivity should be true")
	}
}

func TestLoadConfig_QuotedValues(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.conf")

	configContent := `MAPPING_METHOD="serial"
COLOR_DISK_HEALTH='255 128 64'
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil", err)
	}

	if cfg.DiskMonitor.MappingMethod != "serial" {
		t.Errorf("MappingMethod = %q, want %q", cfg.DiskMonitor.MappingMethod, "serial")
	}
	if cfg.DiskMonitor.ColorDiskHealth.R != 255 || cfg.DiskMonitor.ColorDiskHealth.G != 128 || cfg.DiskMonitor.ColorDiskHealth.B != 64 {
		t.Errorf("ColorDiskHealth = %v, want RGB{255, 128, 64}", cfg.DiskMonitor.ColorDiskHealth)
	}
}

func TestLoadConfig_CommentsAndEmptyLines(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.conf")

	configContent := `# This is a comment
MAPPING_METHOD=ata

# Another comment
CHECK_SMART=true
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil", err)
	}

	if cfg.DiskMonitor.MappingMethod != "ata" {
		t.Errorf("MappingMethod = %q, want %q", cfg.DiskMonitor.MappingMethod, "ata")
	}
	if !cfg.DiskMonitor.CheckSmart {
		t.Error("CheckSmart should be true")
	}
}

func TestLoadConfig_InvalidFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.conf")

	// Create a directory with the config name to simulate read error
	if err := os.MkdirAll(configPath, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("LoadConfig() error = nil, want error")
	}
}

func TestLoadConfig_BoolValues(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.conf")

	configContent := `CHECK_SMART=true
CHECK_ZPOOL=false
DEBUG_ZPOOL=true
CHECK_GATEWAY_CONNECTIVITY=false
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil", err)
	}

	if !cfg.DiskMonitor.CheckSmart {
		t.Error("CheckSmart should be true")
	}
	if cfg.DiskMonitor.CheckZpool {
		t.Error("CheckZpool should be false")
	}
	if !cfg.DiskMonitor.DebugZpool {
		t.Error("DebugZpool should be true")
	}
	if cfg.NetworkMonitor.CheckGatewayConnectivity {
		t.Error("CheckGatewayConnectivity should be false")
	}
}

func TestLoadConfig_IntValues(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.conf")

	configContent := `CHECK_SMART_INTERVAL=120
CHECK_ZPOOL_INTERVAL=10
BRIGHTNESS_DISK_LEDS=200
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil", err)
	}

	if cfg.DiskMonitor.CheckSmartInterval != 120 {
		t.Errorf("CheckSmartInterval = %d, want %d", cfg.DiskMonitor.CheckSmartInterval, 120)
	}
	if cfg.DiskMonitor.CheckZpoolInterval != 10 {
		t.Errorf("CheckZpoolInterval = %d, want %d", cfg.DiskMonitor.CheckZpoolInterval, 10)
	}
	if cfg.DiskMonitor.BrightnessDiskLeds != 200 {
		t.Errorf("BrightnessDiskLeds = %d, want %d", cfg.DiskMonitor.BrightnessDiskLeds, 200)
	}
}

func TestLoadConfig_FloatValues(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.conf")

	configContent := `LED_REFRESH_INTERVAL=0.25
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil", err)
	}

	if cfg.DiskMonitor.LedRefreshInterval != 0.25 {
		t.Errorf("LedRefreshInterval = %f, want %f", cfg.DiskMonitor.LedRefreshInterval, 0.25)
	}
}

