package led

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const sysfsLEDPath = "/sys/class/leds"

// LED represents a single LED device
type LED struct {
	name string
	path string
}

// NewLED creates a new LED controller for the given LED name
func NewLED(name string) *LED {
	return &LED{
		name: name,
		path: filepath.Join(sysfsLEDPath, name),
	}
}

// Exists checks if the LED device exists
func (l *LED) Exists() bool {
	_, err := os.Stat(l.path)
	return err == nil
}

// Write writes a value to a sysfs file
func (l *LED) Write(file, value string) error {
	path := filepath.Join(l.path, file)
	return os.WriteFile(path, []byte(value), 0644)
}

// Read reads a value from a sysfs file
func (l *LED) Read(file string) (string, error) {
	path := filepath.Join(l.path, file)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// SetTrigger sets the LED trigger
func (l *LED) SetTrigger(trigger string) error {
	return l.Write("trigger", trigger)
}

// SetColor sets the LED color (RGB format: "r g b")
func (l *LED) SetColor(r, g, b int) error {
	return l.Write("color", fmt.Sprintf("%d %d %d", r, g, b))
}

// SetBrightness sets the LED brightness (0-255)
func (l *LED) SetBrightness(brightness int) error {
	return l.Write("brightness", fmt.Sprintf("%d", brightness))
}

// TriggerShot triggers a oneshot LED blink
func (l *LED) TriggerShot() error {
	return l.Write("shot", "1")
}

// SetInvert sets the LED invert flag
func (l *LED) SetInvert(invert int) error {
	return l.Write("invert", fmt.Sprintf("%d", invert))
}

// SetDelayOn sets the oneshot delay_on in milliseconds
func (l *LED) SetDelayOn(delay int) error {
	return l.Write("delay_on", fmt.Sprintf("%d", delay))
}

// SetDelayOff sets the oneshot delay_off in milliseconds
func (l *LED) SetDelayOff(delay int) error {
	return l.Write("delay_off", fmt.Sprintf("%d", delay))
}

// For netdev trigger
func (l *LED) SetDeviceName(name string) error {
	return l.Write("device_name", name)
}

func (l *LED) SetLink(link int) error {
	return l.Write("link", fmt.Sprintf("%d", link))
}

func (l *LED) SetTx(tx int) error {
	return l.Write("tx", fmt.Sprintf("%d", tx))
}

func (l *LED) SetRx(rx int) error {
	return l.Write("rx", fmt.Sprintf("%d", rx))
}

func (l *LED) SetInterval(interval int) error {
	return l.Write("interval", fmt.Sprintf("%d", interval))
}

