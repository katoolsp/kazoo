package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

const (
	baseURL        = "https://www.katools.org/kazoo/packages"
	installDir     = "/usr/local/bin"
	defaultVersion = "latest"
	packageRegistry = "/.kazoo/packages"
)

type PkgReq struct {
	Version string `json:"version"`
	OS      string `json:"os"`
	Arch    string `json:"architecture"`
}

type Action int

const (
	Install Action = iota
	Remove
	Update
	Version
)

type Cmd struct {
	Action  Action
	Package string
	Version string
}

type InstPkg struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Version string `json:"version"`
}

func main() {
	err := ensureKazooRegistered()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cmd, err := parseArgs()
	if err != nil {
		fmt.Println("Error:", err)
		help()
		os.Exit(1)
	}

	switch cmd.Action {
		case Install:
			err = instPkg(cmd.Package, cmd.Version)
		case Remove:
			err = rmPkg(cmd.Package, false)
		case Update:
			err = updPkg(cmd.Package, cmd.Version)
		case Version:
			sv()
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func ensureKazooRegistered() error {
	installedPackages, err := readInstPkgs()
	if err != nil {
		return fmt.Errorf("failed to read package registry: %v", err)
	}

	if _, exists := installedPackages["kazoo"]; !exists {
		kazooPath := filepath.Join(installDir, "kazoo")
		if _, err := os.Stat(kazooPath); os.IsNotExist(err) {
			return nil
		}

		installedPackages["kazoo"] = InstPkg{
			Name:    "kazoo",
			Path:    kazooPath,
			Version: defaultVersion,
		}

		err = writeInstPkgs(installedPackages)
		if err != nil {
			return fmt.Errorf("failed to register kazoo: %v", err)
		}
		fmt.Println("Registered kazoo in package registry")
	}
	return nil
}

func parseArgs() (*Cmd, error) {
	if len(os.Args) == 1 {
		return &Cmd{Action: Version}, nil
	}
	if len(os.Args) < 2 {
		return nil, fmt.Errorf("Usage: kazoo -i|-r|-u <package> [-v version]")
	}

	var cmd Cmd

	switch os.Args[1] {
	case "-i":
		cmd.Action = Install
	case "-r":
		cmd.Action = Remove
	case "-u":
		cmd.Action = Update
	default:
		return nil, fmt.Errorf("Unknown action: %s", os.Args[1])
	}

	if cmd.Action == Update {
		// Default to "kazoo" if no package is specified for update
		if len(os.Args) < 3 {
			cmd.Package = "kazoo"
		} else {
			cmd.Package = os.Args[2]
		}
		if len(os.Args) > 3 && os.Args[3] == "-v" && len(os.Args) > 4 {
			cmd.Version = os.Args[4]
		} else {
			cmd.Version = defaultVersion
		}
	} else if cmd.Action != Version {
		if len(os.Args) < 3 {
			return nil, fmt.Errorf("Usage: kazoo -i|-r|-u <package> [-v version]")
		}
		cmd.Package = os.Args[2]
		if len(os.Args) > 3 && os.Args[3] == "-v" && len(os.Args) > 4 {
			cmd.Version = os.Args[4]
		} else {
			cmd.Version = defaultVersion
		}
	}

	return &cmd, nil
}

func actToStr(action Action) string {
	switch action {
		case Install:
			return "Install"
		case Remove:
			return "Remove"
		case Update:
			return "Update"
		case Version:
			return "Version"
		default:
			return "Unknown"
	}
}

func help() {
	fmt.Println("Usage: kazoo -i|-r|-u <package> [-v version]")
	fmt.Println("  -i <package>       Install the specified package")
	fmt.Println("  -r <package>       Remove the specified package")
	fmt.Println("  -u <package>       Update the specified package")
}

func instPkg(packageName, version string) error {
	fmt.Printf("Requesting %s v%s for %s/%s...\n", packageName, version, runtime.GOOS, runtime.GOARCH)
	err := reqAndDlPkg(packageName, version)
	if err != nil {
		return fmt.Errorf("Error downloading package: %v", err)
	}

	installPath := filepath.Join(installDir, packageName)
	err = instPkgBin(packageName, installPath)
	if err != nil {
		return fmt.Errorf("Error installing package: %v", err)
	}

	err = updPkgReg(packageName, installPath, version)
	if err != nil {
		return fmt.Errorf("Error updating package registry: %v", err)
	}

	fmt.Printf("Successfully installed %s v%s\n", packageName, version)
	return nil
}

func reqAndDlPkg(packageName, version string) error {
	url := fmt.Sprintf("%s/%s/", baseURL, packageName)
	reqBody := PkgReq{
		Version: version,
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
	}
	fmt.Printf("Sending request to %s\n", url)
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("Response Status: %s\n", resp.Status)
	fmt.Printf("Content-Type: %s\n", resp.Header.Get("Content-Type"))

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to download package: HTTP %d, body: %s", resp.StatusCode, string(body))
	}

	if resp.Header.Get("Content-Type") != "application/octet-stream" {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected content type: %s, body: %s", resp.Header.Get("Content-Type"), string(body))
	}

	tempFile := fmt.Sprintf("%s_%s_%s", packageName, runtime.GOOS, runtime.GOARCH)
	out, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	defer out.Close()

	n, err := io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save binary: %v, bytes written: %d", err, n)
	}
	fmt.Printf("Wrote %d bytes to %s\n", n, tempFile)

	if runtime.GOOS != "windows" {
		err = os.Chmod(tempFile, 0755)
		if err != nil {
			return fmt.Errorf("failed to set executable permissions: %v", err)
		}
	}

	return nil
}

func instPkgBin(packageName, installPath string) error {
	err := os.MkdirAll(installDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create install directory: %v", err)
	}

	srcFileName := fmt.Sprintf("%s_%s_%s", packageName, runtime.GOOS, runtime.GOARCH)
	srcFile, err := os.Open(srcFileName)
	if err != nil {
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(installPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %v", err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %v", err)
	}

	if runtime.GOOS != "windows" {
		err = os.Chmod(installPath, 0755)
		if err != nil {
			return fmt.Errorf("failed to set executable permissions: %v", err)
		}
	}

	err = os.Remove(srcFileName)
	if err != nil {
		return fmt.Errorf("failed to remove source file: %v", err)
	}

	return nil
}

func updPkgReg(packageName, installPath, version string) error {
	installedPackages, err := readInstPkgs()
	if err != nil {
		return fmt.Errorf("failed to read installed packages: %v", err)
	}

	installedPackages[packageName] = InstPkg{
		Name:    packageName,
		Path:    installPath,
		Version: version,
	}

	err = writeInstPkgs(installedPackages)
	if err != nil {
		return fmt.Errorf("failed to write installed packages: %v", err)
	}

	return nil
}

func readInstPkgs() (map[string]InstPkg, error) {
	usrHomeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %v", err)
	}
	pkgRegPath := filepath.Join(usrHomeDir, packageRegistry)

	if _, err := os.Stat(pkgRegPath); os.IsNotExist(err) {
		return make(map[string]InstPkg), nil
	}

	file, err := os.Open(pkgRegPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open registry file: %v", err)
	}
	defer file.Close()

	var installedPackages map[string]InstPkg
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&installedPackages)
	if err != nil {
		return nil, fmt.Errorf("failed to decode registry file: %v", err)
	}

	return installedPackages, nil
}

func writeInstPkgs(installedPackages map[string]InstPkg) error {
	usrHomeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %v", err)
	}
	pkgRegPath := filepath.Join(usrHomeDir, packageRegistry)

	pkgRegDir := filepath.Dir(pkgRegPath)
	err = os.MkdirAll(pkgRegDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create package registry directory: %v", err)
	}

	file, err := os.Create(pkgRegPath)
	if err != nil {
		return fmt.Errorf("failed to create registry file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(installedPackages)
	if err != nil {
		return fmt.Errorf("failed to encode installed packages: %v", err)
	}

	return nil
}

func rmPkg(packageName string, isUpdate bool) error {
	installedPackages, err := readInstPkgs()
	if err != nil {
		return fmt.Errorf("failed to read installed packages: %v", err)
	}

	installedPackage, exists := installedPackages[packageName]
	if !exists {
		return fmt.Errorf("package %s is not installed", packageName)
	}

	var backupPath string
	if isUpdate {
		backupPath = fmt.Sprintf("%s.bak", installedPackage.Path)
		err = os.Rename(installedPackage.Path, backupPath)
		if err != nil {
			return fmt.Errorf("failed to backup package: %v", err)
		}
		fmt.Printf("Backed up %s to %s\n", packageName, backupPath)
	} else {
		err = os.Remove(installedPackage.Path)
		if err != nil {
			return fmt.Errorf("failed to remove package: %v", err)
		}
	}

	delete(installedPackages, packageName)

	err = writeInstPkgs(installedPackages)
	if err != nil {
		if isUpdate {
			os.Rename(backupPath, installedPackage.Path) // Restore backup on registry failure
			fmt.Printf("Restored %s from backup due to registry update failure\n", packageName)
		}
		return fmt.Errorf("failed to update package registry: %v", err)
	}

	if !isUpdate {
		fmt.Printf("Successfully removed %s\n", packageName)
	}

	if packageName == "kazoo" && !isUpdate {
		fmt.Println("Removing Kazoo itself. Exiting...")
		os.Exit(0)
	}

	return nil
}

func updPkg(packageName, version string) error {
	fmt.Printf("Updating %s to version %s...\n", packageName, version)
	err := rmPkg(packageName, true)
	if err != nil {
		return fmt.Errorf("failed to remove package before updating: %v", err)
	}

	err = instPkg(packageName, version)
	if err != nil {
		// Attempt to restore backup
		backupPath := fmt.Sprintf("%s.bak", filepath.Join(installDir, packageName))
		originalPath := filepath.Join(installDir, packageName)
		if _, errStat := os.Stat(backupPath); errStat == nil {
			errRestore := os.Rename(backupPath, originalPath)
			if errRestore != nil {
				fmt.Printf("Failed to restore %s from backup: %v\n", packageName, errRestore)
			} else {
				fmt.Printf("Restored %s from backup\n", packageName)
				errReg := updPkgReg(packageName, originalPath, version)
				if errReg != nil {
					fmt.Printf("Failed to re-register %s: %v\n", packageName, errReg)
				}
			}
		}
		return fmt.Errorf("failed to install updated package: %v", err)
	}

	// Remove backup if installation succeeds
	backupPath := fmt.Sprintf("%s.bak", filepath.Join(installDir, packageName))
	os.Remove(backupPath) // Ignore errors, as backup may not exist
	fmt.Printf("Successfully updated %s to version %s\n", packageName, version)
	return nil
}

func sv() {
	fmt.Println("Kazoo Package Installer, Version 1.0")
}
