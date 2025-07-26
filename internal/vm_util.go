package internal

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

// Global variable to track current mount point for cleanup
var currentMountPoint string
var currentNBDDevice string

// umountRetry is a wrapper for umount command - try several times if it is still mounted
func umountRetry(mountpoint string) error {
	tries := 0

	// Sync to ensure all writes are flushed
	syscall.Sync()

	for {
		// Check if still mounted using mountpoint command
		cmd := exec.Command("mountpoint", "-q", mountpoint)
		if err := cmd.Run(); err != nil {
			// Not mounted anymore, we're done
			break
		}

		// Try to unmount
		cmd = exec.Command("sudo", "umount", mountpoint)
		err := cmd.Run()
		if err == nil {
			break // Successfully unmounted
		}

		if tries >= 10 {
			Warning("Could not unmount " + mountpoint + ", unmounting it lazily.")
			cmd = exec.Command("sudo", "umount", "-l", mountpoint)
			return cmd.Run()
		}

		time.Sleep(1 * time.Second)
		tries++
	}

	return nil
}

// ConnectQcow2ToNbd finds unused /dev/nbd* device to use, registers it with the specified qcow2 image, returns the device path
func ConnectQcow2ToNbd(qcow2File string) (string, error) {
	// Load nbd kernel module
	cmd := exec.Command("sudo", "modprobe", "nbd")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to load nbd module: %v", err)
	}

	// Sync to wait for any writes to finish
	syscall.Sync()

	// Find an unused /dev/nbd* device
	var nbdDevice string
	for i := 0; i < 16; i++ { // Check up to nbd15
		device := fmt.Sprintf("/dev/nbd%d", i)

		// Check if device exists
		if _, err := os.Stat(device); os.IsNotExist(err) {
			return "", fmt.Errorf("nbd device %s does not exist - all seem to be taken", device)
		}

		// Check if device is in use by running lsblk
		cmd := exec.Command("lsblk", device, "-no", "MOUNTPOINTS")
		output, err := cmd.Output()
		if err != nil || strings.TrimSpace(string(output)) == "" {
			// Device is not in use
			nbdDevice = device
			break
		}
	}

	if nbdDevice == "" {
		return "", fmt.Errorf("failed to find available nbd device - all seem to be taken")
	}

	// Try to connect the qcow2 file to the nbd device, retry up to 5 times
	for attempt := 0; attempt < 5; attempt++ {
		cmd := exec.Command("sudo", "qemu-nbd", "--connect="+nbdDevice, qcow2File)
		if err := cmd.Run(); err == nil {
			break // Success
		}

		// Disconnect and try again
		exec.Command("sudo", "qemu-nbd", "--disconnect", nbdDevice).Run()
		// Try to unmount any existing mounts
		UnmountQcow2(qcow2File)
		time.Sleep(2 * time.Second)

		if attempt == 4 {
			return "", fmt.Errorf("failed to connect %s to nbd device %s after 5 attempts", qcow2File, nbdDevice)
		}
	}

	// Wait for partition 4 to appear (Windows main partition), give it up to 10 seconds
	partition4 := nbdDevice + "p4"
	for attempt := 0; attempt < 5; attempt++ {
		if _, err := os.Stat(partition4); err == nil {
			// Partition 4 exists
			return nbdDevice, nil
		}
		time.Sleep(2 * time.Second)
	}

	// Partition 4 not found, show what partitions are available and disconnect
	cmd = exec.Command("lsblk", nbdDevice)
	output, _ := cmd.Output()
	Status("Available partitions on " + nbdDevice + ":")
	Status(string(output))

	exec.Command("sudo", "qemu-nbd", "--disconnect", nbdDevice).Run()
	return "", fmt.Errorf("partition 4 not found on %s", nbdDevice)
}

// MountQcow2 mounts the qcow2 disk image to /media/$USER/bvm-mount{rdp_port} folder
func MountQcow2(qcow2File string) error {
	// Get current user
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %v", err)
	}

	// Connect qcow2 to nbd device
	nbdDevice, err := ConnectQcow2ToNbd(qcow2File)
	if err != nil {
		return err
	}

	// Set global variables for cleanup
	currentNBDDevice = nbdDevice
	mountBase := getMountPointBase(currentUser.Username)
	currentMountPoint = fmt.Sprintf("%s/bvm-mount%d", mountBase, BVMConfig.RdpPort)

	Debug(fmt.Sprintf("Using mount point: %s", currentMountPoint))

	// Create mount directory
	cmd := exec.Command("sudo", "mkdir", "-p", currentMountPoint)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create mount directory: %v", err)
	}

	// Unmount if already mounted
	umountRetry(currentMountPoint)

	// Mount partition 4 (Windows main partition)
	partition4 := nbdDevice + "p4"
	cmd = exec.Command("sudo", "mount", partition4, currentMountPoint)
	if err := cmd.Run(); err != nil {
		// Mount failed, clean up
		UnmountQcow2(qcow2File)
		return fmt.Errorf("mount command failed: %v", err)
	}

	Status(fmt.Sprintf("Successfully mounted %s to %s", qcow2File, currentMountPoint))
	return nil
}

// UnmountQcow2 unmounts what was mounted by the MountQcow2 function
func UnmountQcow2(qcow2File string) error {
	// Get current user
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %v", err)
	}

	mountBase := getMountPointBase(currentUser.Username)
	mountPoint := fmt.Sprintf("%s/bvm-mount%d", mountBase, BVMConfig.RdpPort)

	// Find which /dev/nbd* is used by this mountpoint
	cmd := exec.Command("lsblk", "-no", "MOUNTPOINTS,PKNAME")
	output, err := cmd.Output()
	var nbdDevice string

	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, mountPoint+" ") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					nbdDevice = "/dev/" + parts[1]
					break
				}
			}
		}
	}

	// Unmount the mountpoint
	if err := umountRetry(mountPoint); err != nil {
		Warning("Failed to unmount " + mountPoint + ": " + err.Error())
	}

	// Disconnect the nbd device if we found it
	if nbdDevice != "" && nbdDevice != "/dev/" {
		cmd = exec.Command("sudo", "qemu-nbd", "--disconnect", nbdDevice)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to disconnect nbd device %s: %v", nbdDevice, err)
		}
	}

	// Remove the mount directory
	cmd = exec.Command("sudo", "rmdir", mountPoint)
	cmd.Run() // Ignore errors here as directory might not be empty or might not exist

	// Clear global variables
	currentMountPoint = ""
	currentNBDDevice = ""

	return nil
}

// RemoveMicrosoftDefender removes Microsoft Defender files from the mounted Windows image
func RemoveMicrosoftDefender(mountpoint string) error {
	if !BVMConfig.Debloat {
		return nil // Debloating is disabled
	}

	defenderFound := false

	// Check if Windows Defender directory exists
	defenderPath := filepath.Join(mountpoint, "ProgramData", "Microsoft", "Windows Defender")
	if _, err := os.Stat(defenderPath); err == nil {
		defenderFound = true
	}

	// List of paths to remove
	pathsToRemove := []string{
		"ProgramData/Microsoft/Windows Defender",
		"ProgramData/Microsoft/Windows Defender Advanced Threat Protection",
		"ProgramData/Microsoft/Windows Security Health",
		"Program Files/Windows Defender",
		"Program Files/Windows Defender Advanced Threat Protection",
		"Windows/System32/smartscreen.dll",
		"Windows/System32/smartscreen.exe",
		"Windows/System32/smartscreenps.dll",
		"Windows/SysWOW64/smartscreen.dll",
		"Windows/SysWOW64/smartscreenps.dll",
	}

	// Remove each path
	for _, relativePath := range pathsToRemove {
		fullPath := filepath.Join(mountpoint, relativePath)
		cmd := exec.Command("sudo", "rm", "-rf", fullPath)
		if err := cmd.Run(); err != nil {
			UnmountQcow2("") // Try to unmount before returning error
			return fmt.Errorf("failed to remove Microsoft Defender from disk.qcow2. Most likely the install was interrupted or the VM encountered an unsafe shutdown: %v", err)
		}
	}

	if defenderFound {
		StatusGreen("Successfully removed Microsoft Defender from disk.qcow2")
	}

	return nil
}

// PrepareVM prepares a VM for first boot by creating unattended.iso and disk.qcow2
func PrepareVM(vmdir string) error {
	Status("Preparing VM for first boot...")

	// Check if unattended directory exists
	unattendedDir := filepath.Join(vmdir, "unattended")
	if _, err := os.Stat(unattendedDir); os.IsNotExist(err) {
		return fmt.Errorf("unattended directory does not exist: %s", unattendedDir)
	}

	// Detect Windows edition and update autounattend.xml with correct product key
	if err := updateAutounattendProductKey(vmdir); err != nil {
		Warning("Failed to auto-detect Windows edition and update product key: " + err.Error())
		Warning("Proceeding with existing autounattend.xml - you may need to manually fix the product key")
	}

	// Create unattended.iso from unattended directory
	unattendedISO := filepath.Join(vmdir, "unattended.iso")
	Status("Making unattended.iso...")

	// Use mkisofs/genisoimage to create the ISO
	// -l: allow full 31-character filenames
	// -J: Generate Joliet directory records
	// -r: Generate SUSP and RR records using the Rock Ridge protocol
	// -allow-lowercase: Allow lowercase characters in addition to the usual ISO9660 allowed characters
	// -allow-multidot: Allow more than one dot in filenames
	cmd := exec.Command("mkisofs", "-quiet", "-l", "-J", "-r", "-allow-lowercase", "-allow-multidot", "-o", unattendedISO, unattendedDir)
	if err := cmd.Run(); err != nil {
		// Try genisoimage as fallback
		cmd = exec.Command("genisoimage", "-quiet", "-l", "-J", "-r", "-allow-lowercase", "-allow-multidot", "-o", unattendedISO, unattendedDir)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create unattended.iso (tried both mkisofs and genisoimage): %v", err)
		}
	}
	StatusGreen("unattended.iso created successfully")

	// Create a second copy for duplicate mounting (needed for reliable autounattend.xml detection)
	unattended2ISO := filepath.Join(vmdir, "unattended2.iso")
	cmd = exec.Command("cp", unattendedISO, unattended2ISO)
	if err := cmd.Run(); err != nil {
		Warning("Failed to create unattended2.iso copy: " + err.Error())
	} else {
		Status("Created unattended2.iso copy for reliable Windows Setup detection")
	}

	// Handle main hard drive creation
	diskPath := filepath.Join(vmdir, "disk.qcow2")
	Status("Setting up main hard drive disk.qcow2")

	// Check if disk.qcow2 already exists
	if _, err := os.Stat(diskPath); err == nil {
		Warning("Proceeding will DELETE your VM's main hard drive and start over with a clean install.")
		Warning(fmt.Sprintf("(%s already exists)", diskPath))

		fmt.Print("Do you want to continue? (Y/n): ")
		reader := bufio.NewReader(os.Stdin)
		answer, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read user input: %v", err)
		}

		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer == "n" || answer == "no" {
			return fmt.Errorf("exiting as you requested")
		}
	}

	// Remove existing disk if present
	if err := os.Remove(diskPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete %s: %v", diskPath, err)
	}

	// Get disk size from config
	disksize := BVMConfig.Disksize
	if disksize == 0 {
		disksize = 40 // Default to 40GB
	}

	// Create new disk.qcow2
	Status(fmt.Sprintf("Allocating %dGB for main install drive...", disksize))

	// Use qemu-img to create the disk with optimal settings
	// cluster_size=2M: Larger cluster size for better performance
	// nocow=on: Disable copy-on-write for better performance on Btrfs
	// preallocation=metadata: Pre-allocate metadata for better performance
	cmd = exec.Command("qemu-img", "create", "-f", "qcow2",
		"-o", "cluster_size=2M,nocow=on,preallocation=metadata",
		diskPath, fmt.Sprintf("%dG", disksize))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create %s: %v\nOutput: %s", diskPath, err, string(output))
	}

	StatusGreen("Main hard drive created successfully")
	StatusGreen("VM preparation completed! You should now be ready for the next step: firstboot")

	return nil
}

// getMountPointBase determines the correct mount point base directory for this system
// This handles different Linux distributions that use different mount conventions:
// - /run/media/$USER: Arch Linux, Fedora, RHEL 7+, systemd-based systems
// - /media/$USER: Ubuntu, Debian, older systems
// - /mnt/$USER: Some custom setups
func getMountPointBase(username string) string {
	// Check common mount point locations in order of preference
	candidates := []string{
		fmt.Sprintf("/run/media/%s", username), // Arch Linux, Fedora, RHEL 7+, systemd-based systems
		fmt.Sprintf("/media/%s", username),     // Ubuntu, Debian, older systems
		fmt.Sprintf("/mnt/%s", username),       // Some custom setups
	}

	// First check if any of these directories already exist
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}

	// If none exist, check which base directory exists and is writable
	baseCandidates := []string{
		"/run/media", // Modern systemd-based systems
		"/media",     // Traditional systems
		"/mnt",       // Fallback
	}

	for i, baseCandidate := range baseCandidates {
		if info, err := os.Stat(baseCandidate); err == nil && info.IsDir() {
			// Check if we can create a subdirectory (test writability)
			testDir := filepath.Join(baseCandidate, username)
			if err := os.MkdirAll(testDir, 0755); err == nil {
				// Verify we can actually write to this directory
				testFile := filepath.Join(testDir, ".bvm-test")
				if file, err := os.Create(testFile); err == nil {
					file.Close()
					os.Remove(testFile)
					return testDir
				}
			}
		}

		// If it's the traditional /media, try to create it even if the base doesn't exist
		if i == 1 { // /media
			fallback := fmt.Sprintf("/media/%s", username)
			if err := os.MkdirAll(fallback, 0755); err == nil {
				// Verify we can actually write to this directory
				testFile := filepath.Join(fallback, ".bvm-test")
				if file, err := os.Create(testFile); err == nil {
					file.Close()
					os.Remove(testFile)
					return fallback
				}
			}
		}
	}

	// Last resort fallback
	fallback := fmt.Sprintf("/media/%s", username)
	os.MkdirAll(fallback, 0755) // Create if it doesn't exist, ignore errors
	return fallback
}

// GetMountPoint returns the current mount point that was set by MountQcow2
func GetMountPoint() string {
	return currentMountPoint
}

// updateAutounattendProductKey detects Windows edition and updates autounattend.xml with correct product key
func updateAutounattendProductKey(vmdir string) error {
	// Generic Install Keys for unattended setup (better for installation than KMS keys)
	// These keys select the Windows edition during setup but don't activate Windows
	editionKeys := map[string]string{
		// Windows 11 Generic Install Keys
		"Windows 11 Home":                   "TX9XD-98N7V-6WMQ6-BX7FG-H8Q99",
		"Windows 11 Pro":                    "VK7JG-NPHTM-C97JM-9MPGT-3V66T",
		"Windows 11 Pro N":                  "2B87N-8KFHP-DKV6R-Y2C8J-PKCKT",
		"Windows 11 Pro for Workstations":   "DXG7C-N36C4-C4HTG-X4T3X-2YV77",
		"Windows 11 Pro for Workstations N": "WYPNQ-8C467-V2W6J-TX4WX-WT2RQ",
		"Windows 11 Pro Education":          "8PTT6-RNW4C-6V7J2-C2D3X-MHBPB",
		"Windows 11 Pro Education N":        "GJTYN-HDMQY-FRR76-HVGC7-QPF8P",
		"Windows 11 Education":              "YNMGQ-8RYV3-4PGQ3-C8XTP-7CFBY",
		"Windows 11 Education N":            "84NGF-MHBT6-FXBX8-QWJK7-DRR8H",
		"Windows 11 Enterprise":             "XGVPP-NMH47-7TTHJ-W3FW7-8HV2C",
		"Windows 11 Enterprise N":           "WGGHN-J84D6-QYCPR-T7PJ7-X766F",
		"Windows 11 Enterprise G":           "YYVX9-NTFWV-6MDM3-9PT4T-4M68B",
		"Windows 11 Enterprise G N":         "44RPN-FTY23-9VTTB-MP9BX-T84FV",

		// Windows 10 Generic Install Keys
		"Windows 10 Home":                   "TX9XD-98N7V-6WMQ6-BX7FG-H8Q99",
		"Windows 10 Pro":                    "VK7JG-NPHTM-C97JM-9MPGT-3V66T",
		"Windows 10 Pro N":                  "2B87N-8KFHP-DKV6R-Y2C8J-PKCKT",
		"Windows 10 Pro for Workstations":   "DXG7C-N36C4-C4HTG-X4T3X-2YV77",
		"Windows 10 Pro for Workstations N": "WYPNQ-8C467-V2W6J-TX4WX-WT2RQ",
		"Windows 10 Pro Education":          "8PTT6-RNW4C-6V7J2-C2D3X-MHBPB",
		"Windows 10 Pro Education N":        "GJTYN-HDMQY-FRR76-HVGC7-QPF8P",
		"Windows 10 Education":              "YNMGQ-8RYV3-4PGQ3-C8XTP-7CFBY",
		"Windows 10 Education N":            "84NGF-MHBT6-FXBX8-QWJK7-DRR8H",
		"Windows 10 Enterprise":             "XGVPP-NMH47-7TTHJ-W3FW7-8HV2C",
		"Windows 10 Enterprise N":           "WGGHN-J84D6-QYCPR-T7PJ7-X766F",
		"Windows 10 Enterprise G":           "YYVX9-NTFWV-6MDM3-9PT4T-4M68B",
		"Windows 10 Enterprise G N":         "44RPN-FTY23-9VTTB-MP9BX-T84FV",
		"Windows 10 Enterprise LTSC 2019":   "XGVPP-NMH47-7TTHJ-W3FW7-8HV2C", // Use generic Enterprise key for LTSC
		"Windows 10 Enterprise LTSC 2021":   "XGVPP-NMH47-7TTHJ-W3FW7-8HV2C", // Use generic Enterprise key for LTSC

		// Windows Server editions (using generic install keys)
		"Windows Server 2019 Standard":   "N69G4-B89J2-4G8F4-WWYCC-J464C",
		"Windows Server 2019 Datacenter": "WMDGN-G9PQG-XVVXX-R3X43-63DFG",
		"Windows Server 2022 Standard":   "VDYBN-27WPP-V4HQT-9VMD4-VMK7H",
		"Windows Server 2022 Datacenter": "WX4NM-KYWYW-QJJR4-XV3QB-6VM33",
	}

	installerISO := filepath.Join(vmdir, "installer.iso")
	if _, err := os.Stat(installerISO); os.IsNotExist(err) {
		return fmt.Errorf("installer.iso not found in %s", vmdir)
	}

	// Try to detect Windows edition from ISO
	mountPoint := "/tmp/bvm_iso_detect"
	os.MkdirAll(mountPoint, 0755)
	defer os.RemoveAll(mountPoint)

	// Mount ISO
	cmd := exec.Command("sudo", "mount", "-r", installerISO, mountPoint)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to mount ISO: %v", err)
	}
	defer exec.Command("sudo", "umount", mountPoint).Run()

	// Check for install.wim first, then install.esd
	var wimFile string
	installWim := filepath.Join(mountPoint, "sources", "install.wim")
	installEsd := filepath.Join(mountPoint, "sources", "install.esd")

	if _, err := os.Stat(installWim); err == nil {
		wimFile = installWim
	} else if _, err := os.Stat(installEsd); err == nil {
		wimFile = installEsd
	} else {
		return fmt.Errorf("neither install.wim nor install.esd found in ISO")
	}

	// Use wiminfo to get Windows edition
	cmd = exec.Command("wiminfo", wimFile)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to run wiminfo: %v", err)
	}

	// Parse wiminfo output to find Windows edition name
	var detectedEdition string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Name:") {
			detectedEdition = strings.TrimSpace(strings.TrimPrefix(line, "Name:"))
			break
		}
	}

	if detectedEdition == "" {
		return fmt.Errorf("could not detect Windows edition from ISO")
	}

	Status(fmt.Sprintf("Detected Windows edition: %s", detectedEdition))

	// Look up the product key for this edition
	var productKey string
	if key, exists := editionKeys[detectedEdition]; exists {
		productKey = key
		Status(fmt.Sprintf("Using KMS client key for %s: %s", detectedEdition, productKey))
	} else {
		productKey = "W269N-WFGWX-YVC9B-4J6C9-T83GX" // Fallback to Pro key
		Warning(fmt.Sprintf("No specific key found for %s, using Windows 10/11 Pro key as fallback", detectedEdition))
	}

	// Update autounattend.xml with the correct product key
	autounattendPath := filepath.Join(vmdir, "unattended", "autounattend.xml")
	if _, err := os.Stat(autounattendPath); os.IsNotExist(err) {
		return fmt.Errorf("autounattend.xml not found at %s", autounattendPath)
	}

	// Read the file
	content, err := os.ReadFile(autounattendPath)
	if err != nil {
		return fmt.Errorf("failed to read autounattend.xml: %v", err)
	}

	// Replace product keys in the file using regex
	contentStr := string(content)

	// Pattern 1: Simple ProductKey tag
	productKeyPattern1 := regexp.MustCompile(`<ProductKey>[^<]*</ProductKey>`)
	contentStr = productKeyPattern1.ReplaceAllString(contentStr, fmt.Sprintf("<ProductKey>%s</ProductKey>", productKey))

	// Pattern 2: ProductKey with Key child element
	productKeyPattern2 := regexp.MustCompile(`<Key>[^<]*</Key>`)
	contentStr = productKeyPattern2.ReplaceAllString(contentStr, fmt.Sprintf("<Key>%s</Key>", productKey))

	// Write the updated content back to the file
	if err := os.WriteFile(autounattendPath, []byte(contentStr), 0644); err != nil {
		return fmt.Errorf("failed to write updated autounattend.xml: %v", err)
	}

	StatusGreen(fmt.Sprintf("Updated autounattend.xml with product key for %s", detectedEdition))
	return nil
}
