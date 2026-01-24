package system

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

// The URL where your Developer Server hosts the latest compiled binary
const UpdateURL = "http://your-dev-server-ip/downloads/sentinex"

// FetchAndUpgradeJumpbox downloads the new binary from your dev server.
func FetchAndUpgradeJumpbox() error {
	fmt.Printf("[*] Checking for updates from %s...\n", UpdateURL)

	resp, err := http.Get(UpdateURL)
	if err != nil {
		return fmt.Errorf("could not connect to update server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned error: %s", resp.Status)
	}

	// 1. Download to a temporary file
	tempPath := "/usr/local/bin/sentinex.tmp"
	out, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	// 2. Atomic Swap: Replace the old binary with the new one
	// This does NOT touch /etc/sentinex/ or your hosts.yml
	err = os.Rename(tempPath, "/usr/local/bin/sentinex")
	if err != nil {
		return fmt.Errorf("failed to swap binary: %v", err)
	}

	fmt.Println("[+] Jumpbox successfully upgraded to the latest build.")
	return nil
}
