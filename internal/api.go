package internal

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

var (
	HostSystemID   string
	HostSystemDesc string
	HostSystemVer  string
	HostSystemCode string
	OrigSystemID   string
	OrigSystemDesc string
	OrigSystemVer  string
	OrigSystemCode string
)

func Error(message string) {
	fmt.Println("\033[91m" + message + "\033[0m")
	os.Exit(1)
}
func ErrorNoExit(message string) {
	fmt.Println("\033[91m" + message + "\033[0m")
}

func Warning(message string) {
	fmt.Println("\033[93m\033[5m◢◣\033[25m WARNING: " + message + "\033[0m")
}

func Status(message string) {
	fmt.Println("\033[96m" + message + "\033[0m")
}

func StatusGreen(message string) {
	fmt.Println("\033[92m" + message + "\033[0m")
}

func StatusCheckGreen() string {
	return "\033[92m" + "✓" + "\033[0m"
}

func StatusErrorRed() string {
	return "\033[91m" + "✕" + "\033[0m"
}

func Debug(message string) {
	if os.Getenv("BVM_DEBUG") == "true" {
		fmt.Println("\033[94m" + message + "\033[0m")
	}
}

func GenerateLogo() {
	icuVersion := GetICUVersion()
	icuMajor := strings.Split(icuVersion, ".")[0]

	// Check if ICU version is 66 or higher for Unicode 13 support
	if icuMajor >= "66" {
		rpigreen := "\033[38;2;106;191;75m"
		rpired := "\033[38;2;195;28;74m"
		black := "\033[30m"
		white := "\033[97m"
		fmt.Println("\033[96m" + " █████🭏 🭖█🭀  🭋█🭡 ██◣   ◢██ " + black + "  " + rpigreen + "   " + black + "   " + rpigreen + "   " + black + "           \033[38;2;241;81;27m      \033[38;2;128;204;40m      " + black + "                   \033[0m")
		fmt.Println("\033[96m" + " █▊  🭨█ 🭦█🭐  🭅█🭛 ███◣ ◢███ " + black + "   " + rpigreen + "   " + black + " " + rpigreen + "   " + black + "            \033[38;2;241;81;27m      \033[38;2;128;204;40m      " + black + "           \033[38;2;255;0;0m   " + black + " \033[38;2;255;0;0m   " + black + "\033[0m")
		fmt.Println("\033[96m" + " █████🭪  🭖█🭀🭋█🭡  █▉◥█🭩█◤🮋█ " + black + "  " + rpired + "  " + black + " " + rpired + "   " + black + " " + rpired + "  " + black + "     " + white + "  " + black + "    \033[38;2;241;81;27m      \033[38;2;128;204;40m      " + black + "  " + white + "      " + black + "  \033[38;2;255;0;0m         " + black + "\033[0m")
		fmt.Println("\033[96m" + " █▊  🭨█  🭦█🭐🭅█🭛  █▉ ◥█◤ 🮋█ " + black + " " + rpired + " " + black + " " + rpired + "   " + black + " " + rpired + "   " + black + " " + rpired + " " + black + "  " + white + "      " + black + "  \033[38;2;0;173;239m      \033[38;2;251;188;9m      " + black + "           \033[38;2;255;0;0m       " + black + "\033[0m")
		fmt.Println("\033[96m" + " █████🭠   🭖██🭡   █▉     🮋█ " + black + "  " + rpired + "  " + black + " " + rpired + "   " + black + " " + rpired + "  " + black + "     " + white + "  " + black + "    \033[38;2;0;173;239m      \033[38;2;251;188;9m      " + black + "  " + white + "      " + black + "    \033[38;2;255;0;0m     " + black + "\033[0m")
		fmt.Println(white + "BOTSPOT  VIRTUAL  MACHINE  " + black + "    " + rpired + "     " + black + "             \033[38;2;0;173;239m      \033[38;2;251;188;9m      " + black + "             \033[38;2;255;0;0m   " + black + "   \033[39m\033[0m")
	} else {
		fmt.Println("BVM - botspot virtual machine (you get the boring logo - unicode 13 not found)")
	}
}

func GenerateFunnyMessages() {
	funnyMessages := []string{
		"Making it easy to run a Windows 11 VM on that potato you found in the woods.",
		"Rest easy knowing you are better than every Windows on Raspberry user.",
		"Who knew a sour berry with too many seeds was this capable?",
		"Introducing: a really good way to use up storage space!",
		"How many of these captions do I have to write!?",
		"Is pure Linux better? Yes. But this helps more people stick around.",
		"Pleased to announce an even more efficient way to run inefficient software.",
		"PROTIP: Try selecting the lines of text above for an easter egg! :)",
		"Mixing Linux and Windows... what could possibly go wrong?",
		"Turns out Raspberry and Windows do mix... forming a very crunchy smoothie.",
		"Now your Pi can experience the joy of Windows updates!",
		"Your Pi is about to feel so sophisticated. Or nauseated. Maybe both. Not sure.",
		"Because there is no true difference between 'unsupported' and 'fun challenge.'",
		"It's Linux... It's Windows... It's BVM!!",
		"Like my logo? Build your own logo with: https://github.com/Botspot/unicode-art",
		"With BVM you get the best of both worlds... or at least bragging rights.",
		"Because nobody stopped to ask if this was a good idea. But clearly it is :)",
		"This was a good reason to stay up all night writing bash scripts -Botspot",
		"Let's find out if your hardware has quality memory and cooling. >:)",
		"They said raspberries support windows. Wrong. Now the wall is stained pink.",
		"Run Windows 11 on a Pi and impress... well, mostly yourself.",
		"This is definitely what the Linux developers intended.",
		"Finally, a Windows 11 VM without any dark magic or human sacrifice required.",
		"Linux + Windows = A match made in... well, somewhere interesting.",
		"They said Windows on ARM was a bad idea. And I took that personally.",
		"Perfect for proving that 'just because you can' is a good enough reason!",
		"Achievement unlocked: Running Windows where it absolutely should not be.",
		"Because sometimes, you just really need to run Notepad on a potato.",
		"Minimum requirements? How about minimum suggestions. That's better.",
		"The only thing more surprising than this working... is how well it works.",
		"Like my logo? Build your own logo with: https://github.com/Botspot/unicode-art",
		"With BVM you get the best of both worlds... or at least bragging rights.",
		"Because nobody stopped to ask if this was a good idea. But clearly it is :)",
		"This was a good reason to stay up all night writing bash scripts -Botspot",
		"Let's find out if your hardware has quality memory and cooling. >:)",
		"They said raspberries support windows. Wrong. Now the wall is stained pink.",
		"Run Windows 11 on a Pi and impress... well, mostly yourself.",
		"This is definitely what the Linux developers intended.",
		"Finally, a Windows 11 VM without any dark magic or human sacrifice required.",
		"Linux + Windows = A match made in... well, somewhere interesting.",
		"They said Windows on ARM was a bad idea. And I took that personally.",
		"Perfect for proving that 'just because you can' is a good enough reason!",
		"Achievement unlocked: Running Windows where it absolutely should not be.",
		"Because sometimes, you just really need to run Notepad on a potato.",
		"Minimum requirements? How about minimum suggestions. That's better.",
		"The only thing more surprising than this working... is how well it works.",
		"Think of this as a very elaborate benchmark... for both you and your hardware.",
		// Related messages related to the rewrite
		"Now rewritten in Go! Because apparently bash wasn't fast enough for virtualization and the fact that it's a interpreted language while Go can be both compiled and ran like if it was interpreted (by either using go run or go build).",
		"From bash to Go: Because someone said 'gofmt' and we took it literally.",
		"BVM Go edition: Now with 100% more goroutines and 50% less sanity!",
		"Rewritten in Go because we needed more ways to handle errors... err != nil",
		"Go rewrite: Because 'if err != nil' is the new 'set -e'.",
		"Trading bash arrays for Go slices... and immediately regretting it.",
		"Go modules: Making dependency management almost as fun as QEMU configuration!",
		"From shell scripts to compiled binaries: Evolution or devolution? You decide!",
		"BVM in Go: Because someone thought bash wasn't verbose enough. And that is if you cause it to panic.",
		"Now featuring proper error handling! (Translation: more ways for things to fail and more ways to handle them by having more return variables assigned to a function)",
		"Go version: Where every function returns (result, error) and your sanity.",
		"Migrated to Go because we missed having a compiler yell at us back in the days of C.",
		"Go rewrite: Making Rob Pike proud and your CPU slightly warmer.",
		"From #!/bin/bash to package main: A journey of questionable decisions.",
		"Now with channels! (No, not the TV kind, the concurrent kind)",
		"Go version: Because 'gopher' sounds cuter than 'basher'.",
		"Rewritten in Go: Where interfaces are satisfied and developers are confused.",
		"BVM-Go: Proving that any problem can be solved with more goroutines.",
		"From bash to Go: Trading simplicity for... well, we're still figuring that out.",
		"Go rewrite: Because someone said 'needs more type safety' and we listened. We are glad that we didn't choose Rust because it's more insane and it enforces more rules than we need.",
		"Now featuring proper structs! (Your VM configs have never been so organized)",
		"Go version: Where 'go run' is the new './bvm' and twice as satisfying. Or you could just compile it.",
		"Migrated to Go because we wanted our errors to have stack traces.",
		"BVM Go edition: Making the impossible slightly more impossible, but with better performance at the cost of your binaries being 100x larger!",
		"From shell to Go: Because apparently we needed our VM manager to compile.",
		"Go rewrite: Where every variable has a type and every type has an opinion.",
		"Now in Go! Because someone thought our bash scripts weren't complex enough.",
		"This was a good reason to stay up all night learning the Go language and writing this rewrite -matu6968",
	}
	randomMessage := funnyMessages[rand.Intn(len(funnyMessages))]
	fmt.Println("\033[96m" + randomMessage + "\033[0m")
}

// Input: folder to check. Output: show many bytes can fit before the disk is full
func GetSpaceFree(folder string) int64 {
	df, err := os.Stat(folder)
	if err != nil {
		Error("Failed to get space free: " + err.Error())
	}
	return df.Size()
}

// return 0 if the $1 PID is running, otherwise 1
// taken from pi-apps-go api
func ProcessExists(pid int) bool {
	_, err := os.FindProcess(pid)
	return err == nil
}

// wrapper for umount command - try several times if it is still mounted
// necessary because first umount attempt usually says "target is busy"
// taken from pi-apps-go api
func UmountRetry(folder string) {
	tries := 0
	exec.Command("sync")
	for exec.Command("mountpoint", "-q", folder).Run() == nil {
		exec.Command("umount", folder).Run()
		os.Remove(folder)
		exec.Command("sync")
		time.Sleep(1 * time.Second)
		if tries == 10 {
			Warning("Could not unmount " + folder + ", unmounting it lazily.")
			os.Remove(folder)
		}
		tries++
	}
}

// GetCodename gets the codename of the system using lsb_release
// taken from pi-apps-go api
func GetCodename() string {
	// Check if lsb_release is installed
	if !CommandExists("lsb_release") {
		Status("Installing lsb-release, please wait...")
		runCommand("sudo", "apt", "install", "-y", "lsb-release")
	}

	// Check for upstream release first (Ubuntu derivatives)
	output, err := exec.Command("lsb_release", "-a", "-u").CombinedOutput()
	if err == nil && !bytes.Contains(output, []byte("command not found")) {
		// This is a Ubuntu Derivative
		re := regexp.MustCompile(`Distributor ID:\s+(.*)\nDescription:\s+(.*)\nRelease:\s+(.*)\nCodename:\s+(.*)`)
		matches := re.FindStringSubmatch(string(output))
		if len(matches) >= 5 {
			HostSystemID = matches[1]
			HostSystemDesc = matches[2]
			HostSystemVer = matches[3]
			HostSystemCode = matches[4]

			// Now get the original info
			output, err = exec.Command("lsb_release", "-a").CombinedOutput()
			if err == nil {
				matches = re.FindStringSubmatch(string(output))
				if len(matches) >= 5 {
					OrigSystemID = matches[1]
					OrigSystemDesc = matches[2]
					OrigSystemVer = matches[3]
					OrigSystemCode = matches[4]
				}
			}
		}
	} else if _, err := os.Stat("/etc/upstream-release/lsb-release"); err == nil {
		// Ubuntu 22.04+ Linux Mint no longer includes the lsb_release -u option
		content, err := os.ReadFile("/etc/upstream-release/lsb-release")
		if err == nil {
			idRe := regexp.MustCompile(`DISTRIB_ID=(.*)`)
			descRe := regexp.MustCompile(`DISTRIB_DESCRIPTION=(.*)`)
			verRe := regexp.MustCompile(`DISTRIB_RELEASE=(.*)`)
			codeRe := regexp.MustCompile(`DISTRIB_CODENAME=(.*)`)

			if matches := idRe.FindStringSubmatch(string(content)); len(matches) > 1 {
				HostSystemID = strings.Trim(matches[1], "\"")
			}
			if matches := descRe.FindStringSubmatch(string(content)); len(matches) > 1 {
				HostSystemDesc = strings.Trim(matches[1], "\"")
			}
			if matches := verRe.FindStringSubmatch(string(content)); len(matches) > 1 {
				HostSystemVer = strings.Trim(matches[1], "\"")
			}
			if matches := codeRe.FindStringSubmatch(string(content)); len(matches) > 1 {
				HostSystemCode = strings.Trim(matches[1], "\"")
			}

			// Now get the original info
			output, err = exec.Command("lsb_release", "-a").CombinedOutput()
			if err == nil {
				re := regexp.MustCompile(`Distributor ID:\s+(.*)\nDescription:\s+(.*)\nRelease:\s+(.*)\nCodename:\s+(.*)`)
				matches := re.FindStringSubmatch(string(output))
				if len(matches) >= 5 {
					OrigSystemID = matches[1]
					OrigSystemDesc = matches[2]
					OrigSystemVer = matches[3]
					OrigSystemCode = matches[4]
				}
			}
		}
	} else if _, err := os.Stat("/etc/lsb-release.diverted"); err == nil {
		// Ubuntu 22.04+ Pop!_OS uses a different file
		content, err := os.ReadFile("/etc/lsb-release.diverted")
		if err == nil {
			idRe := regexp.MustCompile(`DISTRIB_ID=(.*)`)
			descRe := regexp.MustCompile(`DISTRIB_DESCRIPTION=(.*)`)
			verRe := regexp.MustCompile(`DISTRIB_RELEASE=(.*)`)
			codeRe := regexp.MustCompile(`DISTRIB_CODENAME=(.*)`)

			if matches := idRe.FindStringSubmatch(string(content)); len(matches) > 1 {
				HostSystemID = strings.Trim(matches[1], "\"")
			}
			if matches := descRe.FindStringSubmatch(string(content)); len(matches) > 1 {
				HostSystemDesc = strings.Trim(matches[1], "\"")
			}
			if matches := verRe.FindStringSubmatch(string(content)); len(matches) > 1 {
				HostSystemVer = strings.Trim(matches[1], "\"")
			}
			if matches := codeRe.FindStringSubmatch(string(content)); len(matches) > 1 {
				HostSystemCode = strings.Trim(matches[1], "\"")
			}

			// Now get the original info
			output, err = exec.Command("lsb_release", "-a").CombinedOutput()
			if err == nil {
				re := regexp.MustCompile(`Distributor ID:\s+(.*)\nDescription:\s+(.*)\nRelease:\s+(.*)\nCodename:\s+(.*)`)
				matches := re.FindStringSubmatch(string(output))
				if len(matches) >= 5 {
					OrigSystemID = matches[1]
					OrigSystemDesc = matches[2]
					OrigSystemVer = matches[3]
					OrigSystemCode = matches[4]
				}
			}
		}
	} else {
		// Regular system, not a derivative
		output, err := exec.Command("lsb_release", "-a").CombinedOutput()
		if err == nil {
			re := regexp.MustCompile(`Distributor ID:\s+(.*)\nDescription:\s+(.*)\nRelease:\s+(.*)\nCodename:\s+(.*)`)
			matches := re.FindStringSubmatch(string(output))
			if len(matches) >= 5 {
				HostSystemID = matches[1]
				HostSystemDesc = matches[2]
				HostSystemVer = matches[3]
				HostSystemCode = matches[4]
			}
		}
	}
	return HostSystemCode
}

// Helper function to run shell commands
// taken from pi-apps-go api
func runCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// PackageInstalled checks if a package is installed
// taken from pi-apps-go api
func PackageInstalled(packageName string) bool {
	if packageName == "" {
		Error("PackageInstalled(): no package specified!")
		return false
	}

	// Use dpkg to check if the package is installed
	cmd := exec.Command("dpkg", "-s", packageName)
	if err := cmd.Run(); err != nil {
		return false
	}

	return true
}

// QemuNewerThan checks if QEMU is newer than a specified version, return true if yes, return false if no
func QemuNewerThan(version string) bool {
	qemuType := "qemu-system-aarch64"
	if runtime.GOARCH == "arm64" {
		qemuType = "qemu-system-aarch64"
	} else if runtime.GOARCH == "arm" {
		qemuType = "qemu-system-arm"
	} else if runtime.GOARCH == "amd64" {
		qemuType = "qemu-system-x86_64"
	} else {
		Error("QemuNewerThan(): unsupported architecture: " + runtime.GOARCH)
	}
	qemuVersion, err := runCommand(qemuType, "--version")
	if err != nil {
		Error("QemuNewerThan(): failed to get qemu version: " + err.Error())
	}
	qemuVersion = strings.Split(qemuVersion, " ")[2]
	compareVersion := version
	return qemuVersion >= compareVersion
}

// InstallDependencies installs the dependencies for the current architecture for BVM
func InstallDependencies() {
	requiredCommands := []string{
		"git",
		"mkisofs",
		"qemu-img",
		"remmina",
		"nmap",
		"wiminfo",
		"socat",
		"nc",
		"seabios",
		"ipxe-qemu",
		"wimtools",
		"ntfs-3g",
	}
	if runtime.GOARCH == "arm64" {
		requiredCommands = append(requiredCommands, "qemu-system-aarch64")
	} else if runtime.GOARCH == "arm" {
		requiredCommands = append(requiredCommands, "qemu-system-arm")
	} else if runtime.GOARCH == "amd64" {
		requiredCommands = append(requiredCommands, "qemu-system-x86_64")
	} else {
		Error("InstallDependencies(): unsupported architecture: " + runtime.GOARCH)
	}

	if os.Getenv("XDG_SESSION_TYPE") == "wayland" {
		requiredCommands = append(requiredCommands, "wlfreerdp")
	} else {
		requiredCommands = append(requiredCommands, "xfreerdp")
	}

	packages := []string{
		"git",
		"genisoimage",
		"qemu-utils",
		"qemu-system-gui",
		"remmina",
		"remmina-plugin-rdp",
		"nmap",
		"seabios",
		"ipxe-qemu",
		"wimtools",
		"ntfs-3g",
		"netcat-traditional",
		"p7zip-full",
		"unzip",
	}

	if runtime.GOARCH == "arm64" {
		packages = append(packages, "qemu-efi-aarch64", "qemu-system-aarch64")
	} else if runtime.GOARCH == "arm" {
		packages = append(packages, "qemu-efi-arm", "qemu-system-arm")
	} else if runtime.GOARCH == "amd64" {
		packages = append(packages, "qemu-efi-x86_64", "qemu-system-x86_64")
	} else {
		Error("InstallDependencies(): unsupported architecture: " + runtime.GOARCH)
	}

	installrequired := false

	// Check if the packages are installed and available, if not, install them
	for _, command := range requiredCommands {
		if !CommandExists(command) {
			installrequired = true
		}
	}
	for _, packageName := range packages {
		if !PackageInstalled(packageName) {
			installrequired = true
		}
	}

	// Not all dependencies have commands. Check for absent needed files
	if _, err := os.Stat("/usr/share/qemu-efi-aarch64/QEMU_EFI.fd"); os.IsNotExist(err) {
		installrequired = true
	}
	if _, err := os.Stat("/usr/lib/*/remmina/plugins/remmina-plugin-rdp.so"); os.IsNotExist(err) {
		installrequired = true
	}
	if _, err := os.Stat("/usr/share/seabios/vgabios-ramfb.bin"); os.IsNotExist(err) {
		installrequired = true
	}
	if _, err := os.Stat("/usr/lib/ipxe/qemu/pxe-virtio.rom"); os.IsNotExist(err) {
		installrequired = true
	}

	// Hide the changelog
	os.Setenv("DEBIAN_FRONTEND", "noninteractive")
	// install dependencies with apt, if apt is found
	if installrequired && CommandExists("apt") {
		Status("Installing dependencies, please wait...")
		// Don't change the flags here in this apt command without updating the apt-detecting code on the pi-apps install script for BVM
		packagesToInstall := []string{}
		for _, packageName := range packages {
			if !PackageInstalled(packageName) {
				packagesToInstall = append(packagesToInstall, packageName)
			}
		}
		// Run apt update only if necessary
		if _, err := os.Stat("/var/cache/apt/pkgcache.bin"); os.IsNotExist(err) {
			runCommand("sudo", "apt", "update")
		}
		// Install the packages
		if _, err := runCommand("sudo", "apt", "install", "-y", strings.Join(packagesToInstall, " ")); err != nil {
			Error("InstallDependencies(): APT failed to install required dependencies")
		}
		// Upgrade qemu to version from bookworm-backports
		if GetCodename() == "bookworm" {
			Status("Upgrading QEMU to version from bookworm-backports repository...")
			if runtime.GOARCH == "arm64" {
				runCommand("sudo", "apt", "install", "-y", "-t", "bookworm-backports", "--only-upgrade", "qemu-system-aarch64", "qemu-system-gui")
			} else if runtime.GOARCH == "arm" {
				runCommand("sudo", "apt", "install", "-y", "-t", "bookworm-backports", "--only-upgrade", "qemu-system-arm", "qemu-system-gui")
			} else if runtime.GOARCH == "amd64" {
				runCommand("sudo", "apt", "install", "-y", "-t", "bookworm-backports", "--only-upgrade", "qemu-system-x86_64", "qemu-system-gui")
			} else {
				Error("InstallDependencies(): unsupported architecture: " + runtime.GOARCH)
			}
			StatusGreen("Package installation complete!")
		}
	} else if installrequired {
		Error("InstallDependencies(): BVM needs these dependencies: " + strings.Join(packages, " ") + "\nBVM could not install them for you as your distro is not based on Debian, or at least, the apt command could not be found.\nPlease install the dependencies yourself, then try again.\n If you are a plugin developer for BVM, please add the dependencies to the plugin's installDependencies() function and replace the installDependencies() function with your own.")
	}
	// Make menu launcher for GUI mode
	if _, err := os.Stat("~/.local/share/applications/bvm-go.desktop"); os.IsNotExist(err) {
		os.MkdirAll("~/.local/share/applications", 0755)
		os.WriteFile("~/.local/share/applications/bvm-go.desktop", []byte("[Desktop Entry]\nName=Botspot Virtual Machine (Go edition)\nComment=Simple GUI for running Windows 10/11 with BVM but rewritten in Go instead\nExec=bvm gui\nIcon=bvm\nTerminal=false\nStartupWMClass=wlfreerdp\nType=Application\nCategories=Office\nStartupNotify=true"), 0644)
	}
	// Make terminal command (may only work on future shell logins)
	if _, err := os.Stat("~/.local/bin/bvm-go"); os.IsNotExist(err) {
		os.MkdirAll("~/.local/bin", 0755)
		os.Symlink("bvm-go", "~/.local/bin/bvm-go")
		Status("From now on you should be able to run BVM simply with 'bvm-go'. The command might not be detected by this terminal, but will work on future terminals.")
	}
}

// CommandExists is a helper function that checks if a command is available in the system, return true if yes, return false if no
func CommandExists(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// UpdateCheck checks for updates and reloads the script if necessary
func UpdateCheck() {
	if BVMConfig.DisableUpdates {
		Status("Updates disabled, skipping update check")
		return
	}
	localhash, err := runCommand("git", "rev-parse", "HEAD")
	if err != nil {
		Error("UpdateCheck(): failed to get local hash: " + err.Error())
	}
	account, repo := GetGitUrl()
	if account == "" || repo == "" {
		Warning("UpdateCheck(): failed to get git URL, setting to default")
		account = "pi-apps-go"
		repo = "bvm-go"
		return
	}
	latesthash, err := runCommand("git", "ls-remote", "https://github.com/"+account+"/"+repo, "HEAD")
	if err != nil {
		Error("UpdateCheck(): failed to get latest hash: " + err.Error())
	}
	if localhash != latesthash {
		Status("Updates available, recompiling and running...")
		runCommand("git", "restore", ".")
		runCommand("git", "pull")
		runCommand("make", "install")
		runCommand("bvm-go", "gui")
	}
}

// Helper function to get the git URL from the git_url file
//
//	account - the account name
//	repo - the repository name
func GetGitUrl() (account, repo string) {
	piAppsDir := os.Getenv("BVM_DIR")
	gitURLPath := filepath.Join(piAppsDir, "resources", "git_url")
	if _, err := os.Stat(gitURLPath); err == nil {
		// Read git URL from file
		gitURLBytes, err := os.ReadFile(gitURLPath)
		if err == nil {
			gitURL := strings.TrimSpace(string(gitURLBytes))

			// Parse account and repository from URL
			parts := strings.Split(gitURL, "/")
			if len(parts) >= 2 {
				account := parts[len(parts)-2]
				repo := parts[len(parts)-1]
				return account, repo
			}
		}
	}
	return account, repo
}

// CreateNewVM initializes a new VM directory with configuration files
func CreateNewVM(vmDir string) error {
	Status("Creating new VM directory: " + vmDir)

	// Create the VM directory
	if err := os.MkdirAll(vmDir, 0755); err != nil {
		return fmt.Errorf("failed to create VM directory '%s': %v", vmDir, err)
	}

	// Get the executable directory to find resources
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}
	execDir := filepath.Dir(execPath)
	resourcesDir := filepath.Join(execDir, "resources")

	// If resources dir doesn't exist relative to executable, try relative to source
	if _, err := os.Stat(resourcesDir); os.IsNotExist(err) {
		// Try to find resources relative to where we might be running from source
		wd, _ := os.Getwd()
		resourcesDir = filepath.Join(wd, "resources")
		if _, err := os.Stat(resourcesDir); os.IsNotExist(err) {
			return fmt.Errorf("could not find resources directory")
		}
	}

	// Copy configuration file
	configSrc := filepath.Join(resourcesDir, "bvm-config.toml")
	configDst := filepath.Join(vmDir, "bvm-config.toml")
	if err := copyFile(configSrc, configDst); err != nil {
		return fmt.Errorf("failed to copy bvm-config: %v", err)
	}
	Status("  ✓ Copied configuration file")

	// Copy remmina config
	remminaSrc := filepath.Join(resourcesDir, "connect.remmina")
	remminaDst := filepath.Join(vmDir, "connect.remmina")
	if err := copyFile(remminaSrc, remminaDst); err != nil {
		return fmt.Errorf("failed to copy connect.remmina: %v", err)
	}
	Status("  ✓ Copied remmina configuration")

	// Create GUI steps complete file
	stepsFile := filepath.Join(vmDir, "gui-steps-complete")
	if err := os.WriteFile(stepsFile, []byte("1"), 0644); err != nil {
		return fmt.Errorf("failed to create gui-steps-complete file: %v", err)
	}
	Status("  ✓ Created GUI progress tracking file")

	// copy over autounattend.xml
	autounattendSrc := filepath.Join(resourcesDir, "autounattend.xml")
	autounattendDst := filepath.Join(vmDir, "unattended", "autounattend.xml")
	// make the unattended directory if it doesn't exist
	if err := os.MkdirAll(filepath.Join(vmDir, "unattended"), 0755); err != nil {
		return fmt.Errorf("failed to create unattended directory: %v", err)
	}
	if err := copyFile(autounattendSrc, autounattendDst); err != nil {
		return fmt.Errorf("failed to copy autounattend.xml: %v", err)
	}
	Status("  ✓ Copied autounattend.xml")

	StatusGreen("Successfully created new VM at: " + vmDir)
	Status("You should now be ready for the next step: bvm download " + vmDir)

	return nil
}

// Function to check if this CPU supports latest Windows 11 on ARM64, which requires the atomics CPU instruction
func IsAtomicsCPU() bool {
	if runtime.GOARCH != "arm64" {
		return false
	}
	cpuInfo, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return false
	}
	return strings.Contains(string(cpuInfo), "atomics")
}

// Function to check if this is an ARMv9 CPU (which doesn't support 32-bit natively)
func IsARMv9CPU() bool {
	if runtime.GOARCH != "arm64" {
		return false
	}
	cpuInfo, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return false
	}

	cpuInfoStr := string(cpuInfo)

	// Check for ARMv9 CPU part numbers
	// ARMv9 cores include: Cortex-X1, Cortex-A78, Cortex-A710, Cortex-X2, Cortex-A510, etc.
	// Part numbers from https://github.com/bp0/armids/blob/master/arm.ids
	armv9Parts := []string{
		"0xd44", // Cortex-X1
		"0xd41", // Cortex-A78
		"0xd46", // Cortex-A510
		"0xd47", // Cortex-A710
		"0xd48", // Cortex-X2
		"0xd49", // Neoverse-N2
		"0xd4a", // Neoverse-E1
		"0xd4b", // Cortex-A78AE
		"0xd4c", // Cortex-X1C
		"0xd4d", // Cortex-A715
		"0xd4e", // Cortex-X3
		"0xd4f", // Neoverse-V2
		"0xd80", // Cortex-A520
		"0xd81", // Cortex-A720
		"0xd82", // Cortex-X4
		"0xd84", // Neoverse-V3
		"0xd85", // Cortex-X925
		"0xd87", // Cortex-A725
	}

	for _, part := range armv9Parts {
		if strings.Contains(cpuInfoStr, part) {
			return true
		}
	}

	// Also check for explicit ARMv9 architecture declaration
	// Some systems may report "CPU architecture: 8" even for ARMv9 cores
	// but we can also check for specific ARMv9 features
	if strings.Contains(cpuInfoStr, "CPU architecture: 9") {
		return true
	}

	return false
}

// Function to check if this CPU supports big.LITTLE cpus such as RK3588. List the performance cores, or all cores if all are equal. Return 1 if performance cores detected.
func ListCoresToUse() ([]string, bool) {
	if runtime.GOARCH != "arm64" {
		return nil, false
	}
	cpuInfo, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return nil, false
	}

	// Parse CPU info similar to bash: grep '^CPU part\|^processor\|^$' | tr '\n' '\r' | sed 's/\r\r/\n/g ; s/\r/ /g'
	lines := strings.Split(string(cpuInfo), "\n")
	var coreEntries []string
	var currentEntry strings.Builder

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "processor") || strings.HasPrefix(line, "CPU part") {
			currentEntry.WriteString(line + " ")
		} else if line == "" && currentEntry.Len() > 0 {
			coreEntries = append(coreEntries, strings.TrimSpace(currentEntry.String()))
			currentEntry.Reset()
		}
	}
	// Add the last entry if it exists
	if currentEntry.Len() > 0 {
		coreEntries = append(coreEntries, strings.TrimSpace(currentEntry.String()))
	}

	// List of A76+ core part numbers from https://github.com/bp0/armids/blob/master/arm.ids
	a76CoreParts := []string{"0xd0b", "0xd0c", "0xd0d", "0xd13", "0xd20", "0xd21", "0xd4a"}

	var allCores []string
	var a76Cores []string

	for _, entry := range coreEntries {
		// Extract processor number
		processorMatch := regexp.MustCompile(`processor\s*:\s*(\d+)`).FindStringSubmatch(entry)
		if len(processorMatch) > 1 {
			processorNum := processorMatch[1]
			allCores = append(allCores, processorNum)

			// Check if this is an A76+ core
			for _, partNum := range a76CoreParts {
				if strings.Contains(entry, partNum) {
					a76Cores = append(a76Cores, processorNum)
					break
				}
			}
		}
	}

	// If there are some A76 cores, and list of A76 cores is not the same as list of total cores
	if len(a76Cores) > 0 && len(a76Cores) != len(allCores) {
		// Performance cores detected, return only them
		return a76Cores, true
	} else {
		// All cores are the same, return them all
		return allCores, false
	}
}

// Function to run a command in a GUI terminal using the terminal-run script
func RunInTerminal(command string, title string) {
	runCommand(filepath.Join(BVMDir, "resources", "terminal-run"), command, title)
}

// uncomment and recompile to skip running steps for debugging
//func RunInTerminal(command string, title string) {
//  StepComplete("$(echo "$1" | awk '{print $NF}')")
//  TODO: replace StepComplete with a function that does something useful
//}
