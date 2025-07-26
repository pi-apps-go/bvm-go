package cli

import (
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/pi-apps-go/bvm-go/internal"
	"libvirt.org/go/libvirt"
)

// LibvirtDomainXML represents the structure for libvirt domain XML
type LibvirtDomainXML struct {
	XMLName       xml.Name `xml:"domain"`
	Type          string   `xml:"type,attr"`
	Name          string   `xml:"name"`
	UUID          string   `xml:"uuid,omitempty"`
	Memory        Memory   `xml:"memory"`
	CurrentMemory Memory   `xml:"currentMemory"`
	VCPU          VCPU     `xml:"vcpu"`
	OS            OS       `xml:"os"`
	Features      Features `xml:"features"`
	CPU           CPU      `xml:"cpu"`
	Clock         Clock    `xml:"clock"`
	OnPoweroff    string   `xml:"on_poweroff"`
	OnReboot      string   `xml:"on_reboot"`
	OnCrash       string   `xml:"on_crash"`
	Devices       Devices  `xml:"devices"`
}

type Memory struct {
	Unit  string `xml:"unit,attr"`
	Value string `xml:",chardata"`
}

type VCPU struct {
	Placement string `xml:"placement,attr"`
	Value     string `xml:",chardata"`
}

type OS struct {
	Type     OSType  `xml:"type"`
	Firmware string  `xml:"firmware,attr,omitempty"`
	Loader   *Loader `xml:"loader,omitempty"`
	NVRam    *NVRam  `xml:"nvram,omitempty"`
	Boot     []Boot  `xml:"boot,omitempty"`
}

type OSType struct {
	Arch    string `xml:"arch,attr"`
	Machine string `xml:"machine,attr"`
	Value   string `xml:",chardata"`
}

type Loader struct {
	ReadOnly string `xml:"readonly,attr"`
	Type     string `xml:"type,attr"`
	Value    string `xml:",chardata"`
}

type NVRam struct {
	Value string `xml:",chardata"`
}

type Boot struct {
	Dev string `xml:"dev,attr"`
}

type Features struct {
	ACPI struct{} `xml:"acpi"`
	APIC struct{} `xml:"apic"`
	GIC  *GIC     `xml:"gic,omitempty"`
}

type GIC struct {
	Version string `xml:"version,attr"`
}

type CPU struct {
	Mode     string    `xml:"mode,attr"`
	Check    string    `xml:"check,attr,omitempty"`
	Topology *Topology `xml:"topology,omitempty"`
}

type Topology struct {
	Sockets string `xml:"sockets,attr"`
	Cores   string `xml:"cores,attr"`
	Threads string `xml:"threads,attr"`
}

type Clock struct {
	Offset string  `xml:"offset,attr"`
	Timer  []Timer `xml:"timer"`
}

type Timer struct {
	Name       string `xml:"name,attr"`
	Tickpolicy string `xml:"tickpolicy,attr,omitempty"`
	Present    string `xml:"present,attr,omitempty"`
}

type Devices struct {
	Emulator    string       `xml:"emulator"`
	Disks       []Disk       `xml:"disk"`
	Controllers []Controller `xml:"controller"`
	Interfaces  []Interface  `xml:"interface"`
	Serials     []Serial     `xml:"serial"`
	Consoles    []Console    `xml:"console"`
	Channels    []Channel    `xml:"channel"`
	Graphics    []Graphics   `xml:"graphics"`
	Videos      []Video      `xml:"video"`
	Inputs      []Input      `xml:"input"`
	USBs        []USB        `xml:"hostdev,omitempty"`
	RNGs        []RNG        `xml:"rng"`
	Sounds      []Sound      `xml:"sound,omitempty"`
	MemBalloon  *MemBalloon  `xml:"memballoon,omitempty"`
}

type Disk struct {
	Type   string      `xml:"type,attr"`
	Device string      `xml:"device,attr"`
	Driver *DiskDriver `xml:"driver,omitempty"`
	Source *DiskSource `xml:"source,omitempty"`
	Target DiskTarget  `xml:"target"`
	Boot   *Boot       `xml:"boot,omitempty"`
}

type DiskDriver struct {
	Name    string `xml:"name,attr"`
	Type    string `xml:"type,attr,omitempty"`
	Cache   string `xml:"cache,attr,omitempty"`
	IO      string `xml:"io,attr,omitempty"`
	Discard string `xml:"discard,attr,omitempty"`
}

type DiskSource struct {
	File string `xml:"file,attr,omitempty"`
}

type DiskTarget struct {
	Dev string `xml:"dev,attr"`
	Bus string `xml:"bus,attr,omitempty"`
}

type Controller struct {
	Type    string   `xml:"type,attr"`
	Index   string   `xml:"index,attr"`
	Model   string   `xml:"model,attr,omitempty"`
	Address *Address `xml:"address,omitempty"`
}

type Interface struct {
	Type   string           `xml:"type,attr"`
	Source *InterfaceSource `xml:"source,omitempty"`
	Model  *InterfaceModel  `xml:"model,omitempty"`
}

type InterfaceSource struct {
	Network string `xml:"network,attr,omitempty"`
}

type InterfaceModel struct {
	Type string `xml:"type,attr"`
}

type Serial struct {
	Type   string        `xml:"type,attr"`
	Target *SerialTarget `xml:"target,omitempty"`
}

type SerialTarget struct {
	Type string `xml:"type,attr,omitempty"`
	Port string `xml:"port,attr,omitempty"`
}

type Console struct {
	Type   string         `xml:"type,attr"`
	Target *ConsoleTarget `xml:"target,omitempty"`
}

type ConsoleTarget struct {
	Type string `xml:"type,attr,omitempty"`
	Port string `xml:"port,attr,omitempty"`
}

type Channel struct {
	Type   string         `xml:"type,attr"`
	Target *ChannelTarget `xml:"target,omitempty"`
}

type ChannelTarget struct {
	Type string `xml:"type,attr"`
	Name string `xml:"name,attr,omitempty"`
}

type Graphics struct {
	Type     string `xml:"type,attr"`
	Port     string `xml:"port,attr,omitempty"`
	AutoPort string `xml:"autoport,attr,omitempty"`
	Listen   string `xml:"listen,attr,omitempty"`
}

type Video struct {
	Model VideoModel `xml:"model"`
}

type VideoModel struct {
	Type    string `xml:"type,attr"`
	VRam    string `xml:"vram,attr,omitempty"`
	Heads   string `xml:"heads,attr,omitempty"`
	Primary string `xml:"primary,attr,omitempty"`
}

type Input struct {
	Type string `xml:"type,attr"`
	Bus  string `xml:"bus,attr"`
}

type USB struct {
	Mode   string     `xml:"mode,attr"`
	Type   string     `xml:"type,attr"`
	Source *USBSource `xml:"source"`
}

type USBSource struct {
	Vendor  *USBVendor  `xml:"vendor"`
	Product *USBProduct `xml:"product"`
}

type USBVendor struct {
	ID string `xml:"id,attr"`
}

type USBProduct struct {
	ID string `xml:"id,attr"`
}

type RNG struct {
	Model   string     `xml:"model,attr"`
	Backend RNGBackend `xml:"backend"`
}

type RNGBackend struct {
	Model string `xml:"model,attr"`
	Value string `xml:",chardata"`
}

type Sound struct {
	Model string `xml:"model,attr"`
}

type MemBalloon struct {
	Model string `xml:"model,attr"`
}

type Address struct {
	Type     string `xml:"type,attr,omitempty"`
	Domain   string `xml:"domain,attr,omitempty"`
	Bus      string `xml:"bus,attr,omitempty"`
	Slot     string `xml:"slot,attr,omitempty"`
	Function string `xml:"function,attr,omitempty"`
}

func NewVM(vmdir string) {
	internal.CreateNewVM(vmdir)
}

// FirstBoot runs the Windows installation process using libvirt
func FirstBoot(vmdir string) error {
	internal.Status("Starting Windows installation using libvirt...")

	// Check for desktop environment
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		return fmt.Errorf("BVM needs a desktop environment to run firstboot")
	}

	// Convert to absolute path for proper file checking
	absVmdir, err := filepath.Abs(vmdir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %s: %v", vmdir, err)
	}

	// Validate required files exist
	installerISO := filepath.Join(absVmdir, "installer.iso")
	unattendedISO := filepath.Join(absVmdir, "unattended.iso")
	diskImage := filepath.Join(absVmdir, "disk.qcow2")

	internal.Debug("Checking for required files:")
	internal.Debug("  installer.iso: " + installerISO)
	internal.Debug("  unattended.iso: " + unattendedISO)
	internal.Debug("  disk.qcow2: " + diskImage)

	if _, err := os.Stat(installerISO); os.IsNotExist(err) {
		return fmt.Errorf("installer.iso not found at %s. Run 'bvm download %s' first", installerISO, vmdir)
	}
	if _, err := os.Stat(unattendedISO); os.IsNotExist(err) {
		return fmt.Errorf("unattended.iso not found at %s. Run 'bvm prepare %s' first", unattendedISO, vmdir)
	}
	if _, err := os.Stat(diskImage); os.IsNotExist(err) {
		return fmt.Errorf("disk.qcow2 not found at %s. Run 'bvm prepare %s' first", diskImage, vmdir)
	}

	// Check file permissions for libvirt access
	for _, file := range []string{installerISO, unattendedISO, diskImage} {
		if info, err := os.Stat(file); err == nil {
			// Check if file is readable
			if info.Mode().Perm()&0444 == 0 {
				internal.Warning(fmt.Sprintf("File %s may not be readable by libvirt. Consider running: chmod +r %s", file, file))
			}
		}
	}

	// For enhanced compatibility with newer Windows Setup, try embedding autounattend.xml in installer ISO
	if err := createBootDriveAutounattend(absVmdir); err != nil {
		internal.Warning("Failed to create enhanced installer ISO: " + err.Error())
		internal.Warning("Proceeding with standard method...")
	}

	// Create floppy disk with autounattend.xml as additional fallback
	if err := createAutounattendFloppy(absVmdir); err != nil {
		internal.Warning("Failed to create autounattend floppy: " + err.Error())
	}

	// Ensure display environment is available for graphics
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		return fmt.Errorf("no display environment found - please run in a desktop session")
	}

	// Connect to libvirt (use session to avoid permission issues with user files)
	internal.Status("Connecting to libvirt...")
	conn, err := libvirt.NewConnect("qemu:///session")
	if err != nil {
		return fmt.Errorf("failed to connect to libvirt: %v\nPlease ensure:\n1. libvirtd service is running: sudo systemctl start libvirtd\n2. User session services are available\n3. You may need to install libvirt-daemon-config-network", err)
	}
	defer conn.Close()

	// Generate domain name based on vmdir
	baseDomainName := fmt.Sprintf("bvm-firstboot-%s", filepath.Base(vmdir))

	// Clean up any existing domain with this name first
	cleanupLibvirtDomain(conn, baseDomainName)

	// Generate unique domain name with timestamp for this run
	domainName := fmt.Sprintf("%s-%d", baseDomainName, time.Now().Unix())

	// Generate domain XML with the unique name
	domainXML, err := generateFirstBootDomainXML(vmdir, domainName)
	if err != nil {
		return fmt.Errorf("failed to generate domain XML: %v", err)
	}

	internal.Debug("Generated domain XML:")
	internal.Debug(domainXML)

	// Create domain
	domain, err := conn.DomainDefineXML(domainXML)
	if err != nil {
		return fmt.Errorf("failed to define domain: %v", err)
	}

	// Cleanup this specific domain when done
	defer cleanupLibvirtDomain(conn, domainName)
	defer domain.Free()

	// Start the domain
	if err := domain.Create(); err != nil {
		return fmt.Errorf("failed to start domain: %v", err)
	}

	internal.Status("Windows installation started. This will take several hours.")
	internal.Status("The VM will automatically shut down when installation is complete.")

	// Get SPICE port for viewer
	spicePort, err := getSpicePort(domain)
	if err != nil {
		internal.Warning("Could not get SPICE port: " + err.Error())
	} else {
		internal.Status(fmt.Sprintf("SPICE server listening on port %d", spicePort))
		// Launch SPICE viewer in the background
		go launchSpiceViewer(spicePort)
	}

	// Monitor the domain
	if err := monitorFirstBootProgress(domain, vmdir); err != nil {
		return fmt.Errorf("error during installation monitoring: %v", err)
	}

	// Post-installation cleanup
	return postFirstBootCleanup(vmdir)
}

// generateFirstBootDomainXML creates the libvirt domain XML for Windows installation
func generateFirstBootDomainXML(vmdir string, domainName ...string) (string, error) {
	// Convert vmdir to absolute path for libvirt
	absVmdir, err := filepath.Abs(vmdir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for %s: %v", vmdir, err)
	}

	// Determine CPU architecture and cores
	var arch, machine, emulator string
	var cores int

	switch runtime.GOARCH {
	case "arm64":
		arch = "aarch64"
		machine = "virt"
		emulator = "/usr/bin/qemu-system-aarch64"
		// Handle big.LITTLE CPU optimization
		cores = getCPUCores()
	case "amd64":
		arch = "x86_64"
		machine = "q35"
		emulator = "/usr/bin/qemu-system-x86_64"
		cores = runtime.NumCPU()
	default:
		return "", fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
	}

	// Determine domain name
	var name string
	if len(domainName) > 0 && domainName[0] != "" {
		name = domainName[0]
	} else {
		name = fmt.Sprintf("bvm-firstboot-%s", filepath.Base(vmdir))
	}

	// Build the domain configuration
	domain := LibvirtDomainXML{
		Type: "kvm",
		Name: name,
		Memory: Memory{
			Unit:  "GiB",
			Value: strconv.Itoa(internal.BVMConfig.VMMem),
		},
		CurrentMemory: Memory{
			Unit:  "GiB",
			Value: strconv.Itoa(internal.BVMConfig.VMMem),
		},
		VCPU: VCPU{
			Placement: "static",
			Value:     strconv.Itoa(cores),
		},
		OS: OS{
			Type: OSType{
				Arch:    arch,
				Machine: machine,
				Value:   "hvm",
			},
			Boot: []Boot{
				{Dev: "cdrom"},
				{Dev: "hd"},
			},
		},
		Features: Features{
			ACPI: struct{}{},
			APIC: struct{}{},
		},
		CPU: CPU{
			Mode:  "host-passthrough",
			Check: "none",
		},
		Clock: Clock{
			Offset: "localtime",
			Timer: []Timer{
				{Name: "rtc", Tickpolicy: "catchup"},
				{Name: "pit", Tickpolicy: "delay"},
				{Name: "hpet", Present: "no"},
			},
		},
		OnPoweroff: "destroy",
		OnReboot:   "restart",
		OnCrash:    "restart",
		Devices: Devices{
			Emulator: emulator,
		},
	}

	// Configure UEFI firmware for both ARM64 and x86_64
	if runtime.GOARCH == "arm64" {
		domain.OS.Firmware = "efi"
		domain.OS.Loader = &Loader{
			ReadOnly: "yes",
			Type:     "pflash",
			Value:    "/usr/share/qemu-efi-aarch64/QEMU_EFI.fd",
		}
		domain.Features.GIC = &GIC{Version: "2"}
	} else if runtime.GOARCH == "amd64" {
		domain.OS.Firmware = "efi"
		domain.OS.Loader = &Loader{
			ReadOnly: "yes",
			Type:     "pflash",
			Value:    "/usr/share/OVMF/OVMF_CODE_4M.fd",
		}
		// Let libvirt handle NVRAM automatically instead of specifying a fixed path
		// domain.OS.NVRam = &NVRam{
		//	Value: "/usr/share/OVMF/OVMF_VARS_4M.fd",
		// }
	}

	// Add main disk
	domain.Devices.Disks = append(domain.Devices.Disks, Disk{
		Type:   "file",
		Device: "disk",
		Driver: &DiskDriver{
			Name:    "qemu",
			Type:    "qcow2",
			Cache:   "none",
			IO:      "threads",
			Discard: "unmap",
		},
		Source: &DiskSource{
			File: filepath.Join(absVmdir, "disk.qcow2"),
		},
		Target: DiskTarget{
			Dev: "vda",
			Bus: "virtio",
		},
	})

	// Add installer ISO as IDE CD-ROM (more reliable for older Windows Setup)
	domain.Devices.Disks = append(domain.Devices.Disks, Disk{
		Type:   "file",
		Device: "cdrom",
		Driver: &DiskDriver{
			Name: "qemu",
			Type: "raw",
		},
		Source: &DiskSource{
			File: filepath.Join(absVmdir, "installer.iso"),
		},
		Target: DiskTarget{
			Dev: "hda",
			Bus: "ide",
		},
		Boot: &Boot{Dev: "cdrom"},
	})

	// Add multiple copies of unattended.iso with different bus types for maximum compatibility
	// This helps ensure Windows Setup finds the autounattend.xml file regardless of drive letter changes
	unattendedISOPath := filepath.Join(absVmdir, "unattended.iso")

	// Primary unattended ISO as IDE (Windows Setup checks IDE drives first)
	domain.Devices.Disks = append(domain.Devices.Disks, Disk{
		Type:   "file",
		Device: "cdrom",
		Driver: &DiskDriver{
			Name: "qemu",
			Type: "raw",
		},
		Source: &DiskSource{
			File: unattendedISOPath,
		},
		Target: DiskTarget{
			Dev: "hdb",
			Bus: "ide",
		},
	})

	// Secondary unattended ISO as SATA (for newer systems)
	domain.Devices.Disks = append(domain.Devices.Disks, Disk{
		Type:   "file",
		Device: "cdrom",
		Driver: &DiskDriver{
			Name: "qemu",
			Type: "raw",
		},
		Source: &DiskSource{
			File: unattendedISOPath,
		},
		Target: DiskTarget{
			Dev: "sda",
			Bus: "sata",
		},
	})

	// Tertiary unattended ISO as SCSI (additional fallback)
	domain.Devices.Disks = append(domain.Devices.Disks, Disk{
		Type:   "file",
		Device: "cdrom",
		Driver: &DiskDriver{
			Name: "qemu",
			Type: "raw",
		},
		Source: &DiskSource{
			File: unattendedISOPath,
		},
		Target: DiskTarget{
			Dev: "sdb",
			Bus: "scsi",
		},
	})

	// Add floppy disk with autounattend.xml if it exists
	floppyPath := filepath.Join(absVmdir, "autounattend.img")
	if _, err := os.Stat(floppyPath); err == nil {
		domain.Devices.Disks = append(domain.Devices.Disks, Disk{
			Type:   "file",
			Device: "floppy",
			Driver: &DiskDriver{
				Name: "qemu",
				Type: "raw",
			},
			Source: &DiskSource{
				File: floppyPath,
			},
			Target: DiskTarget{
				Dev: "fda",
				Bus: "fdc",
			},
		})
		internal.Debug("Added autounattend.img as floppy disk")
	}

	internal.Debug("Added installer.iso as IDE CD-ROM")
	internal.Debug("Added unattended.iso with multiple bus types for reliable detection")

	// Add controllers - support both IDE and SATA for maximum compatibility
	if runtime.GOARCH == "amd64" {
		domain.Devices.Controllers = []Controller{
			{Type: "usb", Index: "0", Model: "qemu-xhci"},
			{Type: "pci", Index: "0", Model: "pcie-root"},
			{Type: "ide", Index: "0"},                        // IDE controller for legacy compatibility
			{Type: "sata", Index: "0"},                       // SATA controller for modern drives
			{Type: "scsi", Index: "0", Model: "virtio-scsi"}, // SCSI controller for additional drives
			{Type: "fdc", Index: "0"},                        // Floppy disk controller
		}
	} else {
		domain.Devices.Controllers = []Controller{
			{Type: "usb", Index: "0", Model: "qemu-xhci"},
			{Type: "pci", Index: "0", Model: "pci-root"},
			{Type: "ide", Index: "0"},                        // IDE controller for legacy compatibility
			{Type: "sata", Index: "0"},                       // SATA controller for modern drives
			{Type: "scsi", Index: "0", Model: "virtio-scsi"}, // SCSI controller for additional drives
			{Type: "fdc", Index: "0"},                        // Floppy disk controller
		}
	}

	// Add network interface
	domain.Devices.Interfaces = []Interface{
		{
			Type:  "user",
			Model: &InterfaceModel{Type: "virtio"},
		},
	}

	// Add graphics (GTK for direct window display)
	domain.Devices.Graphics = []Graphics{
		{
			Type:     "spice",
			Port:     "-1",
			AutoPort: "yes",
			Listen:   "127.0.0.1",
		},
	}

	// Add video device
	domain.Devices.Videos = []Video{
		{
			Model: VideoModel{
				Type:    "virtio",
				VRam:    "16384",
				Heads:   "1",
				Primary: "yes",
			},
		},
	}

	// Add input devices
	domain.Devices.Inputs = []Input{
		{Type: "keyboard", Bus: "usb"},
		{Type: "tablet", Bus: "usb"},
	}

	// Add channels for SPICE guest agent
	domain.Devices.Channels = []Channel{
		{
			Type: "spicevmc",
			Target: &ChannelTarget{
				Type: "virtio",
				Name: "com.redhat.spice.0",
			},
		},
	}

	// Add RNG device
	domain.Devices.RNGs = []RNG{
		{
			Model: "virtio",
			Backend: RNGBackend{
				Model: "random",
				Value: "/dev/urandom",
			},
		},
	}

	// Add sound device (commented out due to audio backend issues in libvirt)
	// domain.Devices.Sounds = []Sound{
	//	{Model: "ich9"},
	// }

	// Add memory balloon
	domain.Devices.MemBalloon = &MemBalloon{Model: "virtio"}

	// Marshal to XML
	xmlData, err := xml.MarshalIndent(domain, "", "  ")
	if err != nil {
		return "", err
	}

	return xml.Header + string(xmlData), nil
}

// getCPUCores determines optimal CPU cores, handling big.LITTLE architectures
func getCPUCores() int {
	// Try to detect performance cores for big.LITTLE CPUs like RK3588
	cores, usePerfCores := internal.ListCoresToUse()
	if usePerfCores {
		return len(cores)
	}
	return runtime.NumCPU()
}

// detectWindowsEditionAndKey detects the Windows edition from ISO and returns appropriate KMS client key
func detectWindowsEditionAndKey(vmdir string) (string, error) {
	// Official Microsoft KMS Client Setup Keys
	// https://docs.microsoft.com/en-us/windows-server/get-started/kms-client-activation-keys
	editionKeys := map[string]string{
		// Windows 11
		"Windows 11 Pro":                    "W269N-WFGWX-YVC9B-4J6C9-T83GX",
		"Windows 11 Pro N":                  "MH37W-N47XK-V7XM9-C7227-GCQG9",
		"Windows 11 Pro for Workstations":   "NRG8B-VKK3Q-CXVCJ-9G2XF-6Q84J",
		"Windows 11 Pro for Workstations N": "9FNHH-K3HBT-3W4TD-6383H-6XYWF",
		"Windows 11 Pro Education":          "6TP4R-GNPTD-KYYHQ-7B7DP-J447Y",
		"Windows 11 Pro Education N":        "YVWGF-BXNMC-HTQYQ-CPQ99-66QFC",
		"Windows 11 Education":              "NW6C2-QMPVW-D7KKK-3GKT6-VCFB2",
		"Windows 11 Education N":            "2WH4N-8QGBV-H22JP-CT43Q-MDWWJ",
		"Windows 11 Enterprise":             "NPPR9-FWDCX-D2C8J-H872K-2YT43",
		"Windows 11 Enterprise N":           "DPH2V-TTNVB-4X9Q3-TJR4H-KHJW4",
		"Windows 11 Enterprise G":           "YYVX9-NTFWV-6MDM3-9PT4T-4M68B",
		"Windows 11 Enterprise G N":         "44RPN-FTY23-9VTTB-MP9BX-T84FV",

		// Windows 10
		"Windows 10 Pro":                    "W269N-WFGWX-YVC9B-4J6C9-T83GX",
		"Windows 10 Pro N":                  "MH37W-N47XK-V7XM9-C7227-GCQG9",
		"Windows 10 Pro for Workstations":   "NRG8B-VKK3Q-CXVCJ-9G2XF-6Q84J",
		"Windows 10 Pro for Workstations N": "9FNHH-K3HBT-3W4TD-6383H-6XYWF",
		"Windows 10 Pro Education":          "6TP4R-GNPTD-KYYHQ-7B7DP-J447Y",
		"Windows 10 Pro Education N":        "YVWGF-BXNMC-HTQYQ-CPQ99-66QFC",
		"Windows 10 Education":              "NW6C2-QMPVW-D7KKK-3GKT6-VCFB2",
		"Windows 10 Education N":            "2WH4N-8QGBV-H22JP-CT43Q-MDWWJ",
		"Windows 10 Enterprise":             "NPPR9-FWDCX-D2C8J-H872K-2YT43",
		"Windows 10 Enterprise N":           "DPH2V-TTNVB-4X9Q3-TJR4H-KHJW4",
		"Windows 10 Enterprise G":           "YYVX9-NTFWV-6MDM3-9PT4T-4M68B",
		"Windows 10 Enterprise G N":         "44RPN-FTY23-9VTTB-MP9BX-T84FV",
		"Windows 10 Enterprise LTSC 2019":   "M7XTQ-FN8P6-TTKYV-9D4CC-J462D",
		"Windows 10 Enterprise LTSC 2021":   "M7XTQ-FN8P6-TTKYV-9D4CC-J462D",

		// Windows Server editions (in case someone uses those)
		"Windows Server 2019 Standard":   "N69G4-B89J2-4G8F4-WWYCC-J464C",
		"Windows Server 2019 Datacenter": "WMDGN-G9PQG-XVVXX-R3X43-63DFG",
		"Windows Server 2022 Standard":   "VDYBN-27WPP-V4HQT-9VMD4-VMK7H",
		"Windows Server 2022 Datacenter": "WX4NM-KYWYW-QJJR4-XV3QB-6VM33",
	}

	installerISO := filepath.Join(vmdir, "installer.iso")
	if _, err := os.Stat(installerISO); os.IsNotExist(err) {
		return "", fmt.Errorf("installer.iso not found in %s", vmdir)
	}

	// Try to detect Windows edition from ISO
	// First try install.wim, then install.esd
	mountPoint := "/tmp/bvm_iso_detect"
	os.MkdirAll(mountPoint, 0755)
	defer os.RemoveAll(mountPoint)

	// Mount ISO
	cmd := exec.Command("sudo", "mount", "-r", installerISO, mountPoint)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to mount ISO: %v", err)
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
		return "", fmt.Errorf("neither install.wim nor install.esd found in ISO")
	}

	// Use wiminfo to get Windows edition
	cmd = exec.Command("wiminfo", wimFile)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run wiminfo: %v", err)
	}

	// Parse wiminfo output to find Windows edition name
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Name:") {
			editionName := strings.TrimSpace(strings.TrimPrefix(line, "Name:"))
			fmt.Printf("Detected Windows edition: %s\n", editionName)

			// Look up the product key for this edition
			if key, exists := editionKeys[editionName]; exists {
				fmt.Printf("Using KMS client key for %s: %s\n", editionName, key)
				return key, nil
			} else {
				fmt.Printf("Warning: No specific key found for %s, using Windows 10/11 Pro key as fallback\n", editionName)
				return "W269N-WFGWX-YVC9B-4J6C9-T83GX", nil // Fallback to Pro key
			}
		}
	}

	return "", fmt.Errorf("could not detect Windows edition from ISO")
}

// monitorFirstBootProgress monitors the installation process
func monitorFirstBootProgress(domain *libvirt.Domain, vmdir string) error {
	internal.Status("Monitoring installation progress...")

	for {
		// Check domain state
		state, _, err := domain.GetState()
		if err != nil {
			return fmt.Errorf("failed to get domain state: %v", err)
		}

		switch state {
		case libvirt.DOMAIN_SHUTOFF:
			internal.Status("Installation completed - VM has shut down")
			return nil
		case libvirt.DOMAIN_CRASHED:
			return fmt.Errorf("VM crashed during installation")
		case libvirt.DOMAIN_RUNNING:
			// Continue monitoring
		default:
			internal.Debug(fmt.Sprintf("Domain state: %d", state))
		}

		// Wait before next check
		time.Sleep(30 * time.Second)
	}
}

// postFirstBootCleanup handles post-installation tasks
func postFirstBootCleanup(vmdir string) error {
	internal.Status("Running post-installation cleanup...")

	// Mount disk and remove Microsoft Defender if debloat is enabled
	if internal.BVMConfig.Debloat {
		if err := removeMicrosoftDefender(vmdir); err != nil {
			internal.Warning("Failed to remove Microsoft Defender: " + err.Error())
		} else {
			internal.StatusGreen("Successfully removed Microsoft Defender from disk.qcow2")
		}
	}

	// Update GUI steps completion
	stepsFile := filepath.Join(vmdir, "gui-steps-complete")
	if err := os.WriteFile(stepsFile, []byte("5"), 0644); err != nil {
		internal.Warning("Failed to update GUI steps completion: " + err.Error())
	}

	internal.StatusGreen("You should now be ready for the next step: bvm boot " + vmdir)
	return nil
}

// getSpicePort retrieves the SPICE port from a running domain
func getSpicePort(domain *libvirt.Domain) (int, error) {
	xmlDesc, err := domain.GetXMLDesc(0)
	if err != nil {
		return 0, err
	}

	// Parse XML to find SPICE port - simplified approach
	// In practice, you'd want to use proper XML parsing
	lines := strings.Split(xmlDesc, "\n")
	for _, line := range lines {
		if strings.Contains(line, "graphics type='spice'") && strings.Contains(line, "port=") {
			// Extract port number from the line
			portStart := strings.Index(line, "port='") + 6
			portEnd := strings.Index(line[portStart:], "'")
			if portEnd > 0 {
				port, err := strconv.Atoi(line[portStart : portStart+portEnd])
				if err == nil {
					return port, nil
				}
			}
		}
	}
	return 5900, fmt.Errorf("could not find SPICE port in domain XML")
}

// launchSpiceViewer launches a SPICE viewer to connect to the VM
func launchSpiceViewer(port int) {
	internal.Status("Launching SPICE viewer...")

	// Wait a moment for the SPICE server to be ready
	time.Sleep(3 * time.Second)

	// Try different SPICE viewers in order of preference
	spiceViewers := []string{"remote-viewer", "spicy", "virt-viewer"}

	for _, viewer := range spiceViewers {
		if _, err := exec.LookPath(viewer); err == nil {
			cmd := exec.Command(viewer, fmt.Sprintf("spice://127.0.0.1:%d", port))
			if err := cmd.Start(); err != nil {
				internal.Warning(fmt.Sprintf("Failed to launch %s: %v", viewer, err))
			} else {
				internal.Status(fmt.Sprintf("Launched %s for VM display", viewer))
				return
			}
		}
	}

	internal.Warning("No SPICE viewer found. Please install virt-viewer or spice-gtk-tools")
	internal.Status(fmt.Sprintf("You can manually connect using: remote-viewer spice://127.0.0.1:%d", port))
}

// removeMicrosoftDefender mounts the disk and removes Defender files
func removeMicrosoftDefender(vmdir string) error {
	diskPath := filepath.Join(vmdir, "disk.qcow2")

	// Use internal VM utilities to mount and clean up
	if err := internal.MountQcow2(diskPath); err != nil {
		return fmt.Errorf("failed to mount disk: %v", err)
	}
	defer internal.UnmountQcow2("")

	// Get mount point
	mountPoint := internal.GetMountPoint()
	if mountPoint == "" {
		return fmt.Errorf("failed to get mount point")
	}

	// Check if installation completed by looking for qemu guest agent
	guestAgentPath := filepath.Join(mountPoint, "Program Files", "Qemu-ga", "qemu-ga.exe")
	if _, err := os.Stat(guestAgentPath); os.IsNotExist(err) {
		return fmt.Errorf("Windows installation appears incomplete - qemu guest agent not found")
	}

	// Remove Microsoft Defender files
	defenderPaths := []string{
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

	defenderFound := false
	for _, path := range defenderPaths {
		fullPath := filepath.Join(mountPoint, path)
		if _, err := os.Stat(fullPath); err == nil {
			defenderFound = true
			if err := os.RemoveAll(fullPath); err != nil {
				return fmt.Errorf("failed to remove %s: %v", path, err)
			}
		}
	}

	if !defenderFound {
		internal.Status("Microsoft Defender was not found (may have been removed already)")
	}

	return nil
}

// cleanupLibvirtDomain ensures the domain is properly destroyed and undefined after firstboot
func cleanupLibvirtDomain(conn *libvirt.Connect, domainName string) {
	// Only cleanup the specific domain we're working with
	domain, err := conn.LookupDomainByName(domainName)
	if err != nil {
		// Domain doesn't exist, which is fine
		internal.Debug("Domain " + domainName + " not found during cleanup: " + err.Error())
		return
	}
	defer domain.Free()

	cleanupSingleDomain(domain, domainName)
}

// cleanupSingleDomain handles cleanup of a single domain
func cleanupSingleDomain(domain *libvirt.Domain, domainName string) {
	// Get current state
	state, _, err := domain.GetState()
	if err != nil {
		internal.Debug("Failed to get domain state during cleanup: " + err.Error())
		// Continue with cleanup anyway
	}

	// Destroy the domain if it's still running
	if err == nil && state == libvirt.DOMAIN_RUNNING {
		internal.Status("Stopping domain " + domainName + "...")
		if err := domain.Destroy(); err != nil {
			internal.Warning("Failed to stop domain during cleanup: " + err.Error())
		}
	}

	// Undefine the domain with proper error handling
	internal.Status("Removing domain definition for " + domainName + "...")

	// Try with NVRAM flag first (for UEFI domains)
	undefineFlags := libvirt.DOMAIN_UNDEFINE_MANAGED_SAVE | libvirt.DOMAIN_UNDEFINE_SNAPSHOTS_METADATA | libvirt.DOMAIN_UNDEFINE_NVRAM
	if err := domain.UndefineFlags(undefineFlags); err != nil {
		// Try without NVRAM flag (for BIOS domains or if NVRAM doesn't exist)
		basicFlags := libvirt.DOMAIN_UNDEFINE_MANAGED_SAVE | libvirt.DOMAIN_UNDEFINE_SNAPSHOTS_METADATA
		if err := domain.UndefineFlags(basicFlags); err != nil {
			// Final fallback to simple undefine
			if err := domain.Undefine(); err != nil {
				internal.Warning("Failed to undefine domain during cleanup: " + err.Error())
				return
			}
		}
	}

	internal.Status("Domain " + domainName + " cleaned up successfully")
}

// createBootDriveAutounattend creates autounattend.xml directly on Windows installer boot drive
// This helps ensure newer Windows Setup versions can find the file
func createBootDriveAutounattend(vmdir string) error {
	internal.Status("Creating enhanced autounattend.xml for newer Windows Setup...")

	// Read the existing autounattend.xml
	unattendedPath := filepath.Join(vmdir, "unattended", "autounattend.xml")
	xmlContent, err := os.ReadFile(unattendedPath)
	if err != nil {
		return fmt.Errorf("failed to read autounattend.xml: %v", err)
	}

	// Create a temporary directory to mount the installer ISO
	tempMount, err := os.MkdirTemp("", "bvm-iso-modify-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempMount)

	installerISO := filepath.Join(vmdir, "installer.iso")

	// Create a working copy of the installer ISO
	modifiedISO := filepath.Join(vmdir, "installer-modified.iso")
	internal.Status("Creating modified installer ISO with embedded autounattend.xml...")

	// Extract the ISO to temporary directory
	cmd := exec.Command("7z", "x", installerISO, "-o"+tempMount, "-y")
	if err := cmd.Run(); err != nil {
		// Fallback to mount method
		cmd = exec.Command("sudo", "mount", "-r", installerISO, tempMount)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to extract/mount installer ISO: %v", err)
		}
		defer exec.Command("sudo", "umount", tempMount).Run()

		// Copy ISO contents to a writable directory
		tempWork, err := os.MkdirTemp("", "bvm-iso-work-*")
		if err != nil {
			return fmt.Errorf("failed to create work directory: %v", err)
		}
		defer os.RemoveAll(tempWork)

		cmd = exec.Command("cp", "-r", tempMount+"/.", tempWork)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to copy ISO contents: %v", err)
		}
		tempMount = tempWork
	}

	// Place autounattend.xml in multiple locations for maximum compatibility
	locations := []string{
		"autounattend.xml",             // Root of ISO
		"sources/autounattend.xml",     // sources directory
		"$OEM$/$1/autounattend.xml",    // OEM folder
		"$OEM$/$$/$1/autounattend.xml", // Alternative OEM location
	}

	for _, location := range locations {
		fullPath := filepath.Join(tempMount, location)

		// Create directory if it doesn't exist
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			internal.Warning("Failed to create directory for " + location + ": " + err.Error())
			continue
		}

		// Write the autounattend.xml file
		if err := os.WriteFile(fullPath, xmlContent, 0644); err != nil {
			internal.Warning("Failed to write autounattend.xml to " + location + ": " + err.Error())
		} else {
			internal.Debug("Added autounattend.xml to " + location)
		}
	}

	// Create new ISO with embedded autounattend.xml
	cmd = exec.Command("genisoimage", "-o", modifiedISO, "-b", "boot/etfsboot.com", "-no-emul-boot",
		"-boot-load-size", "8", "-boot-info-table", "-iso-level", "2", "-udf", "-joliet",
		"-D", "-N", "-relaxed-filenames", "-allow-leading-dots", "-allow-multidot",
		"-V", "BVM_WIN11_SETUP", tempMount)

	if err := cmd.Run(); err != nil {
		internal.Warning("Failed to create modified ISO, using original: " + err.Error())
		return nil // Don't fail the entire process
	}

	// Replace original with modified (backup original first)
	backupISO := filepath.Join(vmdir, "installer-original.iso")
	if err := os.Rename(installerISO, backupISO); err != nil {
		internal.Warning("Failed to backup original ISO: " + err.Error())
		return nil
	}

	if err := os.Rename(modifiedISO, installerISO); err != nil {
		// Restore original if move fails
		os.Rename(backupISO, installerISO)
		return fmt.Errorf("failed to replace installer ISO: %v", err)
	}

	internal.StatusGreen("Successfully created enhanced installer ISO with embedded autounattend.xml")
	return nil
}

// createAutounattendFloppy creates a floppy disk image with autounattend.xml
// This provides an additional method for Windows Setup to find the unattended file
func createAutounattendFloppy(vmdir string) error {
	internal.Status("Creating autounattend.xml floppy disk as additional fallback...")

	floppyPath := filepath.Join(vmdir, "autounattend.img")
	unattendedPath := filepath.Join(vmdir, "unattended", "autounattend.xml")

	// Create a 1.44MB floppy disk image
	cmd := exec.Command("dd", "if=/dev/zero", "of="+floppyPath, "bs=1024", "count=1440")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create floppy image: %v", err)
	}

	// Format as FAT12 filesystem
	cmd = exec.Command("mkfs.fat", "-F", "12", "-n", "AUTOUNATTEND", floppyPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to format floppy image: %v", err)
	}

	// Mount the floppy image temporarily
	tempMount, err := os.MkdirTemp("", "bvm-floppy-*")
	if err != nil {
		return fmt.Errorf("failed to create temp mount point: %v", err)
	}
	defer os.RemoveAll(tempMount)

	// Mount the floppy
	cmd = exec.Command("sudo", "mount", "-o", "loop", floppyPath, tempMount)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to mount floppy image: %v", err)
	}
	defer exec.Command("sudo", "umount", tempMount).Run()

	// Copy autounattend.xml to the floppy
	cmd = exec.Command("sudo", "cp", unattendedPath, filepath.Join(tempMount, "autounattend.xml"))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to copy autounattend.xml to floppy: %v", err)
	}

	// Set proper permissions
	cmd = exec.Command("sudo", "chmod", "644", filepath.Join(tempMount, "autounattend.xml"))
	cmd.Run() // Ignore errors for this

	internal.StatusGreen("Created autounattend floppy disk: " + floppyPath)
	return nil
}
