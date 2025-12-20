package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type RGB struct {
	R, G, B int
}

func (r RGB) String() string {
	return fmt.Sprintf("%d %d %d", r.R, r.G, r.B)
}

func parseRGB(s string) RGB {
	parts := strings.Fields(s)
	if len(parts) != 3 {
		return RGB{255, 255, 255}
	}
	r, _ := strconv.Atoi(parts[0])
	g, _ := strconv.Atoi(parts[1])
	b, _ := strconv.Atoi(parts[2])
	return RGB{R: r, G: g, B: b}
}

type DiskMonitorConfig struct {
	Enable                bool
	MappingMethod         string // "ata", "hctl", "serial"
	CheckSmart            bool
	CheckSmartInterval    int // seconds
	LedRefreshInterval    float64 // seconds
	CheckZpool            bool
	CheckZpoolInterval    int // seconds
	DebugZpool            bool
	CheckDiskOnlineInterval int // seconds
	ColorDiskHealth       RGB
	ColorDiskUnavail      RGB
	ColorDiskStandby      RGB
	ColorZpoolFail        RGB
	ColorSmartFail        RGB
	BrightnessDiskLeds    int
	StandbyMonPath        string
	StandbyCheckInterval  int
	BlinkMonPath          string
}

type NetworkMonitorConfig struct {
	Enable                      bool
	Interfaces                  []string
	ColorNormal                 RGB
	ColorGatewayUnreachable     RGB
	ColorLinkPurpleDefault      RGB
	ColorLink100                *RGB
	ColorLink1000               *RGB
	ColorLink2000               *RGB
	ColorLink2500               *RGB
	ColorLink5000               *RGB
	ColorLink10000              *RGB
	BrightnessLed               int
	CheckInterval               int // seconds
	CheckGatewayConnectivity    bool
	CheckLinkSpeed              bool
	CheckLinkSpeedDynamic       bool
	CheckLinkSpeedDynamicColorLow  RGB
	CheckLinkSpeedDynamicColorHigh RGB
	CheckLinkSpeedDynamicSpeedLow  int // Mbps
	CheckLinkSpeedDynamicSpeedHigh int // Mbps
	BlinkTx                     int
	BlinkRx                     int
	BlinkInterval               int // milliseconds
}

type Config struct {
	DiskMonitor    DiskMonitorConfig
	NetworkMonitor NetworkMonitorConfig
}

func (c *Config) setDefaults() {
	// Set hardcoded defaults
	c.DiskMonitor.Enable = true
	c.DiskMonitor.MappingMethod = "ata"
	c.DiskMonitor.CheckSmart = true
	c.DiskMonitor.CheckSmartInterval = 360
	c.DiskMonitor.LedRefreshInterval = 0.1
	c.DiskMonitor.CheckZpool = false
	c.DiskMonitor.CheckZpoolInterval = 5
	c.DiskMonitor.DebugZpool = false
	c.DiskMonitor.CheckDiskOnlineInterval = 5
	c.DiskMonitor.ColorDiskHealth = RGB{255, 255, 255}
	c.DiskMonitor.ColorDiskUnavail = RGB{255, 0, 0}
	c.DiskMonitor.ColorDiskStandby = RGB{0, 0, 255}
	c.DiskMonitor.ColorZpoolFail = RGB{255, 0, 0}
	c.DiskMonitor.ColorSmartFail = RGB{255, 0, 0}
	c.DiskMonitor.BrightnessDiskLeds = 255
	c.DiskMonitor.StandbyMonPath = "/usr/bin/ugreen-check-standby"
	c.DiskMonitor.StandbyCheckInterval = 1
	c.DiskMonitor.BlinkMonPath = "/usr/bin/ugreen-blink-disk"

	c.NetworkMonitor.Enable = false
	c.NetworkMonitor.Interfaces = []string{}
	c.NetworkMonitor.ColorNormal = RGB{255, 255, 255}
	c.NetworkMonitor.ColorGatewayUnreachable = RGB{255, 0, 0}
	c.NetworkMonitor.ColorLinkPurpleDefault = RGB{128, 0, 128}
	c.NetworkMonitor.BrightnessLed = 255
	c.NetworkMonitor.CheckInterval = 60
	c.NetworkMonitor.CheckGatewayConnectivity = false
	c.NetworkMonitor.CheckLinkSpeed = false
	c.NetworkMonitor.CheckLinkSpeedDynamic = false
	c.NetworkMonitor.CheckLinkSpeedDynamicColorLow = RGB{255, 0, 0}
	c.NetworkMonitor.CheckLinkSpeedDynamicColorHigh = RGB{0, 255, 0}
	c.NetworkMonitor.CheckLinkSpeedDynamicSpeedLow = 0
	c.NetworkMonitor.CheckLinkSpeedDynamicSpeedHigh = 10000
	c.NetworkMonitor.BlinkTx = 1
	c.NetworkMonitor.BlinkRx = 1
	c.NetworkMonitor.BlinkInterval = 200
}

func (c *Config) SetDefaults() {
	c.setDefaults()
}

func LoadConfig(path string) (*Config, error) {
	cfg := &Config{}
	cfg.setDefaults()

	// Load from config file if it exists
	// The config file format is shell-style variable assignments (KEY=VALUE)
	if _, err := os.Stat(path); err != nil {
		// Config file doesn't exist, return defaults
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse shell-style config file and apply values directly
	configMap := make(map[string]string)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		// Remove quotes if present
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}

		configMap[key] = value
	}

	// Apply config values
	getValue := func(key string) string {
		if v, ok := configMap[key]; ok {
			return v
		}
		return ""
	}

	getBool := func(key string, defaultValue bool) bool {
		if v := getValue(key); v != "" {
			return v == "true"
		}
		return defaultValue
	}

	getInt := func(key string, defaultValue int) int {
		if v := getValue(key); v != "" {
			if i, err := strconv.Atoi(v); err == nil {
				return i
			}
		}
		return defaultValue
	}

	getFloat := func(key string, defaultValue float64) float64 {
		if v := getValue(key); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return f
			}
		}
		return defaultValue
	}

	// Disk monitor config
	cfg.DiskMonitor.Enable = getBool("DISK_MONITOR_ENABLE", cfg.DiskMonitor.Enable)
	cfg.DiskMonitor.MappingMethod = getValue("MAPPING_METHOD")
	if cfg.DiskMonitor.MappingMethod == "" {
		cfg.DiskMonitor.MappingMethod = "ata"
	}
	cfg.DiskMonitor.CheckSmart = getBool("CHECK_SMART", cfg.DiskMonitor.CheckSmart)
	cfg.DiskMonitor.CheckSmartInterval = getInt("CHECK_SMART_INTERVAL", cfg.DiskMonitor.CheckSmartInterval)
	cfg.DiskMonitor.LedRefreshInterval = getFloat("LED_REFRESH_INTERVAL", cfg.DiskMonitor.LedRefreshInterval)
	cfg.DiskMonitor.CheckZpool = getBool("CHECK_ZPOOL", cfg.DiskMonitor.CheckZpool)
	cfg.DiskMonitor.CheckZpoolInterval = getInt("CHECK_ZPOOL_INTERVAL", cfg.DiskMonitor.CheckZpoolInterval)
	cfg.DiskMonitor.DebugZpool = getBool("DEBUG_ZPOOL", cfg.DiskMonitor.DebugZpool)
	cfg.DiskMonitor.CheckDiskOnlineInterval = getInt("CHECK_DISK_ONLINE_INTERVAL", cfg.DiskMonitor.CheckDiskOnlineInterval)
	if v := getValue("COLOR_DISK_HEALTH"); v != "" {
		cfg.DiskMonitor.ColorDiskHealth = parseRGB(v)
	}
	if v := getValue("COLOR_DISK_UNAVAIL"); v != "" {
		cfg.DiskMonitor.ColorDiskUnavail = parseRGB(v)
	}
	if v := getValue("COLOR_DISK_STANDBY"); v != "" {
		cfg.DiskMonitor.ColorDiskStandby = parseRGB(v)
	}
	if v := getValue("COLOR_ZPOOL_FAIL"); v != "" {
		cfg.DiskMonitor.ColorZpoolFail = parseRGB(v)
	}
	if v := getValue("COLOR_SMART_FAIL"); v != "" {
		cfg.DiskMonitor.ColorSmartFail = parseRGB(v)
	}
	cfg.DiskMonitor.BrightnessDiskLeds = getInt("BRIGHTNESS_DISK_LEDS", cfg.DiskMonitor.BrightnessDiskLeds)
	cfg.DiskMonitor.StandbyMonPath = getValue("STANDBY_MON_PATH")
	if cfg.DiskMonitor.StandbyMonPath == "" {
		cfg.DiskMonitor.StandbyMonPath = "/usr/bin/ugreen-check-standby"
	}
	cfg.DiskMonitor.StandbyCheckInterval = getInt("STANDBY_CHECK_INTERVAL", cfg.DiskMonitor.StandbyCheckInterval)
	cfg.DiskMonitor.BlinkMonPath = getValue("BLINK_MON_PATH")
	if cfg.DiskMonitor.BlinkMonPath == "" {
		cfg.DiskMonitor.BlinkMonPath = "/usr/bin/ugreen-blink-disk"
	}

	// Network monitor config
	if v := getValue("NETWORK_INTERFACES"); v != "" {
		cfg.NetworkMonitor.Interfaces = strings.Fields(v)
		cfg.NetworkMonitor.Enable = len(cfg.NetworkMonitor.Interfaces) > 0
	}
	if v := getValue("COLOR_NETDEV_NORMAL"); v != "" {
		cfg.NetworkMonitor.ColorNormal = parseRGB(v)
	}
	if v := getValue("COLOR_NETDEV_GATEWAY_UNREACHABLE"); v != "" {
		cfg.NetworkMonitor.ColorGatewayUnreachable = parseRGB(v)
	}
	if v := getValue("COLOR_NETDEV_LINK_PURPLE_DEFAULT"); v != "" {
		cfg.NetworkMonitor.ColorLinkPurpleDefault = parseRGB(v)
	}
	if v := getValue("COLOR_NETDEV_LINK_100"); v != "" {
		rgb := parseRGB(v)
		cfg.NetworkMonitor.ColorLink100 = &rgb
	}
	if v := getValue("COLOR_NETDEV_LINK_1000"); v != "" {
		rgb := parseRGB(v)
		cfg.NetworkMonitor.ColorLink1000 = &rgb
	}
	if v := getValue("COLOR_NETDEV_LINK_2000"); v != "" {
		rgb := parseRGB(v)
		cfg.NetworkMonitor.ColorLink2000 = &rgb
	}
	if v := getValue("COLOR_NETDEV_LINK_2500"); v != "" {
		rgb := parseRGB(v)
		cfg.NetworkMonitor.ColorLink2500 = &rgb
	}
	if v := getValue("COLOR_NETDEV_LINK_5000"); v != "" {
		rgb := parseRGB(v)
		cfg.NetworkMonitor.ColorLink5000 = &rgb
	}
	if v := getValue("COLOR_NETDEV_LINK_10000"); v != "" {
		rgb := parseRGB(v)
		cfg.NetworkMonitor.ColorLink10000 = &rgb
	}
	cfg.NetworkMonitor.BrightnessLed = getInt("BRIGHTNESS_NETDEV_LED", cfg.NetworkMonitor.BrightnessLed)
	cfg.NetworkMonitor.CheckInterval = getInt("CHECK_NETDEV_INTERVAL", cfg.NetworkMonitor.CheckInterval)
	cfg.NetworkMonitor.CheckGatewayConnectivity = getBool("CHECK_GATEWAY_CONNECTIVITY", cfg.NetworkMonitor.CheckGatewayConnectivity)
	cfg.NetworkMonitor.CheckLinkSpeed = getBool("CHECK_LINK_SPEED", cfg.NetworkMonitor.CheckLinkSpeed)
	cfg.NetworkMonitor.CheckLinkSpeedDynamic = getBool("CHECK_LINK_SPEED_DYNAMIC", cfg.NetworkMonitor.CheckLinkSpeedDynamic)
	if v := getValue("CHECK_LINK_SPEED_DYNAMIC_COLOR_LOW"); v != "" {
		cfg.NetworkMonitor.CheckLinkSpeedDynamicColorLow = parseRGB(v)
	}
	if v := getValue("CHECK_LINK_SPEED_DYNAMIC_COLOR_HIGH"); v != "" {
		cfg.NetworkMonitor.CheckLinkSpeedDynamicColorHigh = parseRGB(v)
	}
	cfg.NetworkMonitor.CheckLinkSpeedDynamicSpeedLow = getInt("CHECK_LINK_SPEED_DYNAMIC_SPEED_LOW", cfg.NetworkMonitor.CheckLinkSpeedDynamicSpeedLow)
	cfg.NetworkMonitor.CheckLinkSpeedDynamicSpeedHigh = getInt("CHECK_LINK_SPEED_DYNAMIC_SPEED_HIGH", cfg.NetworkMonitor.CheckLinkSpeedDynamicSpeedHigh)
	cfg.NetworkMonitor.BlinkTx = getInt("NETDEV_BLINK_TX", cfg.NetworkMonitor.BlinkTx)
	cfg.NetworkMonitor.BlinkRx = getInt("NETDEV_BLINK_RX", cfg.NetworkMonitor.BlinkRx)
	cfg.NetworkMonitor.BlinkInterval = getInt("NETDEV_BLINK_INTERVAL", cfg.NetworkMonitor.BlinkInterval)

	return cfg, nil
}

