package system

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
)

// The URL of your Linux Developer Server where the new binary is hosted
// In v2, the Jumpbox pings this URL to see if a new version is available
const UpdateURL = "http://YOUR_DEV_SERVER_IP/sentinex"

// FetchAndUpgradeJumpbox pulls the new binary from your central dev server.
func FetchAndUpgradeJumpbox() error {
	fmt.Printf("[*] Connecting to Developer Server: %s\n", UpdateURL)

	// 1. Download the new version
	resp, err := http.Get(UpdateURL)
	if err != nil {
		return fmt.Errorf("failed to reach update server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server error: %s", resp.Status)
	}

	// 2. Create a temporary file to prevent corruption
	tempPath := "/usr/local/bin/sentinex.tmp"
	out, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return fmt.Errorf("cannot create temp file: %v", err)
	}
	
	_, err = io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		return fmt.Errorf("download failed: %v", err)
	}

	// 3. Atomic Swap: Replace the old binary with the new one
	// This does NOT touch /etc/sentinex or your hosts.yml
	err = os.Rename(tempPath, "/usr/local/bin/sentinex")
	if err != nil {
		return fmt.Errorf("failed to overwrite binary: %v", err)
	}

	fmt.Println("[+] Jumpbox upgraded to latest version locally.")
	return nil
}

// InstallService handles the full setup of the binary and the systemd daemon
func InstallService() {
	fmt.Println("[*] Setting up sentinex as a system service...")

	configDir := "/etc/sentinex"
	binaryDestination := "/usr/local/bin/sentinex"
	servicePath := "/etc/systemd/system/sentinex.service"

	// 1. Ensure the configuration directory exists
	os.MkdirAll(configDir, 0755)

	// 2. Get current binary path and copy to /usr/local/bin
	self, err := os.Executable()
	if err == nil {
		input, _ := os.ReadFile(self)
		os.WriteFile(binaryDestination, input, 0755)
	}

	// 3. Define the service configuration
	serviceFile := `[Unit]
Description=sentinex Security Agent
After=network.target

[Service]
ExecStart=/usr/local/bin/sentinex daemon
Restart=always
User=root
WorkingDirectory=/etc/sentinex

[Install]
WantedBy=multi-user.target`

	// 4. Write and enable the service
	os.WriteFile(servicePath, []byte(serviceFile), 0644)
	exec.Command("systemctl", "daemon-reload").Run()
	exec.Command("systemctl", "enable", "sentinex").Run()
	exec.Command("systemctl", "start", "sentinex").Run()

	fmt.Println("[+] sentinex service installed and started.")
}

// CreatesentinexUser prepares the specialized SSH user on Child nodes
func CreatesentinexUser() {
	fmt.Println("[*] Preparing 'sentinex' user...")
	exec.Command("useradd", "-m", "-s", "/bin/bash", "sentinex").Run()
	sshPath := "/home/sentinex/.ssh"
	os.MkdirAll(sshPath, 0700)
	exec.Command("chown", "-R", "sentinex:sentinex", "/home/sentinex").Run()
}
