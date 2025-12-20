package led

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewLED(t *testing.T) {
	led := NewLED("test-led")
	if led.name != "test-led" {
		t.Errorf("LED.name = %q, want %q", led.name, "test-led")
	}
	expectedPath := filepath.Join(sysfsLEDPath, "test-led")
	if led.path != expectedPath {
		t.Errorf("LED.path = %q, want %q", led.path, expectedPath)
	}
}

func TestLEDExists(t *testing.T) {
	tmpDir := t.TempDir()
	ledPath := filepath.Join(tmpDir, "test-led")
	
	// Create LED directory
	if err := os.MkdirAll(ledPath, 0755); err != nil {
		t.Fatalf("Failed to create LED directory: %v", err)
	}

	// Create a LED with custom path by directly constructing it
	led := &LED{
		name: "test-led",
		path: ledPath,
	}
	if !led.Exists() {
		t.Error("LED.Exists() = false, want true")
	}

	// Test non-existent LED
	led2 := &LED{
		name: "nonexistent-led",
		path: filepath.Join(tmpDir, "nonexistent-led"),
	}
	if led2.Exists() {
		t.Error("LED.Exists() = true, want false")
	}
}

func TestLEDWrite(t *testing.T) {
	tmpDir := t.TempDir()
	ledPath := filepath.Join(tmpDir, "test-led")
	
	if err := os.MkdirAll(ledPath, 0755); err != nil {
		t.Fatalf("Failed to create LED directory: %v", err)
	}

	led := &LED{
		name: "test-led",
		path: ledPath,
	}
	
	// Test writing to a file
	if err := led.Write("brightness", "128"); err != nil {
		t.Fatalf("LED.Write() error = %v", err)
	}

	// Verify the file was written
	filePath := filepath.Join(ledPath, "brightness")
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}
	if string(data) != "128" {
		t.Errorf("File content = %q, want %q", string(data), "128")
	}
}

func TestLEDRead(t *testing.T) {
	tmpDir := t.TempDir()
	ledPath := filepath.Join(tmpDir, "test-led")
	
	if err := os.MkdirAll(ledPath, 0755); err != nil {
		t.Fatalf("Failed to create LED directory: %v", err)
	}

	led := &LED{
		name: "test-led",
		path: ledPath,
	}
	
	// Write test data
	testData := "255 128 64\n"
	filePath := filepath.Join(ledPath, "color")
	if err := os.WriteFile(filePath, []byte(testData), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Test reading
	result, err := led.Read("color")
	if err != nil {
		t.Fatalf("LED.Read() error = %v", err)
	}
	expected := "255 128 64"
	if result != expected {
		t.Errorf("LED.Read() = %q, want %q", result, expected)
	}
}

func TestLEDSetColor(t *testing.T) {
	tmpDir := t.TempDir()
	ledPath := filepath.Join(tmpDir, "test-led")
	
	if err := os.MkdirAll(ledPath, 0755); err != nil {
		t.Fatalf("Failed to create LED directory: %v", err)
	}

	led := &LED{
		name: "test-led",
		path: ledPath,
	}
	
	if err := led.SetColor(255, 128, 64); err != nil {
		t.Fatalf("LED.SetColor() error = %v", err)
	}

	// Verify
	filePath := filepath.Join(ledPath, "color")
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read color file: %v", err)
	}
	expected := "255 128 64"
	if string(data) != expected {
		t.Errorf("Color = %q, want %q", string(data), expected)
	}
}

func TestLEDSetBrightness(t *testing.T) {
	tmpDir := t.TempDir()
	ledPath := filepath.Join(tmpDir, "test-led")
	
	if err := os.MkdirAll(ledPath, 0755); err != nil {
		t.Fatalf("Failed to create LED directory: %v", err)
	}

	led := &LED{
		name: "test-led",
		path: ledPath,
	}
	
	if err := led.SetBrightness(200); err != nil {
		t.Fatalf("LED.SetBrightness() error = %v", err)
	}

	// Verify
	filePath := filepath.Join(ledPath, "brightness")
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read brightness file: %v", err)
	}
	expected := "200"
	if string(data) != expected {
		t.Errorf("Brightness = %q, want %q", string(data), expected)
	}
}

func TestLEDSetTrigger(t *testing.T) {
	tmpDir := t.TempDir()
	ledPath := filepath.Join(tmpDir, "test-led")
	
	if err := os.MkdirAll(ledPath, 0755); err != nil {
		t.Fatalf("Failed to create LED directory: %v", err)
	}

	led := &LED{
		name: "test-led",
		path: ledPath,
	}
	
	if err := led.SetTrigger("oneshot"); err != nil {
		t.Fatalf("LED.SetTrigger() error = %v", err)
	}

	// Verify
	filePath := filepath.Join(ledPath, "trigger")
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read trigger file: %v", err)
	}
	expected := "oneshot"
	if string(data) != expected {
		t.Errorf("Trigger = %q, want %q", string(data), expected)
	}
}

func TestLEDTriggerShot(t *testing.T) {
	tmpDir := t.TempDir()
	ledPath := filepath.Join(tmpDir, "test-led")
	
	if err := os.MkdirAll(ledPath, 0755); err != nil {
		t.Fatalf("Failed to create LED directory: %v", err)
	}

	led := &LED{
		name: "test-led",
		path: ledPath,
	}
	
	if err := led.TriggerShot(); err != nil {
		t.Fatalf("LED.TriggerShot() error = %v", err)
	}

	// Verify
	filePath := filepath.Join(ledPath, "shot")
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read shot file: %v", err)
	}
	expected := "1"
	if string(data) != expected {
		t.Errorf("Shot = %q, want %q", string(data), expected)
	}
}

func TestLEDNetdevMethods(t *testing.T) {
	tmpDir := t.TempDir()
	ledPath := filepath.Join(tmpDir, "test-led")
	
	if err := os.MkdirAll(ledPath, 0755); err != nil {
		t.Fatalf("Failed to create LED directory: %v", err)
	}

	led := &LED{
		name: "test-led",
		path: ledPath,
	}
	
	// Test SetDeviceName
	if err := led.SetDeviceName("eth0"); err != nil {
		t.Fatalf("LED.SetDeviceName() error = %v", err)
	}
	
	// Test SetLink
	if err := led.SetLink(1); err != nil {
		t.Fatalf("LED.SetLink() error = %v", err)
	}
	
	// Test SetTx
	if err := led.SetTx(1); err != nil {
		t.Fatalf("LED.SetTx() error = %v", err)
	}
	
	// Test SetRx
	if err := led.SetRx(1); err != nil {
		t.Fatalf("LED.SetRx() error = %v", err)
	}
	
	// Test SetInterval
	if err := led.SetInterval(200); err != nil {
		t.Fatalf("LED.SetInterval() error = %v", err)
	}

	// Verify device_name
	deviceNamePath := filepath.Join(ledPath, "device_name")
	data, err := os.ReadFile(deviceNamePath)
	if err != nil {
		t.Fatalf("Failed to read device_name file: %v", err)
	}
	if string(data) != "eth0" {
		t.Errorf("device_name = %q, want %q", string(data), "eth0")
	}
}

