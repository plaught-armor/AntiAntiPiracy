package main

import (
	"fmt"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if !isAdmin() {
		fmt.Printf(`
You need to have administrator privileges.

This is required to modify or create a registry key to bypass the AKI piracy check.
The code is available for you to read and understand what is going on.

Please close this terminal and re-run the program as an Administrator.

Press Enter to exit
`)

		if _, err := fmt.Scanln(); err != nil {
			println(err.Error())
			return
		}
		return
	}

	if err := run(); err != nil {
		println(err.Error())

		fmt.Printf("\n\nPress Enter to exit")
		if _, err := fmt.Scanln(); err != nil {
			println(err.Error())
			return
		}
		return
	}

	fmt.Printf("\n\nPress Enter to exit")
	if _, err := fmt.Scanln(); err != nil {
		println(err.Error())
		return
	}
}

// run performs a series of operations to:
// 1. Open a file selector allowing the user to select the directory where EscapeFromTarkov is installed.
// 2. Checks or creates a specific registry key under `Software\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\EscapeFromTarkov`.
// 3. Verifies or updates the "InstallLocation" registry entry with the selected directory path.
// 4. Ensures the existence of specific dummy files inside the selected directory by creating them if they don't exist.
//
// Returns:
// - An error if any of the operations (registry or file creation) fails, or if no directory is selected.
func run() error {
	// The path to the registry key that will be used.
	const registryPath = `Software\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\EscapeFromTarkov`

	// Step 1: Open a file selector to choose the installation directory for EscapeFromTarkov.
	installLocation, err := openFileSelector()
	if err != nil {
		return err
	}

	fmt.Println("> Begin Registry Entry Check")

	// Step 2: Check if a registry entry for the EscapeFromTarkov installation exists.
	// Open the registry key in read-write access.
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, registryPath, registry.ALL_ACCESS)
	if err != nil {
		fmt.Println("-> Could not find Registry Key")
		// If the registry key does not exist, create it and set the "InstallLocation" to the chosen directory.
		if err := createKey(registryPath, installLocation); err != nil {
			return fmt.Errorf("failed to create Key: %v", err)
		}
	}

	// Step 3: Verify or update the "InstallLocation" in the registry.
	if value, _, err := key.GetStringValue("InstallLocation"); value != installLocation || err != nil {
		fmt.Println("-> Registry Key `InstallLocation` incorrect, resetting...")
		if err = key.SetStringValue("InstallLocation", installLocation); err != nil {
			return err
		}
		fmt.Println("-> Registry Key `InstallLocation` successfully reset")
	}
	fmt.Println("> End Registry Entry Check")

	// Step 4: Ensure the existence of specific dummy files in the chosen directory.
	dummyFiles := []string{
		"BattlEye/BEClient_x64.dll",
		"BattlEye/BEService_x64.exe",
		"ConsistencyInfo",
		"EscapeFromTarkov_BE.exe",
		"Uninstall.exe",
		"UnityCrashHandler64.exe",
	}

	fmt.Println("> Start Dummy Files Scan Process")
	for _, relativePath := range dummyFiles {
		// Combine the install location with each dummy file's relative path to form the full path.
		fullPath := filepath.Join(installLocation, relativePath)
		dir := filepath.Dir(fullPath)

		// Ensure that the directory exists, if not create it.
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return fmt.Errorf("Failed to create directory: %v\n", err)
		}

		// Check if the dummy file already exists. If it does, skip the creation.
		if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
			continue
		}

		// Create the dummy file.
		file, err := os.Create(fullPath)
		if err != nil {
			return fmt.Errorf("Failed to create file: %v\n", err)
		}
		if err := file.Close(); err != nil {
			return err
		}
		fmt.Printf("-> Created dummy file: %s\n", fullPath)
	}

	fmt.Println("> End Dummy Files Scan Process")
	return nil
}

// createKey creates a new registry key in the specified registry path and sets the "InstallLocation" value.
//
// Parameters:
// - registryPath (string): The path in the Windows registry where the key should be created.
// - installLocation (string): The value to set for the "InstallLocation" registry entry.
//
// Returns:
// - error: Returns an error if the key creation or value setting fails; otherwise, nil.
func createKey(registryPath, installLocation string) error {
	// Notify the user that the process of creating the registry key has started.
	fmt.Println("--> Creating Registry Key")

	// Attempt to create the registry key with all-access permissions.
	key, _, err := registry.CreateKey(registry.LOCAL_MACHINE, registryPath, registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("Failed to create registry key: %v\n", err)
	}
	// Ensure the key is properly closed after operations are complete.
	defer key.Close()

	// Set the "InstallLocation" value within the created registry key.
	if err := key.SetStringValue("InstallLocation", installLocation); err != nil {
		return fmt.Errorf("Failed to set InstallLocation value: %v\n", err)
	}

	// Confirm the successful creation of the registry key and value.
	fmt.Println("--> Registry Key Created Successfully")
	return nil
}

// isAdmin function checks if the currently running process has administrator privileges.
//
// This function utilizes the `windows` package to inspect the token of the current process.
// It uses the `GetCurrentProcessToken()` method to retrieve the token associated with the process,
// and then checks if the token is elevated using `IsElevated()`.
//
// Returns:
// - `true` if the current process is running with elevated (administrator) privileges.
// - `false` otherwise.
func isAdmin() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}

// openFileSelector opens a file browser dialog for the user to locate the EscapeFromTarkov installation directory.
// It validates the selected folder to ensure it contains the required "EscapeFromTarkov.exe" file.
//
// Returns:
// - string: The path of the selected directory.
// - error: An error if no directory is selected or the required file is not found.
func openFileSelector() (string, error) {
	// Notify the user about the purpose of the dialog and wait for confirmation to proceed.
	fmt.Printf(`
A browser popup will display, requesting that you locate your current EscapeFromTarkov client files.

This action is required to modify or create a registry key to bypass the AKI piracy check.
The code is available for you to read and understand what is happening.

Press Enter to continue
`)

	// Wait for the user to press Enter.
	if _, err := fmt.Scanln(); err != nil {
		return "", err
	}

	// PowerShell script to open the file browser dialog.
	// The script creates an OpenFileDialog and configures it to act as a folder selector.
	psScript := `
	Add-Type -AssemblyName System.Windows.Forms
	$FileDialog = New-Object System.Windows.Forms.OpenFileDialog
	$FileDialog.Title = "Please select the EscapeFromTarkov installation directory"
	$FileDialog.Filter = "Folders|*.none"
	$FileDialog.CheckFileExists = $false
	$FileDialog.CheckPathExists = $true
	$FileDialog.FileName = "Select Folder"
	If ($FileDialog.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) {
		Split-Path -Parent $FileDialog.FileName
	} else {
		Write-Error "No folder selected"
	}
	`

	// Execute the PowerShell script.
	cmd := exec.Command("powershell", "-NoProfile", "-Command", psScript)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("User closed: %v\n", err)
	}

	// Trim any surrounding whitespace from the output.
	selectedPath := strings.TrimSpace(string(output))
	if selectedPath == "" {
		return "", fmt.Errorf("No folder selected")
	}

	// Validate that "EscapeFromTarkov.exe" exists in the selected directory.
	exePath := filepath.Join(selectedPath, "EscapeFromTarkov.exe")
	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		return "", fmt.Errorf("Could not find a valid EscapeFromTarkov installation directory")
	}

	// Print and return the valid directory path.
	fmt.Printf("Valid directory selected: %s\n", selectedPath)

	// Return the trimmed directory path.
	return strings.TrimSpace(string(output)), nil
}
