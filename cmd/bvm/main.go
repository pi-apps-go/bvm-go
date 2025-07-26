package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pi-apps-go/bvm-go/internal"
	"github.com/pi-apps-go/bvm-go/pkg/cli"
)

func main() {

	// initialize the environment
	internal.Init()

	// generate the logo and funny messages
	internal.GenerateLogo()
	internal.GenerateFunnyMessages()

	// check arguments: if no arguments, print help
	if len(os.Args) == 1 {
		internal.ErrorNoExit("Must specify a mode")
		printHelp()
		os.Exit(1)
	}

	// check if the command is valid
	switch os.Args[1] {
	case "new-vm":
		// Create a new virtual machine directory with config files
		if len(os.Args) < 3 {
			internal.ErrorNoExit("Must specify a directory for new-vm mode")
			printHelp()
			os.Exit(1)
		}
		vmDir := os.Args[2]

		if err := internal.CreateNewVM(vmDir); err != nil {
			fmt.Printf("Error creating new VM: %v\n", err)
			os.Exit(1)
		}

	case "download":
		// Interactive TUI for downloading Windows ISOs
		vmName := "default-vm"
		if len(os.Args) > 2 {
			vmName = os.Args[2]
		} else {
			internal.ErrorNoExit("Must specify a VM name for download mode")
			printHelp()
			os.Exit(1)
		}

		tui := cli.DownloadCLI(vmName)
		program := tea.NewProgram(tui, tea.WithAltScreen())

		finalModel, err := program.Run()
		if err != nil {
			internal.ErrorNoExit("TUI error: " + err.Error())
			os.Exit(1)
		}

		// Get selections from the TUI
		if downloadModel, ok := finalModel.(cli.DownloadModel); ok {
			if selections := downloadModel.GetSelections(); selections != nil {
				internal.Status("Starting Windows download with selected options...")

				// Convert selections to parameters for DownloadWindowsISO
				var release, version, arch, edition string

				if selections.SelectedVersion == "Custom ISO" {
					// Handle custom ISO
					release = "Custom ISO"
					internal.DownloadWindowsISO("", selections.VmName, release, "", "", "", selections.SelectedCustomISO, selections.SelectedCustomVirtio)
				} else {
					// Handle standard Windows versions
					release = selections.SelectedVersion

					// Map version based on architecture selection
					if strings.Contains(selections.SelectedArch, "22631") {
						version = "22631"
					} else {
						version = "latest"
					}

					// Map architecture
					switch selections.SelectedArch {
					case "ARM64 (Latest)", "ARM64":
						arch = "arm64"
					case "ARM64 (22631)":
						arch = "arm64"
					case "ARMv7":
						arch = "arm"
					case "x64":
						arch = "x64"
					default:
						arch = "arm64" // default fallback
					}

					// Set edition (default to empty for most cases)
					edition = selections.SelectedEdition

					internal.DownloadWindowsISO(selections.SelectedLanguage, selections.VmName, release, version, arch, edition)
				}

				internal.StatusGreen("Download completed successfully!")
			} else {
				internal.Status("Download cancelled by user")
			}
		}
	case "list-languages":
		fmt.Println(internal.ListDownloadLanguages())
	case "testGreen":
		fmt.Println(internal.StatusCheckGreen())
	case "generate-logo":
		internal.GenerateLogo()
	case "prepare":
		// Prepare a VM for use
		if len(os.Args) < 3 {
			internal.ErrorNoExit("Must specify a VM directory for prepare mode")
			printHelp()
			os.Exit(1)
		}
		vmDir := os.Args[2]

		// Check if VM directory exists
		if _, err := os.Stat(vmDir); os.IsNotExist(err) {
			internal.ErrorNoExit("VM directory does not exist: " + vmDir)
			os.Exit(1)
		}

		if err := internal.PrepareVM(vmDir); err != nil {
			fmt.Printf("Error preparing VM: %v\n", err)
			os.Exit(1)
		}
	case "firstboot":
		// First boot a VM using libvirt
		if len(os.Args) < 3 {
			internal.ErrorNoExit("Must specify a VM directory for firstboot mode")
			printHelp()
			os.Exit(1)
		}
		vmDir := os.Args[2]

		// Check if VM directory exists
		if _, err := os.Stat(vmDir); os.IsNotExist(err) {
			internal.ErrorNoExit("VM directory does not exist: " + vmDir)
			os.Exit(1)
		}

		if err := cli.FirstBoot(vmDir); err != nil {
			fmt.Printf("Error during first boot: %v\n", err)
			os.Exit(1)
		}
	case "boot":
		//internal.BootVM()
		fmt.Println("Not implemented")
	case "boot-nodisplay":
		//internal.BootVMNoDisplay()
		fmt.Println("Not implemented")
	case "boot-gtk":
		//internal.BootVMGTK()
		fmt.Println("Not implemented")
	case "boot-ramfb":
		//internal.BootVMRAMFB()
		fmt.Println("Not implemented")
	case "manage-vms":
		//internal.ManageVMs()
		fmt.Println("Not implemented")
	case "manage-vms-cli":
		//internal.ManageVMsCLI()
		fmt.Println("Not implemented")
	case "connect":
		//internal.ConnectVM()
		fmt.Println("Not implemented")
	case "connect-remmina":
		//internal.ConnectVMRemmina()
		fmt.Println("Not implemented")
	case "connect-freerdp":
		//internal.ConnectVMFreeRDP()
		fmt.Println("Not implemented")
	case "gui":
		//internal.GUI()
		fmt.Println("Not implemented")
	case "run-in-terminal":
		if len(os.Args) < 3 {
			internal.ErrorNoExit("Must specify a command to run in terminal")
			printHelp()
			os.Exit(1)
		}
		internal.RunInTerminal(os.Args[2], os.Args[3])
	default:
		fmt.Println("Invalid command")
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	internal.Status("Usage: bvm <command> [options]")
	fmt.Println("Commands:")
	fmt.Println()
	fmt.Println("VM creation and single VM management:")
	fmt.Println()
	internal.Status("  new-vm: Create a new virtual machine")
	fmt.Println("  To get a fresh VM up and running, run 'bvm new-vm ~/win11'")
	fmt.Println("  This makes a config file: ~/win11/bvm-config.toml  <--- Please read it!")
	fmt.Println()
	internal.Status("  download: Download Windows ISO images")
	fmt.Println("  This downloads Windows and necessary drivers, with a option to select the language and Windows version.")
	fmt.Println()
	internal.Status("  prepare - Prepare a VM for use")
	fmt.Println("   This bundles everything up to get ready for first boot.")
	fmt.Println()
	internal.Status("  firstboot - First boot a VM")
	fmt.Println("   This runs the first boot of a VM, by running the 'bvm prepare' command and then the 'bvm start' command.")
	fmt.Println("   If the Windows install is interrupted, you can run this command again to continue the install.")
	fmt.Println("   Be aware: when Windows finishes installing, the VM will shutdown and all .iso files and the unattended folder could be deleted once this step is complete.")
	fmt.Println()
	internal.Status("  boot - Start a VM (uses SDL2 for display)")
	fmt.Println("   Main command to use the VM. Be aware: this mode will be laggy and lack crucial features.")
	fmt.Println("   If you want to use the VM in a better way, start the VM in headless mode (using the 'bvm boot-nodisplay' command) and connect to it with RDP (using the 'bvm connect' command).")
	fmt.Println()
	internal.Status("  boot-nodisplay - Start a VM in headless mode")
	fmt.Println("   This starts a VM in headless mode, which means it will not have a display and will not be able to be used directly.")
	fmt.Println()
	internal.Status("  boot-gtk: Start a VM with GTK frontend")
	fmt.Println("   This starts a VM, but uses GTK to show the VM. This is useful if you want to use the VM in a better way, but you don't want to use the connect mode.")
	fmt.Println()
	internal.Status("  boot-ramfb: Start a VM with a generic framebuffer")
	fmt.Println("   This starts a VM, but uses a generic framebuffer to show the VM. This is useful if you are unable to connect to the VM.")
	fmt.Println("   This mode is useful for troubleshooting.")
	fmt.Println()
	fmt.Println("Multiple VM management:")
	internal.Status("  manage-vms - Manage VMs")
	fmt.Println("   This command is a GUI way to list all VMs, create a new VM, delete a VM, and edit a VM.")
	fmt.Println("   This command will also show the VM's status, and the VM's configuration file.")
	internal.Status("  manage-vms-cli - Manage VMs (CLI)")
	fmt.Println("   This command is a CLI way to list all VMs, create a new VM, delete a VM, and edit a VM.")
	fmt.Println("   This command will also show the VM's status, and the VM's configuration file.")
	fmt.Println()
	fmt.Println("Single VM connection management:")
	internal.Status("  connect - Connect to a VM")
	fmt.Println("   This command will open a RDP connection to the VM. You can use this to use the VM.")
	fmt.Println("   The connect mode has better audio, clipboard sync, file sharing, dynamic screen resizing, and a higher frame rate.")
	fmt.Println("   Default connect mode uses the Remmina client.")
	fmt.Println()
	internal.Status("  connect-remmina - Connect to a VM with Remmina")
	fmt.Println("   This command will open a Remmina connection to the VM. You can use this to use the VM.")
	fmt.Println("   The connect-remmina mode is the default connect mode.")
	fmt.Println()
	internal.Status("  connect-freerdp - Connect to a VM with FreeRDP")
	fmt.Println("   This command will open a FreeRDP connection to the VM. You can use this to use the VM.")
	fmt.Println("   The connect-freerdp mode is a fallback to the connect mode, if the Remmina client does not work.")
	fmt.Println()
	fmt.Println("Other commands:")
	fmt.Println()
	internal.Status("  list-languages: List available languages")
	fmt.Println("   This command will list all available languages for the Windows ISO images.")
	fmt.Println()
	internal.Status("  gui: Open the GUI")
	fmt.Println("   This command will open the GUI. You can use this to graphically manage the VM.")
}
