package netmon

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/scottjab/nix-ugreen-leds-controller/internal/config"
	"github.com/scottjab/nix-ugreen-leds-controller/internal/led"
)

func Run(ctx context.Context, cfg *config.NetworkMonitorConfig, interfaceName string) error {
	// Check if we need to do anything
	if !cfg.CheckGatewayConnectivity && !cfg.CheckLinkSpeed && !cfg.CheckLinkSpeedDynamic {
		return nil
	}

	ledName := "netdev"
	l := led.NewLED(ledName)
	if !l.Exists() {
		return fmt.Errorf("LED %s does not exist", ledName)
	}

	// Initialize LED for netdev trigger
	if err := l.SetTrigger("netdev"); err != nil {
		return fmt.Errorf("failed to set netdev trigger: %w", err)
	}
	if err := l.SetDeviceName(interfaceName); err != nil {
		return fmt.Errorf("failed to set device name: %w", err)
	}
	if err := l.SetLink(1); err != nil {
		return fmt.Errorf("failed to set link: %w", err)
	}
	if err := l.SetTx(cfg.BlinkTx); err != nil {
		return fmt.Errorf("failed to set tx: %w", err)
	}
	if err := l.SetRx(cfg.BlinkRx); err != nil {
		return fmt.Errorf("failed to set rx: %w", err)
	}
	if err := l.SetInterval(cfg.BlinkInterval); err != nil {
		return fmt.Errorf("failed to set interval: %w", err)
	}
	if err := l.SetColor(cfg.ColorNormal.R, cfg.ColorNormal.G, cfg.ColorNormal.B); err != nil {
		return fmt.Errorf("failed to set color: %w", err)
	}
	if err := l.SetBrightness(cfg.BrightnessLed); err != nil {
		return fmt.Errorf("failed to set brightness: %w", err)
	}

	ticker := time.NewTicker(time.Duration(cfg.CheckInterval) * time.Second)
	defer ticker.Stop()

	gwConn := true

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			// Check gateway connectivity if enabled
			if cfg.CheckGatewayConnectivity {
				gw, err := getGateway()
				if err != nil {
					log.Printf("Failed to get gateway: %v", err)
					gwConn = false
				} else {
					gwConn = pingGateway(gw)
				}
			}

			// Set color based on state
			if !gwConn {
				// Gateway unreachable
				l.SetColor(cfg.ColorGatewayUnreachable.R, cfg.ColorGatewayUnreachable.G, cfg.ColorGatewayUnreachable.B)
			} else {
				// Set normal color based on link speed
				color := getNormalColor(cfg, interfaceName)
				l.SetColor(color.R, color.G, color.B)
			}
		}
	}
}

func getGateway() (string, error) {
	cmd := exec.Command("ip", "route")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "default") {
			fields := strings.Fields(line)
			for i, field := range fields {
				if field == "via" && i+1 < len(fields) {
					return fields[i+1], nil
				}
			}
		}
	}

	return "", fmt.Errorf("no default gateway found")
}

func pingGateway(gw string) bool {
	cmd := exec.Command("ping", "-q", "-c", "1", "-W", "1", gw)
	err := cmd.Run()
	return err == nil
}

func getNormalColor(cfg *config.NetworkMonitorConfig, interfaceName string) config.RGB {
	if cfg.CheckLinkSpeedDynamic {
		return getDynamicColor(cfg, interfaceName)
	}

	if cfg.CheckLinkSpeed {
		return getLinkSpeedColor(cfg, interfaceName)
	}

	return cfg.ColorNormal
}

func getDynamicColor(cfg *config.NetworkMonitorConfig, interfaceName string) config.RGB {
	speed, err := getLinkSpeed(interfaceName)
	if err != nil {
		return cfg.ColorNormal
	}

	// Calculate percentage
	speedLow := float64(cfg.CheckLinkSpeedDynamicSpeedLow)
	speedHigh := float64(cfg.CheckLinkSpeedDynamicSpeedHigh)
	speedFloat := float64(speed)

	if speedHigh == speedLow {
		return cfg.ColorNormal
	}

	percentage := (speedFloat - speedLow) / (speedHigh - speedLow)
	if percentage < 0 {
		percentage = 0
	}
	if percentage > 1 {
		percentage = 1
	}

	// Interpolate colors
	r := int(float64(cfg.CheckLinkSpeedDynamicColorLow.R) + percentage*float64(cfg.CheckLinkSpeedDynamicColorHigh.R-cfg.CheckLinkSpeedDynamicColorLow.R))
	g := int(float64(cfg.CheckLinkSpeedDynamicColorLow.G) + percentage*float64(cfg.CheckLinkSpeedDynamicColorHigh.G-cfg.CheckLinkSpeedDynamicColorLow.G))
	b := int(float64(cfg.CheckLinkSpeedDynamicColorLow.B) + percentage*float64(cfg.CheckLinkSpeedDynamicColorHigh.B-cfg.CheckLinkSpeedDynamicColorLow.B))

	return config.RGB{R: r, G: g, B: b}
}

func getLinkSpeedColor(cfg *config.NetworkMonitorConfig, interfaceName string) config.RGB {
	speed, err := getLinkSpeed(interfaceName)
	if err != nil {
		return cfg.ColorNormal
	}

	switch speed {
	case 100:
		if cfg.ColorLink100 != nil {
			return *cfg.ColorLink100
		}
		return cfg.ColorNormal
	case 1000:
		if cfg.ColorLink1000 != nil {
			return *cfg.ColorLink1000
		}
		return cfg.ColorNormal
	case 2000:
		if cfg.ColorLink2000 != nil {
			return *cfg.ColorLink2000
		}
		return cfg.ColorLinkPurpleDefault
	case 2500:
		if cfg.ColorLink2500 != nil {
			return *cfg.ColorLink2500
		}
		return cfg.ColorNormal
	case 5000:
		if cfg.ColorLink5000 != nil {
			return *cfg.ColorLink5000
		}
		if cfg.ColorLink10000 != nil {
			return *cfg.ColorLink10000
		}
		return cfg.ColorLinkPurpleDefault
	case 10000:
		if cfg.ColorLink10000 != nil {
			return *cfg.ColorLink10000
		}
		if cfg.ColorLink5000 != nil {
			return *cfg.ColorLink5000
		}
		return cfg.ColorLinkPurpleDefault
	default:
		return cfg.ColorNormal
	}
}

func getLinkSpeed(interfaceName string) (int, error) {
	speedPath := filepath.Join("/sys/class/net", interfaceName, "speed")
	data, err := os.ReadFile(speedPath)
	if err != nil {
		return 0, err
	}

	speed, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, err
	}

	return speed, nil
}

