package internal

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/pbnjay/memory"
)

type TOMLConfig struct {
	Config struct {
		User struct {
			VMUsername string `toml:"vm_username"`
			VMPassword string `toml:"vm_password"`
		} `toml:"user"`
		Download struct {
			DownloadLanguage string `toml:"download_language"`
		} `toml:"download"`
		Debloat struct {
			Debloat bool `toml:"debloat"`
		} `toml:"debloat"`
		Disksize struct {
			Disksize int `toml:"disksize"`
		} `toml:"disksize"`
		RdpPort struct {
			RdpPort int `toml:"rdp_port"`
		} `toml:"rdp_port"`
		VMMem struct {
			VMMem int `toml:"vm_mem"`
		} `toml:"vm_mem"`
		FreeRamGoal struct {
			FreeRamGoal int `toml:"free_ram_goal"`
		} `toml:"free_ram_goal"`
		UsbPassthrough struct {
			UsbPassthrough string `toml:"usb_passthrough"`
		} `toml:"usb_passthrough"`
		ReduceGraphics struct {
			ReduceGraphics bool `toml:"reduce_graphics"`
		} `toml:"reduce_graphics"`
		Fullscreen struct {
			Fullscreen bool `toml:"fullscreen"`
		} `toml:"fullscreen"`
		BVMDebug struct {
			BVMDebug bool `toml:"bvm_debug"`
		} `toml:"bvm_debug"`
		Debug struct {
			Debug bool `toml:"debug"`
		} `toml:"debug"`
		DisableUpdates struct {
			DisableUpdates bool `toml:"disable_updates"`
		} `toml:"disable_updates"`
		AddFreerdpFlags struct {
			AddFreerdpFlags string `toml:"add_freerdp_flags"`
		} `toml:"add_freerdp_flags"`
		NetworkFlags struct {
			NetworkFlags string `toml:"network_flags"`
		} `toml:"network_flags"`
		Virtualization struct {
			Virtualization string `toml:"virtualization"`
		} `toml:"virtualization"`
	} `toml:"config"`
	BVM struct {
		General struct {
			DisableUpdates bool `toml:"disable_updates"`
		} `toml:"general"`
		Splash struct {
			Splash bool `toml:"splash"`
		} `toml:"splash"`
	} `toml:"bvm"`
}

var (
	BVMDir    string
	BVMConfig struct {
		VMName           string
		VMDir            string
		VMPassword       string
		VMUsername       string
		DownloadLanguage string
		Debloat          bool
		Disksize         int
		FreeRamGoal      int
		VMMem            int
		RdpPort          int
		UsbPassthrough   string
		ReduceGraphics   bool
		Fullscreen       bool
		BVMDebug         bool
		Debug            bool
		DisableUpdates   bool
		AddFreerdpFlags  string
		NetworkFlags     string
		Splash           bool
		Mode             string
		Virtualization   string
	}
)

// Init intializes enviroment variables required for BVM Go to function.
//
// It should be called automatically in the bvm-go command line tool but in some circumstances it is required to call this function manually
func Init() {
	initBVMDir()

	// Check if the bvm-config.toml file exists in the BVM directory
	bvmConfigFile := filepath.Join(BVMDir, "resources", "bvm-config.toml")
	if _, err := os.Stat(bvmConfigFile); os.IsNotExist(err) {
		ErrorNoExit("bvm-config.toml file not found in " + BVMDir)
		os.Exit(1)
	}
	// Parse the nested TOML structure
	var tomlConfig TOMLConfig
	_, err := toml.DecodeFile(bvmConfigFile, &tomlConfig)
	if err != nil {
		ErrorNoExit("bvm-config.toml file is invalid in " + BVMDir + ": " + err.Error())
		os.Exit(1)
	}

	// Map the nested structure to our flat BVMConfig
	BVMConfig.VMUsername = tomlConfig.Config.User.VMUsername
	BVMConfig.VMPassword = tomlConfig.Config.User.VMPassword
	BVMConfig.DownloadLanguage = tomlConfig.Config.Download.DownloadLanguage
	BVMConfig.Debloat = tomlConfig.Config.Debloat.Debloat
	BVMConfig.Disksize = tomlConfig.Config.Disksize.Disksize
	BVMConfig.RdpPort = tomlConfig.Config.RdpPort.RdpPort
	BVMConfig.VMMem = tomlConfig.Config.VMMem.VMMem
	BVMConfig.FreeRamGoal = tomlConfig.Config.FreeRamGoal.FreeRamGoal
	BVMConfig.UsbPassthrough = tomlConfig.Config.UsbPassthrough.UsbPassthrough
	BVMConfig.ReduceGraphics = tomlConfig.Config.ReduceGraphics.ReduceGraphics
	BVMConfig.Fullscreen = tomlConfig.Config.Fullscreen.Fullscreen
	BVMConfig.BVMDebug = tomlConfig.Config.BVMDebug.BVMDebug
	BVMConfig.Debug = tomlConfig.Config.Debug.Debug
	BVMConfig.DisableUpdates = tomlConfig.Config.DisableUpdates.DisableUpdates
	BVMConfig.AddFreerdpFlags = tomlConfig.Config.AddFreerdpFlags.AddFreerdpFlags
	BVMConfig.NetworkFlags = tomlConfig.Config.NetworkFlags.NetworkFlags
	BVMConfig.Splash = tomlConfig.BVM.Splash.Splash
	BVMConfig.Virtualization = tomlConfig.Config.Virtualization.Virtualization
	// Populate the confugration file if it is empty
	if BVMConfig.VMName == "" {
		BVMConfig.VMName = "default-vm"
	}
	if BVMConfig.VMDir == "" {
		BVMConfig.VMDir = filepath.Join(BVMDir, BVMConfig.VMName)
	}
	if BVMConfig.VMPassword == "" {
		BVMConfig.VMPassword = "win11arm"
	}
	if BVMConfig.VMUsername == "" {
		BVMConfig.VMUsername = "Win11ARM"
	}
	if BVMConfig.DownloadLanguage == "" {
		BVMConfig.DownloadLanguage = "English (United States)"
	}
	if !BVMConfig.Debloat {
		BVMConfig.Debloat = true
	}
	if BVMConfig.Disksize == 0 {
		BVMConfig.Disksize = 40
	}
	if BVMConfig.FreeRamGoal == 0 {
		BVMConfig.FreeRamGoal = 100
	}
	// Choose RAM to allocate to VM - 1GB less than total RAM
	if BVMConfig.VMMem == 0 {
		BVMConfig.VMMem = int(memory.TotalMemory() / 1024 / 1024 / 1024)
		//Force 2GB on <=2GB devices
		if BVMConfig.VMMem <= 2 {
			BVMConfig.VMMem = 2
		}
		//Take off 1GB on >=5GB devices
		if BVMConfig.VMMem >= 5 {
			BVMConfig.VMMem = BVMConfig.VMMem - 1
		}
		//so for boot mode, RAM allocation works out like this:
		//Pi     VM
		//1GB -> 2GB (likely fails)
		//2GB -> 2GB
		//4GB -> 3GB
		//8GB -> 6GB
		//16GB -> 14GB
		// More than 4GB has no benefit for firstinstall
		if BVMConfig.Mode == "firstinstall" && BVMConfig.VMMem > 4 {
			BVMConfig.VMMem = 4
		}
	}
	if BVMConfig.RdpPort == 0 {
		BVMConfig.RdpPort = 3389
	}
	if BVMConfig.UsbPassthrough == "" {
		BVMConfig.UsbPassthrough = ""
	}
	if !BVMConfig.ReduceGraphics {
		BVMConfig.ReduceGraphics = false
	}
	if !BVMConfig.Fullscreen {
		BVMConfig.Fullscreen = false
	}
	if !BVMConfig.BVMDebug {
		BVMConfig.BVMDebug = false
	}
	if !BVMConfig.Debug {
		BVMConfig.Debug = false
	}
	if !BVMConfig.DisableUpdates {
		BVMConfig.DisableUpdates = false
	}
	if BVMConfig.AddFreerdpFlags == "" {
		BVMConfig.AddFreerdpFlags = "(-drives -home-drive -wallpaper)"
	}
	if BVMConfig.NetworkFlags == "" {
		BVMConfig.NetworkFlags = "(-netdev user,id=nic,hostfwd=tcp:127.0.0.1:" + strconv.Itoa(BVMConfig.RdpPort) + "-:3389 -device virtio-net-pci,netdev=nic)"
	}
	if !BVMConfig.Splash {
		BVMConfig.Splash = false
	}
	if BVMConfig.Mode == "" {
		BVMConfig.Mode = "firstinstall"
	}
	if BVMConfig.Virtualization == "" {
		BVMConfig.Virtualization = "qemu"
	}
	// The bvm-config.toml template file should already exist in the resources directory
	// We don't need to generate it dynamically since it's a template with comments

	// fixes https://github.com/Botspot/bvm/issues/26
	os.Setenv("PAN_MESA_DEBUG", "gl3")

	Debug("BVMDir: " + BVMDir)
	Debug("mode: " + BVMConfig.Mode)
	Debug("vmdir: " + BVMConfig.VMDir)
	Debug("vm_username: " + BVMConfig.VMUsername)
	Debug("vm_password: " + BVMConfig.VMPassword)
	Debug("vm_mem: " + strconv.Itoa(BVMConfig.VMMem))
	Debug("rdp_port: " + strconv.Itoa(BVMConfig.RdpPort))
	Debug("disksize: " + strconv.Itoa(BVMConfig.Disksize))
	Debug("debloat: " + strconv.FormatBool(BVMConfig.Debloat))
	Debug("fullscreen: " + strconv.FormatBool(BVMConfig.Fullscreen))
	Debug("virtualization: " + BVMConfig.Virtualization)

	// Update check
	UpdateCheck()

	// Copy icons
	CopyIcons()

	// Check system for compatibility
	if memory.TotalMemory() < 2*1024*1024*1024 {
		ErrorNoExit("Your system needs at least 2 GB of RAM. Be aware that a VM might be able to boot on 1 GB of RAM, but it cannot install Windows with less than 2 GB.")
		os.Exit(1)
	}
	// Check for the following: AMD64 userland/ARM64 userland or ARMv7 userland
	// Check OS architecture: ARM64, AMD64, or ARM32 userland required
	if runtime.GOARCH == "386" {
		ErrorNoExit("User error: OS CPU architecture is 32-bit x86! BVM only works on 64-bit operating systems (ARM64 or AMD64).")
		os.Exit(1)
	}

	// Additional check for ARM32 vs ARM64 by examining the init binary
	if runtime.GOARCH == "arm" {
		// Check if this is actually ARM32 by examining the ELF header of /sbin/init
		initPath, err := filepath.EvalSymlinks("/sbin/init")
		if err != nil {
			initPath = "/sbin/init"
		}

		file, err := os.Open(initPath)
		if err == nil {
			defer file.Close()

			// Read the EI_CLASS byte at offset 4 in the ELF header
			// 0x01 = 32-bit, 0x02 = 64-bit
			buffer := make([]byte, 5)
			if n, err := file.Read(buffer); err == nil && n >= 5 {
				if buffer[4] == 0x01 {
					ErrorNoExit("User error: OS CPU architecture is 32-bit ARM! BVM only works on ARM 64-bit operating systems.")
					os.Exit(1)
				}
			}
		}
	}

	// Check storage location for VM directory is not a FAT partition
	if BVMConfig.VMDir != "" {
		if IsFATPartition(BVMConfig.VMDir) {
			ErrorNoExit("The VM directory is on a FAT32/FAT16/vfat partition. This type of partition cannot contain files larger than 4GB, however the Windows image will be larger than that.\nPlease format the drive with an Ext4 partition, or use another drive.")
			os.Exit(1)
		}
	}

	// Check if the user is root
	if os.Getuid() == 0 {
		ErrorNoExit("BVM cannot be run as root user. Wayland programs need to be run as a non-root user.")
		os.Exit(1)
	}
}

// helper function to check if a partition is a FAT partition
func IsFATPartition(path string) bool {
	dfOutput, err := runCommand("df", "-T", path)
	if err != nil {
		return false
	}
	return strings.Contains(dfOutput, "fat")
}

// initBVMDir determines and sets the BVM directory location
func initBVMDir() {
	// Check if BVM_DIR environment variable is already set
	bvmDir := os.Getenv("BVM_DIR")
	if bvmDir != "" && isValidBVMDir(bvmDir) {
		BVMDir = bvmDir
		return
	}

	// Try to determine the directory based on the executable location
	// This approach mimics the bash script behavior
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		// If this is being run from the bvm-go/bin directory,
		// go up one level to get the bvm-go directory
		if strings.HasSuffix(exeDir, "/bvm-go/bin") {
			BVMDir = filepath.Dir(filepath.Dir(exeDir))
		} else {
			// Otherwise assume the executable directory is the bvm-go directory
			BVMDir = exeDir
		}

		// Check if this is a valid BVM directory
		if isValidBVMDir(BVMDir) {
			os.Setenv("BVM_DIR", BVMDir)
			return
		}
	}

	// Try current working directory
	if wd, err := os.Getwd(); err == nil && isValidBVMDir(wd) {
		BVMDir = wd
		os.Setenv("BVM_DIR", BVMDir)
		return
	}

	// If we still don't have a valid directory, use the default
	homeDir, _ := os.UserHomeDir()
	BVMDir = filepath.Join(homeDir, "bvm-go")

	// Set BVM_DIR environment variable
	os.Setenv("BVM_DIR", BVMDir)
}

// isValidBVMDir checks if a directory is a valid BVM directory
func isValidBVMDir(dir string) bool {
	if dir == "" {
		return false
	}

	// Check if the directory exists and has the expected files
	configFile := filepath.Join(dir, "resources", "bvm-config.toml")
	iconFile := filepath.Join(dir, "resources", "graphics", "icon.png")

	_, err := os.Stat(dir)
	if err != nil {
		return false
	}
	_, err = os.Stat(configFile)
	if err != nil {
		return false
	}
	_, err = os.Stat(iconFile)
	return err == nil
}

// Helper function to install icons to a location where the panel will notice them
func CopyIcons() {
	iconsDir := filepath.Join(os.Getenv("HOME"), ".local", "share", "icons", "hicolor", "scalable", "apps")
	os.MkdirAll(iconsDir, 0755)
	os.Symlink(filepath.Join(BVMDir, "resources", "graphics", "icon.png"), filepath.Join(iconsDir, "qemu.png"))
	os.Symlink(filepath.Join(BVMDir, "resources", "graphics", "icon.png"), filepath.Join(iconsDir, "bvm.png"))
	os.Symlink(filepath.Join(BVMDir, "resources", "graphics", "icon.png"), filepath.Join(iconsDir, "wlfreerdp.png"))
	os.Symlink(filepath.Join(BVMDir, "resources", "graphics", "icon.png"), filepath.Join(iconsDir, "xfreerdp.png"))
	//for wf-panel-pi at least, updating icon caches seems to do no good
}
