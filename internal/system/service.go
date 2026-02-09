package system

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
)

/* ========================================================================
   neurader SYSTEM CORE (v2.0.0)
   Handles Service Installation, Updates, and User Management.
   ======================================================================== */

const (
	// Replace with your Developer Server URL for the v2 update push
	UpdateURL         = "http://YOUR_DEV_SERVER_IP/neurader"
	BinaryDestination = "/usr/local/bin/neurader"
	ConfigDir         = "/etc/neurader"
	ServicePath       = "/etc/systemd/system/neurader.service"
)

/* =========================
   1. UPGRADE LOGIC
========================= */

// FetchAndUpgradeJumpbox pulls the latest compiled binary from the Dev Server.
// It uses an "Atomic Swap" to replace the binary without downtime or config loss.
func FetchAndUpgradeJumpbox() error {
	fmt.Printf("[*] Connecting to Developer Server: %s\n", UpdateURL)

	resp, err := http.Get(UpdateURL)
	if err != nil {
		return fmt.Errorf("failed to reach update server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server error: %s", resp.Status)
	}

	// Step A: Download to a temp file
	tempPath := BinaryDestination + ".tmp"
	out, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("cannot create temp file: %v", err)
	}
	
	_, err = io.Copy(out, resp.Body)
	out.Close() // Close before swapping
	if err != nil {
		return fmt.Errorf("download failed: %v", err)
	}

	// Step B: Atomic Swap (Binary replacement only)
	// This preserves /etc/neurader/hosts.yml perfectly.
	err = os.Rename(tempPath, BinaryDestination)
	if err != nil {
		return fmt.Errorf("failed to overwrite binary: %v", err)
	}

	fmt.Println("[+] Jumpbox upgraded to latest version locally.")
	return nil
}

/* =========================
   2. INSTALLATION LOGIC
========================= */

// InstallService configures neurader as a background systemd daemon.
func InstallService() {
	fmt.Println("[*] Setting up neurader as a system service...")

	// Create config directory for inventory/keys
	os.MkdirAll(ConfigDir, 0755)

	// Move binary to system path (/usr/local/bin)
	self, err := os.Executable()
	if err == nil {
		input, _ := os.ReadFile(self)
		os.WriteFile(BinaryDestination, input, 0755)
	}

	// Define systemd unit
	serviceFile := `[Unit]
Description=neurader Security Agent
After=network.target

[Service]
ExecStart=/usr/local/bin/neurader daemon
Restart=always
User=root
WorkingDirectory=/etc/neurader

[Install]
WantedBy=multi-user.target`

	// Write and activate service
	os.WriteFile(ServicePath, []byte(serviceFile), 0644)
	exec.Command("systemctl", "daemon-reload").Run()
	exec.Command("systemctl", "enable", "neurader").Run()
	exec.Command("systemctl", "start", "neurader").Run()

	fmt.Println("[+] neurader service installed and started.")
}

/* =========================
   3. USER MANAGEMENT
========================= */

// CreateneuraderUser prepares the Child nodes with a dedicated automation user.
func CreateneuraderUser() {
	fmt.Println("[*] Preparing 'neurader' user...")
	
	// Create user with a home directory for SSH keys
	exec.Command("useradd", "-m", "-s", "/bin/bash", "neurader").Run()
	
	// Prepare .ssh folder
	sshPath := "/home/neurader/.ssh"
	os.MkdirAll(sshPath, 0700)
	
	// Set ownership
	exec.Command("chown", "-R", "neurader:neurader", "/home/neurader").Run()
}
