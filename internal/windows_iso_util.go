// this is a work in progress, so don't expect much yet

package internal

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
)

// Progress model for downloads
type downloadModel struct {
	progress progress.Model
	filename string
	total    int64
	current  int64
	done     bool
	err      error
}

// Spinner model for operations
type operationModel struct {
	spinner   spinner.Model
	operation string
	done      bool
	err       error
}

// Progress messages
type progressMsg struct {
	current int64
	total   int64
}

type downloadCompleteMsg struct {
	err error
}

type operationCompleteMsg struct {
	err error
}

// ISO creation progress model
type isoProgressModel struct {
	progress  progress.Model
	operation string
	percent   float64
	status    string
	done      bool
	err       error
}

// ISO progress messages
type isoProgressMsg struct {
	percent float64
	status  string
}

type isoCompleteMsg struct {
	err error
}

// Microsoft API response structures
type SKUInfo struct {
	Id                string `json:"Id"`
	LocalizedLanguage string `json:"LocalizedLanguage"`
	Language          string `json:"Language"`
}

type SKUResponse struct {
	Skus []SKUInfo `json:"Skus"`
}

type ProductDownloadOption struct {
	Uri string `json:"Uri"`
}

type DownloadResponse struct {
	ProductDownloadOptions []ProductDownloadOption `json:"ProductDownloadOptions"`
}

func (m downloadModel) Init() tea.Cmd {
	return nil
}

func (m downloadModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case progressMsg:
		m.current = msg.current
		m.total = msg.total
		return m, nil
	case downloadCompleteMsg:
		m.done = true
		m.err = msg.err
		return m, tea.Quit
	}

	var cmd tea.Cmd
	progressModel, progressCmd := m.progress.Update(msg)
	if p, ok := progressModel.(progress.Model); ok {
		m.progress = p
	}
	cmd = tea.Batch(cmd, progressCmd)
	return m, cmd
}

func (m downloadModel) View() string {
	if m.done {
		if m.err != nil {
			return fmt.Sprintf("%s Download failed: %v\n", StatusErrorRed(), m.err)
		}
		return fmt.Sprintf("%s Downloaded %s successfully!\n", StatusCheckGreen(), m.filename)
	}

	percent := float64(m.current) / float64(m.total)
	if m.total == 0 {
		percent = 0
	}

	// Convert bytes to GB for better readability
	currentGB := float64(m.current) / (1024 * 1024 * 1024)
	totalGB := float64(m.total) / (1024 * 1024 * 1024)

	return fmt.Sprintf("📥 Downloading %s...\n%s %.1f%% (%.2fGB/%.2fGB)\n",
		m.filename,
		m.progress.ViewAs(percent),
		percent*100,
		currentGB,
		totalGB)
}

func (m operationModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m operationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case operationCompleteMsg:
		m.done = true
		m.err = msg.err
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m operationModel) View() string {
	if m.done {
		if m.err != nil {
			return fmt.Sprintf("%s %s failed: %v\n", StatusErrorRed(), m.operation, m.err)
		}
		return fmt.Sprintf("%s %s completed successfully!\n", StatusCheckGreen(), m.operation)
	}

	return fmt.Sprintf("%s %s...\n", m.spinner.View(), m.operation)
}

func (m isoProgressModel) Init() tea.Cmd {
	return nil
}

func (m isoProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case isoProgressMsg:
		m.percent = msg.percent
		m.status = msg.status
		return m, nil
	case isoCompleteMsg:
		m.done = true
		m.err = msg.err
		return m, tea.Quit
	}

	var cmd tea.Cmd
	progressModel, progressCmd := m.progress.Update(msg)
	if p, ok := progressModel.(progress.Model); ok {
		m.progress = p
	}
	cmd = tea.Batch(cmd, progressCmd)
	return m, cmd
}

func (m isoProgressModel) View() string {
	if m.done {
		if m.err != nil {
			return fmt.Sprintf("%s %s failed: %v\n", StatusErrorRed(), m.operation, m.err)
		}
		return fmt.Sprintf("%s %s completed successfully!\n", StatusCheckGreen(), m.operation)
	}

	progress := m.percent / 100.0
	if progress > 1.0 {
		progress = 1.0
	}

	return fmt.Sprintf("💿 %s...\n%s %.1f%% %s\n",
		m.operation,
		m.progress.ViewAs(progress),
		m.percent,
		m.status)
}

func ListDownloadLanguages() string {
	return "ar-sa:Arabic\npt-br:Brazilian Portuguese\nbg-bg:Bulgarian\nzh-cn:Chinese (Simplified)\nzh-tw:Chinese (Traditional)\nhr-hr:Croatian\ncs-cz:Czech\nda-dk:Danish\nnl-nl:Dutch\nen-us:English (United States)\nen-gb:English International\net-ee:Estonian\nfi-fi:Finnish\nfr-fr:French\nfr-ca:French Canadian\nde-de:German\nel-gr:Greek\nhe-il:Hebrew\nhu-hu:Hungarian\nit-it:Italian\nja-jp:Japanese\nko-kr:Korean\nlv-lv:Latvian\nlt-lt:Lithuanian\nnb-no:Norwegian\npl-pl:Polish\npt-pt:Portuguese\nro-ro:Romanian\nru-ru:Russian\nsr-latn-rs:Serbian Latin\nsk-sk:Slovak\nsl-si:Slovenian\nes-es:Spanish\nes-mx:Spanish (Mexico)\nsv-se:Swedish\nth-th:Thai\ntr-tr:Turkish\nuk-ua:Ukrainian"
}

// DownloadWindowsISO is the main function that downloads the Windows ISO image and prepares it for use in the VM
//
//	language: the language of the Windows ISO image
//	vmdir: the directory to store the Windows ISO image
//	release: the major version of the Windows ISO image
//	version: the build version of the Windows ISO image (valid are: 22631, 15035, none and optional, assume latest if not set)
//	arch: the architecture of the Windows ISO image
//	edition: the edition of the Windows ISO image (for example Home or Pro, only valid for build 22631 and optional, if not set then default to Pro)
func DownloadWindowsISO(language string, vmdir string, release string, version string, arch string, edition string, customISOPath ...string) {
	Status("Starting Windows ISO download process...")

	// Handle custom ISO case
	if release == "Custom ISO" {
		if len(customISOPath) == 0 || customISOPath[0] == "" {
			ErrorNoExit("Custom ISO path not provided")
			return
		}

		// Extract custom VirtIO path if provided (second parameter)
		var customVirtioPath string
		if len(customISOPath) > 1 {
			customVirtioPath = customISOPath[1]
		}

		Status("Processing custom Windows ISO...")
		if err := ProcessCustomWindowsISO(customISOPath[0], vmdir, customVirtioPath); err != nil {
			ErrorNoExit("Failed to process custom Windows ISO: " + err.Error())
			return
		}

		StatusGreen("Custom Windows ISO processed successfully!")
		return
	}

	// Create VM directory if it doesn't exist
	if err := os.MkdirAll(vmdir, 0755); err != nil {
		ErrorNoExit("Failed to create VM directory: " + err.Error())
		return
	}

	// Rest of the existing download logic for standard Windows versions...
	installerISO := filepath.Join(vmdir, "installer.iso")

	// Check if installer.iso already exists
	if _, err := os.Stat(installerISO); err == nil {
		Status("installer.iso already exists, proceeding to virtio driver download")
		if err := DownloadVirtioDrivers(vmdir, arch); err != nil {
			ErrorNoExit("Failed to download virtio drivers: " + err.Error())
			return
		}
		return
	}

	Debug("Download parameters:")
	Debug("  Language: " + language)
	Debug("  VM Directory: " + vmdir)
	Debug("  Release: " + release)
	Debug("  Version: " + version)
	Debug("  Architecture: " + arch)
	Debug("  Edition: " + edition)

	// Check if all required variables are set
	if language == "" || vmdir == "" || release == "" || arch == "" {
		ErrorNoExit("Missing required variables")
		return
	}

	// Typical editions you will find on the ISO's provided by the Microsoft website as a end user
	// Enterprise and Enterprise N is only seen on a seperate branch of the Windows 10/11 ISO's not seen as a end user
	// only valid for Windows 11 build 22631 image creation because the prebuilt ISO images already have the needed editions built in
	validEditions := []string{
		"Home", "Home N", "Pro", "Pro N", "Pro Education", "Pro for Workstations", "Pro N for Workstations",
		"Pro Education N", "Education", "Education N",
	}

	if version == "22631" {
		// Check if the edition is valid if not blank and if the build version is 22631
		if edition != "" && !slices.Contains(validEditions, edition) {
			ErrorNoExit("Invalid edition: " + edition)
			return
		} else if edition == "" {
			edition = "Pro"
		}
	}
	// Print out a message according to the Windows version and it's major version
	if version == "22631" {
		Status("Downloading Windows 11 ARM64 build " + version + " (" + language + ", last compatible version for ARMv8.0 CPUs)")
	} else if version == "latest" || version == "" {
		version = "(latest)"
	} else {
		if release == "11" {
			if arch == "ARM64" {
				Status("Downloading Windows 11 ARM64 build " + version + " (" + language + ")")
			} else if arch == "x64" {
				Status("Downloading Windows 11 x64 build " + version + " (" + language + ")")
			} else {
				ErrorNoExit("Invalid architecture: " + arch)
				return
			}
		} else if release == "10" {
			if arch == "ARM64" {
				Status("Downloading Windows 10 ARM64 build " + version + " (" + language + ")")
			} else if arch == "x64" {
				Status("Downloading Windows 10 x64 build " + version + " (" + language + ")")
				// Leaked Windows 10 ARMv7 build 15035 is the only compatible version for ARMv7 CPUs and may not work with QEMU drivers, so keep it in mind that it could be removed if it's not working
			} else if arch == "ARMv7" && version == "15035" {
				Status("Downloading Windows 10 ARMv7 build " + version + " (" + language + ", only compatible version for ARMv7 CPUs)")
			} else if arch == "ARMv7" {
				ErrorNoExit("Only leaked Windows 10 ARMv7 build 15035 is compatible with ARMv7 CPUs, all other versions are not supported")
				return
			} else {
				ErrorNoExit("Invalid architecture/build: " + arch + " " + version)
				return
			}
		} else {
			ErrorNoExit("Invalid version: " + version)
			return
		}
	}

	// Get download link, size, and SHA1 hash for ESD if version is 22631
	// otherwise, get the download link from the Microsoft website or files.open-rt.party for Windows 10 ARMv7 leaked build
	var URL string // Used by non-22631 code paths
	if arch == "ARM64" {
		if version == "22631" {
			URL = "https://worproject.com/dldserv/esd/getcatalog.php?build=22631.2861&arch=ARM64&edition=Professional"
		} else {
			URL = "https://www.microsoft.com/en-us/software-download/windows11arm64"
		}
	} else if arch == "x64" {
		if release == "11" || release == "Windows 11" {
			URL = "https://www.microsoft.com/en-us/software-download/windows11"
		} else if release == "10" || release == "Windows 10" {
			URL = "https://www.microsoft.com/en-us/software-download/windows10"
		} else {
			ErrorNoExit("Invalid release: " + release)
			return
		}
	} else if arch == "ARMv7" && version == "15035" {
		// There are 2 ways to download the Windows 10 ARMv7 leaked build 15035 without needing an account, either from archive.org or files.open-rt.party
		// The archive.org version is slower to download, so we will download the files.open-rt.party version
		// as a fallback replace the URL with the archive.org version if it were to go down in the future
		URL = "https://files.open-rt.party/10.0.15035.0.armfre.rs2_release.170209-1535.7z"
	}

	// for Windows 11 build 22631, we need handle the ESD download link from the Microsoft's update servers via a proxy server provided by the windows on r project
	if version == "22631" && arch == "ARM64" && (release == "11" || release == "Windows 11") {
		// Convert from pretty language name to short-code used by esd releases
		langCode := getLanguageCode(language)
		if langCode == "" {
			ErrorNoExit("Language must be specified in download_language variable. Get list of available languages by running bvm list-languages")
			return
		}

		// Get ESD catalog
		fmt.Println("  - Getting ESD download URL...")

		// Get the Windows ESD catalog
		resp, err := http.Get(URL)
		if err != nil {
			ErrorNoExit("Could not get list of Windows ESD releases: " + err.Error())
			return
		}
		defer resp.Body.Close()

		catalogBody, err := io.ReadAll(resp.Body)
		if err != nil {
			ErrorNoExit("Failed to read catalog response: " + err.Error())
			return
		}

		catalog := string(catalogBody)
		if catalog == "" {
			ErrorNoExit("Could not get list of Windows ESD releases. If you ran this step several times recently, the site likely temporarily banned your IP address.")
			return
		}

		// Parse catalog to extract language-specific section
		catalog = parseCatalogForLanguage(catalog, langCode)
		if catalog == "" {
			ErrorNoExit("Could not find language " + langCode + " in catalog")
			return
		}

		// Create esdextract directory
		esdExtractDir := filepath.Join(vmdir, "esdextract")
		if err := os.RemoveAll(esdExtractDir); err != nil {
			ErrorNoExit("Failed to remove esdextract folder: " + err.Error())
			return
		}
		if err := os.MkdirAll(esdExtractDir, 0755); err != nil {
			ErrorNoExit("Directory creation failed: " + err.Error())
			return
		}

		// Extract download URL, size, and SHA1 hash
		downloadURL := extractXMLValue(catalog, "FilePath")
		expectedSHA1 := extractXMLValue(catalog, "Sha1")
		sourceFile := filepath.Join(vmdir, "image.esd")

		// Download ESD if not already present or invalid
		if !isValidESDFile(sourceFile, expectedSHA1) {
			fmt.Println("  - Downloading Windows ESD image")
			if err := downloadFile(downloadURL, sourceFile); err != nil {
				ErrorNoExit("Failed to download ESD image: " + err.Error())
				return
			}

			fmt.Println("  - Verifying download...")
			if !verifyFileSHA1(sourceFile, expectedSHA1) {
				os.Remove(sourceFile)
				ErrorNoExit("Successfully downloaded ESD image but it appears to be corrupted. Please run bvm again.")
				return
			}
			fmt.Println("Done")
		} else {
			fmt.Println("  - Not downloading " + sourceFile + " - file exists")
		}

		// Find Windows 11 <edition> partition number
		fmt.Println("  - Scanning ESD image for partitions...")
		professionalPartitionNum, err := getWindowsEditionPartition(sourceFile, edition)
		if err != nil {
			ErrorNoExit("Could not find Windows " + edition + " in image.esd: " + err.Error())
			return
		}

		// Extract Windows Setup Media
		Status("Extracting Windows Setup Media to esdextract")
		if err := runCommandWithSpinner("Extracting Windows Setup Media", "wimapply", sourceFile, "1", esdExtractDir); err != nil {
			ErrorNoExit("Operation failed: " + err.Error())
			return
		}

		// Extract Microsoft Windows PE to boot.wim
		Status("Extracting Microsoft Windows PE to boot.wim")
		bootWimPath := filepath.Join(esdExtractDir, "sources", "boot.wim")
		if err := runCommandWithSpinner("Extracting Windows PE", "wimexport", sourceFile, "2", bootWimPath, "--compress=LZX", "--chunk-size=32K"); err != nil {
			ErrorNoExit("Operation failed: " + err.Error())
			return
		}

		// Extract Microsoft Windows Setup to boot.wim
		Status("Extracting Microsoft Windows Setup to boot.wim")
		if err := runCommandWithSpinner("Extracting Windows Setup", "wimexport", sourceFile, "3", bootWimPath, "--compress=LZX", "--chunk-size=32K", "--boot"); err != nil {
			ErrorNoExit("Operation failed: " + err.Error())
			return
		}

		// Extract Windows 11 Pro to install.wim
		Status("Extracting Windows 11 Pro to install.wim")
		installWimPath := filepath.Join(esdExtractDir, "sources", "install.wim")
		if err := runCommandWithSpinner("Extracting Windows 11 Pro", "wimexport", sourceFile, professionalPartitionNum, installWimPath, "--compress=none"); err != nil {
			ErrorNoExit("Operation failed: " + err.Error())
			return
		}

		// Make boot noninteractive
		efisysPath := filepath.Join(esdExtractDir, "efi", "microsoft", "boot", "efisys.bin")
		efisysNopromptPath := filepath.Join(esdExtractDir, "efi", "microsoft", "boot", "efisys_noprompt.bin")
		if err := copyFile(efisysNopromptPath, efisysPath); err != nil {
			ErrorNoExit("Failed to copy efisys_noprompt.bin: " + err.Error())
			return
		}

		// Create installer.iso
		installerISOPath := filepath.Join("..", "installer.iso")
		os.Remove(installerISOPath) // Remove if exists

		Status("Making installer.iso disk image...")
		// Initial cleanup
		Status("Removing unnecessary .esd file before continuing...")
		os.Remove(sourceFile)
		args := []string{"-o", installerISOPath, "-R", "-iso-level", "3", "-udf",
			"-b", "efi/microsoft/boot/efisys.bin", "-no-emul-boot", "-V", "ESD_ISO",
			"-allow-limited-size", "."}

		if err := runGenisoWithProgress(args, esdExtractDir); err != nil {
			ErrorNoExit("Operation failed: " + err.Error())
			fmt.Println("DEBUG: genisoimage " + strings.Join(args, " "))
			return
		}

		// Cleanup
		os.RemoveAll(esdExtractDir)

		StatusGreen("Windows 11 ARM64 22631 ISO created successfully")

		// Download virtio drivers
		if err := DownloadVirtioDrivers(vmdir, arch); err != nil {
			ErrorNoExit("Failed to download virtio drivers: " + err.Error())
			return
		}

		return
	} else if arch == "ARM64" && (release == "11" || release == "Windows 11") {
		// for the latest Windows 11 ARM64 build, we need to get the download link from the Microsoft website by scraping the website for the download link
		if err := downloadWindowsFromMicrosoft(release, arch, language, vmdir); err != nil {
			ErrorNoExit(err.Error())
			return
		}
		StatusGreen("Windows 11 ARM64 ISO downloaded successfully")

		// Download virtio drivers
		if err := DownloadVirtioDrivers(vmdir, arch); err != nil {
			ErrorNoExit("Failed to download virtio drivers: " + err.Error())
			return
		}

		return
	} else if arch == "ARM64" && (release == "10" || release == "Windows 10") {
		// for the latest Windows 10 ARM64 build, we need to get the download link from the Microsoft website by scraping the website for the download link
		if err := downloadWindowsFromMicrosoft(release, arch, language, vmdir); err != nil {
			ErrorNoExit(err.Error())
			return
		}
		StatusGreen("Windows 10 ARM64 ISO downloaded successfully")

		// Download virtio drivers
		if err := DownloadVirtioDrivers(vmdir, arch); err != nil {
			ErrorNoExit("Failed to download virtio drivers: " + err.Error())
			return
		}
		installerISOPath := filepath.Join(vmdir, "installer.iso")
		// Patch the ISO to make it noninteractive
		if err := PatchWindowsISO(installerISOPath); err != nil {
			ErrorNoExit("Failed to patch ISO: " + err.Error())
			return
		}
		return
	} else if arch == "x64" && (release == "11" || release == "Windows 11") {
		// for the latest Windows 11 x64 build, we need to get the download link from the Microsoft website by scraping the website for the download link
		if err := downloadWindowsFromMicrosoft(release, arch, language, vmdir); err != nil {
			ErrorNoExit(err.Error())
			return
		}
		StatusGreen("Windows 11 x64 ISO downloaded successfully")

		// Download virtio drivers
		if err := DownloadVirtioDrivers(vmdir, arch); err != nil {
			ErrorNoExit("Failed to download virtio drivers: " + err.Error())
			return
		}
		installerISOPath := filepath.Join(vmdir, "installer.iso")
		// Patch the ISO to make it noninteractive
		if err := PatchWindowsISO(installerISOPath); err != nil {
			ErrorNoExit("Failed to patch ISO: " + err.Error())
			return
		}
		return
	} else if arch == "x64" && (release == "10" || release == "Windows 10") {
		// for the latest Windows 10 x64 build, we need to get the download link from the Microsoft website by scraping the website for the download link
		if err := downloadWindowsFromMicrosoft(release, arch, language, vmdir); err != nil {
			ErrorNoExit(err.Error())
			return
		}
		StatusGreen("Windows 10 x64 ISO downloaded successfully")

		// Download virtio drivers
		if err := DownloadVirtioDrivers(vmdir, arch); err != nil {
			ErrorNoExit("Failed to download virtio drivers: " + err.Error())
			return
		}
		return
	} else if arch == "ARMv7" && version == "15035" {
		// for the Windows 10 ARMv7 leaked build, download a cached downloaded image from the open-rt.party file server
		// Only English (United States) is supported for this build, will ignore any language other than specified
		if language != "English (United States)" {
			ErrorNoExit("Windows 10 ARMv7 build 15035 is only supported for English (United States)")
			return
		}

		// Create esdextract directory (same as build 22631 process)
		esdExtractDir := filepath.Join(vmdir, "esdextract")
		if err := os.RemoveAll(esdExtractDir); err != nil {
			ErrorNoExit("Failed to remove esdextract folder: " + err.Error())
			return
		}
		if err := os.MkdirAll(esdExtractDir, 0755); err != nil {
			ErrorNoExit("Directory creation failed: " + err.Error())
			return
		}

		sourceFile := filepath.Join(vmdir, "image.7z")

		// Download 7z archive if not already present
		if _, err := os.Stat(sourceFile); os.IsNotExist(err) {
			fmt.Println("  - Downloading Windows 10 ARMv7 build 15035 archive")
			if err := downloadFile(URL, sourceFile); err != nil {
				ErrorNoExit("Failed to download Windows 10 ARMv7 archive: " + err.Error())
				return
			}
		} else {
			fmt.Println("  - Not downloading " + sourceFile + " - file exists")
		}

		// Extract 7z archive using 7z command
		Status("Extracting Windows 10 ARMv7 build 15035 archive")
		if err := runCommandWithSpinner("Extracting archive", "7z", "x", sourceFile, "-o"+esdExtractDir); err != nil {
			ErrorNoExit("Failed to extract 7z archive: " + err.Error())
			return
		}

		// Make boot noninteractive (same as build 22631 process)
		efisysPath := filepath.Join(esdExtractDir, "efi", "microsoft", "boot", "efisys.bin")
		efisysNopromptPath := filepath.Join(esdExtractDir, "efi", "microsoft", "boot", "efisys_noprompt.bin")
		if err := copyFile(efisysNopromptPath, efisysPath); err != nil {
			ErrorNoExit("Failed to copy efisys_noprompt.bin: " + err.Error())
			return
		}

		// Create installer.iso (same as build 22631 process)
		installerISOPath := filepath.Join("..", "installer.iso")
		os.Remove(installerISOPath) // Remove if exists

		Status("Making installer.iso disk image...")
		// Initial cleanup
		Status("Removing unnecessary .7z file before continuing...")
		os.Remove(sourceFile)
		args := []string{"-o", installerISOPath, "-R", "-iso-level", "3", "-udf",
			"-b", "efi/microsoft/boot/efisys.bin", "-no-emul-boot", "-V", "WIN10_ARMV7",
			"-allow-limited-size", "."}

		if err := runGenisoWithProgress(args, esdExtractDir); err != nil {
			ErrorNoExit("Operation failed: " + err.Error())
			fmt.Println("DEBUG: genisoimage " + strings.Join(args, " "))
			return
		}

		// Cleanup
		os.RemoveAll(esdExtractDir)

		StatusGreen("Windows 10 ARMv7 build 15035 ISO created successfully")

		// Download virtio drivers
		// note: virtio drivers will need to be different for ARMv7 than for ARM64, so we need to handle this separately, for now let it skip as we don't have a full virtio driver set for ARMv7 (only viostor)
		// uncomment this when we have a full virtio driver set for ARMv7
		// if err := DownloadVirtioDrivers(vmdir, arch); err != nil {
		// 	ErrorNoExit("Failed to download virtio drivers: " + err.Error())
		// 	return
		// }
		return
	} else {
		ErrorNoExit("Invalid architecture/build: " + arch + " " + version)
		return
	}
}

// getLanguageCode converts pretty language name to short-code used by ESD releases
func getLanguageCode(language string) string {
	languageMap := map[string]string{
		"Arabic":                  "ar-sa",
		"Brazilian Portuguese":    "pt-br",
		"Bulgarian":               "bg-bg",
		"Chinese (Simplified)":    "zh-cn",
		"Chinese (Traditional)":   "zh-tw",
		"Croatian":                "hr-hr",
		"Czech":                   "cs-cz",
		"Danish":                  "da-dk",
		"Dutch":                   "nl-nl",
		"English (United States)": "en-us",
		"English International":   "en-gb",
		"Estonian":                "et-ee",
		"Finnish":                 "fi-fi",
		"French":                  "fr-fr",
		"French Canadian":         "fr-ca",
		"German":                  "de-de",
		"Greek":                   "el-gr",
		"Hebrew":                  "he-il",
		"Hungarian":               "hu-hu",
		"Italian":                 "it-it",
		"Japanese":                "ja-jp",
		"Korean":                  "ko-kr",
		"Latvian":                 "lv-lv",
		"Lithuanian":              "lt-lt",
		"Norwegian":               "nb-no",
		"Polish":                  "pl-pl",
		"Portuguese":              "pt-pt",
		"Romanian":                "ro-ro",
		"Russian":                 "ru-ru",
		"Serbian Latin":           "sr-latn-rs",
		"Slovak":                  "sk-sk",
		"Slovenian":               "sl-si",
		"Spanish":                 "es-es",
		"Spanish (Mexico)":        "es-mx",
		"Swedish":                 "sv-se",
		"Thai":                    "th-th",
		"Turkish":                 "tr-tr",
		"Ukrainian":               "uk-ua",
	}
	return languageMap[language]
}

// parseCatalogForLanguage extracts the language-specific section from the ESD catalog
func parseCatalogForLanguage(catalog, langCode string) string {
	// Split catalog by > and < to get individual elements
	catalog = strings.ReplaceAll(catalog, "><", ">\n<")

	// Find the section for our language
	lines := strings.Split(catalog, "\n")
	var result []string
	inLanguageSection := false
	foundLanguage := false

	for _, line := range lines {
		if strings.Contains(line, "<LanguageCode>"+langCode) {
			inLanguageSection = true
			foundLanguage = true
		}
		if inLanguageSection && strings.Contains(line, "</File>") {
			result = append(result, line)
			break
		}
		if inLanguageSection || strings.Contains(line, "<Languages>") {
			if strings.Contains(line, "<Languages>") {
				break // Stop before Languages section
			}
			result = append(result, line)
		}
	}

	if !foundLanguage {
		return ""
	}
	return strings.Join(result, "\n")
}

// extractXMLValue extracts a value from XML content
func extractXMLValue(content, tag string) string {
	pattern := "<" + tag + ">(.*?)</" + tag + ">"
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(content)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// isValidESDFile checks if the ESD file exists and has the correct SHA1
func isValidESDFile(filename, expectedSHA1 string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}
	return verifyFileSHA1(filename, expectedSHA1)
}

// verifyFileSHA1 checks if a file has the expected SHA1 hash
func verifyFileSHA1(filename, expectedSHA1 string) bool {
	file, err := os.Open(filename)
	if err != nil {
		return false
	}
	defer file.Close()

	hash := sha1.New()
	if _, err := io.Copy(hash, file); err != nil {
		return false
	}

	actualSHA1 := hex.EncodeToString(hash.Sum(nil))
	return strings.EqualFold(actualSHA1, expectedSHA1)
}

// downloadFile downloads a file from URL to local path with live progress feedback
func downloadFile(url, filepath string) error {
	// Create progress bar model
	p := progress.New(progress.WithDefaultGradient())
	filename := filepath[strings.LastIndex(filepath, "/")+1:]

	m := downloadModel{
		progress: p,
		filename: filename,
	}

	// Start the Bubble Tea program in a goroutine
	var program *tea.Program
	var finalErr error

	go func() {
		// Start HTTP request
		resp, err := http.Get(url)
		if err != nil {
			program.Send(downloadCompleteMsg{err: err})
			return
		}
		defer resp.Body.Close()

		// Get total size
		total := resp.ContentLength
		program.Send(progressMsg{current: 0, total: total})

		// Create output file
		out, err := os.Create(filepath)
		if err != nil {
			program.Send(downloadCompleteMsg{err: err})
			return
		}
		defer out.Close()

		// Create a reader that reports progress
		var current int64
		buf := make([]byte, 32*1024) // 32KB buffer

		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				if _, writeErr := out.Write(buf[:n]); writeErr != nil {
					program.Send(downloadCompleteMsg{err: writeErr})
					return
				}
				current += int64(n)
				program.Send(progressMsg{current: current, total: total})
			}

			if err == io.EOF {
				break
			}
			if err != nil {
				program.Send(downloadCompleteMsg{err: err})
				return
			}
		}

		program.Send(downloadCompleteMsg{err: nil})
	}()

	// Run the TUI
	program = tea.NewProgram(m)
	finalModel, err := program.Run()
	if err != nil {
		return err
	}

	// Extract any error from the final model
	if dm, ok := finalModel.(downloadModel); ok {
		finalErr = dm.err
	}

	return finalErr
}

// getWindowsEditionPartition finds the partition number for a specific Windows edition with fallbacks
func getWindowsEditionPartition(esdFile string, edition string) (string, error) {
	cmd := exec.Command("wiminfo", esdFile)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Define fallback editions for each main edition type
	fallbacks := map[string][]string{
		"Pro":          {"Pro N", "Pro Education", "Pro for Workstations"},
		"Pro N":        {"Pro", "Pro Education N", "Pro N for Workstations"},
		"Home":         {"Home N"},
		"Home N":       {"Home"},
		"Education":    {"Education N", "Pro Education"},
		"Education N":  {"Education", "Pro Education N"},
		"Enterprise":   {"Enterprise N"},
		"Enterprise N": {"Enterprise"},
	}

	// Try the requested edition first
	if partition, err := findEditionPartition(string(output), edition); err == nil {
		return partition, nil
	}

	// Try fallback editions if available
	if fallbackList, ok := fallbacks[edition]; ok {
		for _, fallback := range fallbackList {
			if partition, err := findEditionPartition(string(output), fallback); err == nil {
				Warning("Failed to locate Windows " + edition + " partition, falling back to " + fallbackList[0])
				return partition, nil
			}
		}
	}

	return "", fmt.Errorf("windows %s partition not found (including fallbacks)", edition)
}

// findEditionPartition searches for a specific edition in the wiminfo output
func findEditionPartition(output string, edition string) (string, error) {
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if strings.Contains(line, "Windows") && strings.Contains(line, edition) {
			// Look for the index number in the previous line
			if i > 0 {
				prevLine := strings.TrimSpace(lines[i-1])
				if strings.HasPrefix(prevLine, "Index:") {
					parts := strings.Fields(prevLine)
					if len(parts) >= 2 {
						return parts[1], nil
					}
				}
			}
		}
	}
	return "", fmt.Errorf("windows %s partition not found", edition)
}

// runCommandWithSpinner runs a command with a spinner showing the operation
func runCommandWithSpinner(operation string, name string, args ...string) error {
	s := spinner.New()
	s.Spinner = spinner.Dot

	m := operationModel{
		spinner:   s,
		operation: operation,
	}

	var program *tea.Program
	var finalErr error

	go func() {
		cmd := exec.Command(name, args...)
		err := cmd.Run()
		program.Send(operationCompleteMsg{err: err})
	}()

	program = tea.NewProgram(m)
	finalModel, err := program.Run()
	if err != nil {
		return err
	}

	if om, ok := finalModel.(operationModel); ok {
		finalErr = om.err
	}

	return finalErr
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// runGenisoWithProgress runs genisoimage with live progress feedback
func runGenisoWithProgress(args []string, workingDir string) error {
	// Create progress bar model for ISO creation
	p := progress.New(progress.WithDefaultGradient())

	m := isoProgressModel{
		progress:  p,
		operation: "Creating installer.iso",
		percent:   0.0,
		status:    "",
	}

	var program *tea.Program
	var finalErr error
	var cmdOutput strings.Builder

	go func() {
		// Create the command directly without script wrapper
		cmd := exec.Command("genisoimage", args...)
		cmd.Dir = workingDir

		// Capture both stdout and stderr
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			program.Send(isoCompleteMsg{err: err})
			return
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			program.Send(isoCompleteMsg{err: err})
			return
		}

		if err := cmd.Start(); err != nil {
			program.Send(isoCompleteMsg{err: err})
			return
		}

		// Read stdout
		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				line := scanner.Text()
				cmdOutput.WriteString(line + "\n")

				// Parse progress from stdout (genisoimage outputs progress here)
				if strings.Contains(line, "% done") {
					if percent := parseGenisoProgress(line); percent >= 0 {
						status := ""
						if strings.Contains(line, "estimate finish") {
							// Extract the estimate part
							parts := strings.Split(line, ",")
							if len(parts) > 1 {
								status = strings.TrimSpace(parts[1])
							}
						}
						program.Send(isoProgressMsg{percent: percent, status: status})
					}
				}
			}
		}()

		// Read stderr
		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				line := scanner.Text()
				cmdOutput.WriteString(line + "\n")

				// Also check stderr for progress (some versions output here)
				if strings.Contains(line, "% done") {
					if percent := parseGenisoProgress(line); percent >= 0 {
						status := ""
						if strings.Contains(line, "estimate finish") {
							parts := strings.Split(line, ",")
							if len(parts) > 1 {
								status = strings.TrimSpace(parts[1])
							}
						}
						program.Send(isoProgressMsg{percent: percent, status: status})
					}
				}
			}
		}()

		err = cmd.Wait()
		if err != nil {
			// If command failed, include the output in the error
			output := cmdOutput.String()
			if exitErr, ok := err.(*exec.ExitError); ok {
				err = fmt.Errorf("genisoimage failed:\nCommand: genisoimage %s\nOutput:\n%s\nError: %s",
					strings.Join(args, " "),
					output,
					string(exitErr.Stderr))
			} else {
				err = fmt.Errorf("genisoimage failed:\nCommand: genisoimage %s\nOutput:\n%s\nError: %v",
					strings.Join(args, " "),
					output,
					err)
			}
		}
		program.Send(isoCompleteMsg{err: err})
	}()

	program = tea.NewProgram(m)
	finalModel, err := program.Run()
	if err != nil {
		return err
	}

	if om, ok := finalModel.(isoProgressModel); ok {
		finalErr = om.err
	}

	return finalErr
}

// parseGenisoProgress extracts percentage from genisoimage progress line
// Example: "99.63% done, estimate finish Thu Jun 12 18:53:31 2025"
func parseGenisoProgress(line string) float64 {
	// Use regex to extract percentage
	re := regexp.MustCompile(`(\d+\.?\d*)%`)
	matches := re.FindStringSubmatch(line)
	if len(matches) >= 2 {
		if percent, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return percent
		}
	}
	return -1 // Return -1 if parsing failed
}

// downloadWindowsFromMicrosoft downloads Windows ISO from Microsoft's official API
func downloadWindowsFromMicrosoft(release, arch, language, vmdir string) error {
	// Determine the URL based on release and architecture
	var url string
	var archFilter string

	switch {
	case (release == "11" || release == "Windows 11") && arch == "ARM64":
		url = "https://www.microsoft.com/en-us/software-download/windows11arm64"
		archFilter = "Arm64"
		// If the release contains the word Windows, strip it off
		release = strings.TrimPrefix(release, "Windows ")
	case (release == "11" || release == "Windows 11") && arch == "x64":
		url = "https://www.microsoft.com/en-us/software-download/windows11"
		archFilter = "x64"
		// If the release contains the word Windows, strip it off
		release = strings.TrimPrefix(release, "Windows ")
	case (release == "10" || release == "Windows 10") && arch == "ARM64":
		url = "https://www.microsoft.com/en-us/software-download/windows10arm64"
		archFilter = "Arm64"
		// If the release contains the word Windows, strip it off
		release = strings.TrimPrefix(release, "Windows ")
	case (release == "10" || release == "Windows 10") && arch == "x64":
		url = "https://www.microsoft.com/en-us/software-download/windows10"
		archFilter = "x64"
		// If the release contains the word Windows, strip it off
		release = strings.TrimPrefix(release, "Windows ")
	default:
		return fmt.Errorf("unsupported combination: Windows %s %s", release, arch)
	}
	// Thanks to a unnoticed adblocker causing a failure during development of this tool, we need to provide a more helpful error message if the user were to be using a network level blocker
	failedInstructions := fmt.Sprintf(`The download failed (possibly due to blocking critical requests by an network level blocker). 
In your adblocker settings, please whitelist the following domains:

	https://www.microsoft.com
	https://vlscppe.microsoft.com
	https://ov-df.microsoft.com

Otherwise if the problem isn't related due to failed requests, using a web browser, please manually download the Windows %s %s ISO from: %s
Save the downloaded ISO to: %s/installer.iso
Make sure the file is named installer.iso
Then run this action again.`, release, arch, url, vmdir)

	Status(fmt.Sprintf("Downloading Windows %s %s (%s)", release, arch, language))

	userAgent := "Mozilla/5.0 (X11; Linux x86_64; rv:130.0) Gecko/20100101 Firefox/130.0"
	sessionID := uuid.New().String()

	// Create HTTP client with timeout and cookie jar
	client := &http.Client{
		Timeout: 60 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Follow up to 10 redirects
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil
		},
	}

	// Helper function to create requests with common headers
	createRequest := func(method, url string) (*http.Request, error) {
		req, err := http.NewRequest(method, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
		req.Header.Set("Accept-Language", "en-US,en;q=0.5")
		// Don't request compressed content to avoid gzip issues
		// req.Header.Set("Accept-Encoding", "gzip, deflate, br")
		req.Header.Set("DNT", "1")
		req.Header.Set("Connection", "keep-alive")
		req.Header.Set("Upgrade-Insecure-Requests", "1")
		req.Header.Set("Sec-Fetch-Dest", "document")
		req.Header.Set("Sec-Fetch-Mode", "navigate")
		req.Header.Set("Sec-Fetch-Site", "none")
		req.Header.Set("Cache-Control", "max-age=0")
		return req, nil
	}

	// Helper function to create API requests with different headers
	createAPIRequest := func(method, url string) (*http.Request, error) {
		req, err := http.NewRequest(method, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("Accept", "application/json, text/plain, */*")
		req.Header.Set("Accept-Language", "en-US,en;q=0.5")
		req.Header.Set("DNT", "1")
		req.Header.Set("Connection", "keep-alive")
		req.Header.Set("Sec-Fetch-Dest", "empty")
		req.Header.Set("Sec-Fetch-Mode", "cors")
		req.Header.Set("Sec-Fetch-Site", "same-origin")
		return req, nil
	}

	// Step 1: Get product edition ID from download page
	fmt.Println("  - Parsing download page:", url)
	req, err := createRequest("GET", url)
	if err != nil {
		return fmt.Errorf("failed to create request: %v\n%s", err, failedInstructions)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to scrape the webpage on step 1: %v\n%s", err, failedInstructions)
	}
	defer resp.Body.Close()

	pageHTML, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read webpage: %v\n%s", err, failedInstructions)
	}

	htmlStr := string(pageHTML)

	// Extract product edition ID with multiple patterns
	fmt.Print("  - Getting Product edition ID: ")
	var productEditionID string

	// Try multiple regex patterns
	patterns := []string{
		`<option value="([0-9]+)">Windows`,                                               // Original pattern
		`<option[^>]+value="([0-9]+)"[^>]*>Windows`,                                      // With additional attributes
		`<option[^>]+value='([0-9]+)'[^>]*>Windows`,                                      // Single quotes
		`value="([0-9]+)"[^>]*>[^<]*Windows`,                                             // More flexible
		`<option[^>]*value="([0-9]+)"[^>]*>[^<]*[Ww]indows`,                              // Case insensitive
		`"([0-9]+)"[^>]*>[^<]*Windows`,                                                   // Just quotes and Windows
		`value\s*=\s*["']([0-9]+)["'][^>]*>[^<]*Windows`,                                 // With whitespace
		`<select[^>]*name="[^"]*edition[^"]*"[^>]*>[\s\S]*?<option[^>]*value="([0-9]+)"`, // Look for edition select
		`ProductEditionId["\s]*[:=]["\s]*([0-9]+)`,                                       // JavaScript variable
		`productEditionId["\s]*[:=]["\s]*([0-9]+)`,                                       // Camel case version
	}

	for i, pattern := range patterns {
		re := regexp.MustCompile(`(?i)` + pattern) // Case insensitive
		matches := re.FindStringSubmatch(htmlStr)
		if len(matches) >= 2 {
			productEditionID = matches[1]
			fmt.Printf("(pattern %d) %s\n", i+1, productEditionID)
			break
		}
	}

	// If still no match, try hardcoded values based on Windows version as fallback
	if productEditionID == "" {
		fmt.Printf("patterns failed, trying fallback... ")
		switch {
		case (release == "10" || release == "Windows 10") && arch == "x64":
			productEditionID = "2618" // Known value for Windows 10 x64
			fmt.Printf("(fallback) %s\n", productEditionID)
		case (release == "10" || release == "Windows 10") && arch == "ARM64":
			productEditionID = "2618" // Same for ARM64
			fmt.Printf("(fallback) %s\n", productEditionID)
		case (release == "11" || release == "Windows 11") && arch == "x64":
			productEditionID = "2935" // Common value for Windows 11 x64
			fmt.Printf("(fallback) %s\n", productEditionID)
		case (release == "11" || release == "Windows 11") && arch == "ARM64":
			productEditionID = "2935" // Same for ARM64
			fmt.Printf("(fallback) %s\n", productEditionID)
		}
	}

	if productEditionID == "" {
		return fmt.Errorf("failed to find product edition ID\n%s", failedInstructions)
	}

	// Add delay to appear more human-like
	time.Sleep(2 * time.Second)

	// Step 2: Permit Session ID
	fmt.Println("  - Permit Session ID:", sessionID)
	permitURL := fmt.Sprintf("https://vlscppe.microsoft.com/tags?org_id=y6jn8c31&session_id=%s", sessionID)
	req, err = createRequest("GET", permitURL)
	if err != nil {
		fmt.Println("  - Warning: Failed to create permit request, continuing anyway:", err)
	} else {
		resp, err = client.Do(req)
		if err != nil {
			// it this fails, step 3 will fail too because of Sentinel, so return a error
			return fmt.Errorf("failed to permit session ID (step 2): %v\n%s", err, failedInstructions)
		} else {
			resp.Body.Close()
			fmt.Println("  - Session ID permitted successfully")
		}
	}

	// Add delay to appear more human-like
	time.Sleep(3 * time.Second)

	// Step 3: Get language SKU ID table
	fmt.Print("  - Getting language SKU ID: ")
	profile := "606624d44113"
	skuURL := fmt.Sprintf("https://www.microsoft.com/software-download-connector/api/getskuinformationbyproductedition?profile=%s&ProductEditionId=%s&SKU=undefined&friendlyFileName=undefined&Locale=en-US&sessionID=%s",
		profile, productEditionID, sessionID)

	req, err = createAPIRequest("GET", skuURL)
	if err != nil {
		return fmt.Errorf("failed to create SKU request: %v\n%s", err, failedInstructions)
	}
	// Add JSON accept header for API calls
	req.Header.Set("Accept", "application/json, text/plain, */*")

	resp, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to scrape the webpage on step 3: %v\n%s", err, failedInstructions)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("received HTTP %d from Microsoft API at step 3\n%s", resp.StatusCode, failedInstructions)
	}

	skuBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read SKU response: %v\n%s", err, failedInstructions)
	}

	var skuResponse SKUResponse
	if err := json.Unmarshal(skuBody, &skuResponse); err != nil {
		return fmt.Errorf("failed to parse SKU JSON: %v\nResponse: %s\n%s", err, string(skuBody), failedInstructions)
	}

	// Find SKU ID for the specified language
	var skuID string

	// Try multiple matching strategies
	for _, sku := range skuResponse.Skus {
		// First try exact match on LocalizedLanguage
		if sku.LocalizedLanguage == language {
			skuID = sku.Id
			break
		}
		// Then try exact match on Language
		if sku.Language == language {
			skuID = sku.Id
			break
		}
		// Try partial match for English variants
		if language == "English (United States)" && (strings.Contains(strings.ToLower(sku.LocalizedLanguage), "english") ||
			strings.Contains(strings.ToLower(sku.Language), "english") ||
			sku.Language == "en-US" ||
			sku.Language == "en-us") {
			skuID = sku.Id
			break
		}
	}

	fmt.Println(skuID)

	if skuID == "" {
		// Show more helpful error with available languages
		var availableLanguages []string
		for _, sku := range skuResponse.Skus {
			availableLanguages = append(availableLanguages, fmt.Sprintf("'%s'", sku.LocalizedLanguage))
		}
		return fmt.Errorf("failed to get the sku_id for language '%s'\nAvailable languages: %s\n%s",
			language, strings.Join(availableLanguages, ", "), failedInstructions)
	}

	// Add delay to appear more human-like
	time.Sleep(5 * time.Second)

	// Step 4: Get ISO download link
	// Note: If any request is going to be blocked by Microsoft it's always this last one
	// (the previous requests always seem to succeed) - hence the referer is critical
	fmt.Println("  - Getting ISO download link...")
	downloadURL := fmt.Sprintf("https://www.microsoft.com/software-download-connector/api/GetProductDownloadLinksBySku?profile=%s&productEditionId=undefined&SKU=%s&friendlyFileName=undefined&Locale=en-US&sessionID=%s",
		profile, skuID, sessionID)

	req, err = createAPIRequest("GET", downloadURL)
	if err != nil {
		return fmt.Errorf("failed to create download link request: %v\n%s", err, failedInstructions)
	}
	req.Header.Set("Referer", url)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	// Add timestamp to make request unique
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")

	resp, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get download links: %v\n%s", err, failedInstructions)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("received HTTP %d from Microsoft API at step 4\n%s", resp.StatusCode, failedInstructions)
	}

	downloadBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read download response: %v\n%s", err, failedInstructions)
	}

	downloadBodyStr := string(downloadBody)

	if downloadBodyStr == "" {
		return fmt.Errorf("microsoft servers gave an empty response to the request for an automated download\n%s", failedInstructions)
	}

	if strings.Contains(downloadBodyStr, "Sentinel marked this request as rejected") {
		return fmt.Errorf("microsoft blocked the automated download request based on your IP address. Follow the instructions below, or wait an hour and try again to see if your IP has been unblocked\n%s", failedInstructions)
	}

	var downloadResponse DownloadResponse
	if err := json.Unmarshal(downloadBody, &downloadResponse); err != nil {
		return fmt.Errorf("failed to parse download JSON: %v\n%s", err, failedInstructions)
	}

	// Filter for the correct architecture ISO download URL
	var isoDownloadLink string
	for _, option := range downloadResponse.ProductDownloadOptions {
		if strings.Contains(option.Uri, archFilter) {
			isoDownloadLink = option.Uri
			break
		}
	}

	if isoDownloadLink == "" {
		return fmt.Errorf("microsoft servers gave no download link for %s architecture\n%s", archFilter, failedInstructions)
	}

	// Extract clean URL (remove query parameters for display)
	cleanURL := isoDownloadLink
	if idx := strings.Index(cleanURL, "?"); idx != -1 {
		cleanURL = cleanURL[:idx]
	}
	fmt.Println("  - URL:", cleanURL)

	// Download ISO with progress bar
	// Ensure the vmdir directory exists
	if err := os.MkdirAll(vmdir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %v\n%s", vmdir, err, failedInstructions)
	}

	installerPath := filepath.Join(vmdir, "installer.iso")
	if err := downloadFile(isoDownloadLink, installerPath); err != nil {
		return fmt.Errorf("failed to download Windows %s installer.iso from Microsoft: %v\n%s", release, err, failedInstructions)
	}

	// Verify SHA256
	fmt.Print("  - Verifying download... ")
	if err := verifyWindowsISO(installerPath, string(pageHTML)); err != nil {
		os.Remove(installerPath)
		return fmt.Errorf("verification failed: %v", err)
	}
	fmt.Println("Done")
	fmt.Println("  - Verification successful.")

	return nil
}

// verifyWindowsISO verifies the downloaded ISO against SHA256 hashes from the download page
func verifyWindowsISO(isoPath, pageHTML string) error {
	file, err := os.Open(isoPath)
	if err != nil {
		return fmt.Errorf("failed to open ISO file: %v", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("failed to calculate SHA256: %v", err)
	}

	actualSHA256 := hex.EncodeToString(hash.Sum(nil))

	// Check if the calculated hash appears anywhere in the download page
	// The page contains SHA256 hashes for all languages
	if strings.Contains(strings.ToLower(pageHTML), strings.ToLower(actualSHA256)) {
		return nil
	}

	// Try multiple patterns to extract hashes from the page
	hashPatterns := []string{
		`<td>[a-fA-F0-9]{64}</td>`,                  // Original pattern (table cells)
		`>[a-fA-F0-9]{64}<`,                         // Between any tags
		`[^a-fA-F0-9]([a-fA-F0-9]{64})[^a-fA-F0-9]`, // Surrounded by non-hex chars
		`hash["\s]*[:=]["\s]*([a-fA-F0-9]{64})`,     // JavaScript hash variables
		`sha256["\s]*[:=]["\s]*([a-fA-F0-9]{64})`,   // SHA256 variables
		`checksum["\s]*[:=]["\s]*([a-fA-F0-9]{64})`, // Checksum variables
	}

	var hashes []string
	for _, pattern := range hashPatterns {
		re := regexp.MustCompile(`(?i)` + pattern)
		matches := re.FindAllStringSubmatch(pageHTML, -1)
		for _, match := range matches {
			var hashValue string
			if len(match) > 1 {
				hashValue = match[1] // Captured group
			} else {
				hashValue = strings.Trim(match[0], "<>td/")
			}

			// Validate it's a proper SHA256 hash (64 hex chars)
			if len(hashValue) == 64 {
				if matched, _ := regexp.Compile(`^[a-fA-F0-9]{64}$`); matched.MatchString(hashValue) {
					// Avoid duplicates
					found := false
					for _, existing := range hashes {
						if strings.EqualFold(existing, hashValue) {
							found = true
							break
						}
					}
					if !found {
						hashes = append(hashes, hashValue)
					}
				}
			}
		}
	}

	// If we found hashes and our hash doesn't match any of them, that's an error
	if len(hashes) > 0 {
		for _, pageHash := range hashes {
			if strings.EqualFold(actualSHA256, pageHash) {
				return nil // Match found
			}
		}
		return fmt.Errorf("installer.iso seems corrupted after download. Its sha256sum was:\n%s\nwhich does not match any on this list:\n%s",
			actualSHA256, strings.Join(hashes, "\n"))
	}

	// If no hashes found on page, just show a warning but don't fail
	// Since the download came from Microsoft's official servers, it's likely legitimate
	Warning("Could not find SHA256 hashes on download page for verification.")
	Warning("Downloaded ISO SHA256: " + actualSHA256)
	Warning("Since the download came directly from Microsoft's servers, the file is likely legitimate.")
	return nil
}

// Force the specified windows ISO image not require keypress to boot into installer
func PatchWindowsISO(isoPath string) error {
	Status("Patching Windows ISO to skip boot prompt...")

	// First try the efisys_noprompt.bin replacement method
	if err := patchISOWithEfisysNoprompt(isoPath); err == nil {
		StatusGreen("Successfully patched ISO using efisys_noprompt.bin method")
		return nil
	} else {
		Debug("efisys_noprompt.bin method failed: " + err.Error())
		Status("Falling back to direct binary patching method...")
	}

	// Fall back to direct binary patching
	if err := patchISODirectBinary(isoPath); err != nil {
		return fmt.Errorf("both patching methods failed: %v", err)
	}

	StatusGreen("Successfully patched ISO using direct binary patching method")
	return nil
}

// patchISOWithEfisysNoprompt tries to patch the ISO by replacing efisys.bin with efisys_noprompt.bin
func patchISOWithEfisysNoprompt(isoPath string) error {
	// Create temporary directory for ISO extraction
	tempDir, err := os.MkdirTemp("", "iso-patch-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Extract ISO contents
	Status("Extracting ISO contents...")
	err = runCommandWithSpinner("Extracting ISO", "7z", "x", isoPath, "-o"+tempDir, "-y")
	if err != nil {
		return fmt.Errorf("failed to extract ISO: %w", err)
	}

	// Look for efisys_noprompt.bin
	efisysNopromptPath := filepath.Join(tempDir, "efi", "microsoft", "boot", "efisys_noprompt.bin")
	if _, err := os.Stat(efisysNopromptPath); os.IsNotExist(err) {
		return fmt.Errorf("efisys_noprompt.bin not found in ISO")
	}

	// Look for efisys.bin
	efisysPath := filepath.Join(tempDir, "efi", "microsoft", "boot", "efisys.bin")
	if _, err := os.Stat(efisysPath); os.IsNotExist(err) {
		return fmt.Errorf("efisys.bin not found in ISO")
	}

	// Replace efisys.bin with efisys_noprompt.bin
	Status("Replacing efisys.bin with efisys_noprompt.bin...")
	err = copyFile(efisysNopromptPath, efisysPath)
	if err != nil {
		return fmt.Errorf("failed to replace efisys.bin: %w", err)
	}

	// Rebuild the ISO
	Status("Rebuilding ISO...")
	err = rebuildISO(tempDir, isoPath)
	if err != nil {
		return fmt.Errorf("failed to rebuild ISO: %w", err)
	}

	return nil
}

// patchISODirectBinary patches the ISO by directly modifying the binary content
func patchISODirectBinary(isoPath string) error {
	// Constants from the bash version
	const matchOffset = 934748 // where 'cdboot.pdb' string is in efisys.bin
	const fileLength = 1720320 // total length of efisys.bin/efisys_noprompt.bin

	Status("Searching for efisys_noprompt.bin in ISO...")

	// First find and extract efisys_noprompt.bin from the ISO
	nopromptData, err := findAndExtractEfisysNoprompt(isoPath, matchOffset, fileLength)
	if err != nil {
		return fmt.Errorf("failed to extract efisys_noprompt.bin: %w", err)
	}

	Status("Searching for efisys.bin instances to replace...")

	// Find and replace all instances of efisys.bin with efisys_noprompt.bin
	err = findAndReplaceEfisysBin(isoPath, nopromptData, matchOffset, fileLength)
	if err != nil {
		return fmt.Errorf("failed to replace efisys.bin instances: %w", err)
	}

	return nil
}

// rebuildISO rebuilds an ISO from extracted contents
func rebuildISO(sourceDir, outputPath string) error {
	// Use genisoimage to rebuild the ISO with proper boot settings
	args := []string{
		"-o", outputPath,
		"-iso-level", "3",
		"-udf",
		"-b", "efi/microsoft/boot/efisys.bin",
		"-no-emul-boot",
		"-V", "ESD_ISO",
		"-allow-limited-size",
		sourceDir,
	}

	return runGenisoWithProgress(args, "")
}

// findAndExtractEfisysNoprompt finds efisys_noprompt.bin in the ISO and extracts it
func findAndExtractEfisysNoprompt(isoPath string, matchOffset int, fileLength int) ([]byte, error) {
	file, err := os.Open(isoPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Search for 'cdboot_noprompt.pdb' pattern
	searchPattern := []byte("cdboot_noprompt.pdb")
	buffer := make([]byte, 1024*1024) // 1MB buffer
	var filePos int64 = 0

	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if n == 0 {
			break
		}

		// Search for pattern in buffer
		for i := 0; i <= n-len(searchPattern); i++ {
			if bytes.Equal(buffer[i:i+len(searchPattern)], searchPattern) {
				// Found the pattern, calculate start of file
				patternPos := filePos + int64(i)
				startPos := patternPos - int64(matchOffset)

				// Verify this is the correct start of file by checking signature
				_, err := file.Seek(startPos, 0)
				if err != nil {
					continue
				}

				signature := make([]byte, 16)
				_, err = file.Read(signature)
				if err != nil {
					continue
				}

				// Check for expected efisys file signature
				expectedSig := []byte{0xEB, 0xEC, 0x90, 0x4D, 0x53, 0x44, 0x4F, 0x53, 0x35, 0x2E, 0x30, 0x00, 0x02, 0x02, 0x01, 0x00}
				if bytes.Equal(signature, expectedSig) {
					// Found it! Extract the file
					_, err = file.Seek(startPos, 0)
					if err != nil {
						return nil, err
					}

					data := make([]byte, fileLength)
					_, err = io.ReadFull(file, data)
					if err != nil {
						return nil, err
					}

					return data, nil
				}
			}
		}

		filePos += int64(n)
		// Seek back a bit to handle pattern spanning buffer boundaries
		if n == len(buffer) {
			file.Seek(filePos-int64(len(searchPattern)), 0)
			filePos -= int64(len(searchPattern))
		}
	}

	return nil, fmt.Errorf("efisys_noprompt.bin not found in ISO")
}

// findAndReplaceEfisysBin finds all instances of efisys.bin and replaces them
func findAndReplaceEfisysBin(isoPath string, nopromptData []byte, matchOffset int, fileLength int) error {
	file, err := os.OpenFile(isoPath, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Search for 'cdboot.pdb' pattern (efisys.bin instances)
	searchPattern := []byte("cdboot.pdb")
	buffer := make([]byte, 1024*1024) // 1MB buffer
	var filePos int64 = 0
	replacements := 0

	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}

		// Search for pattern in buffer
		for i := 0; i <= n-len(searchPattern); i++ {
			if bytes.Equal(buffer[i:i+len(searchPattern)], searchPattern) {
				// Found the pattern, calculate start of file
				patternPos := filePos + int64(i)
				startPos := patternPos - int64(matchOffset)

				// Verify this is the correct start of file by checking signature
				currentPos, _ := file.Seek(0, 1) // Get current position
				_, err := file.Seek(startPos, 0)
				if err != nil {
					file.Seek(currentPos, 0) // Restore position
					continue
				}

				signature := make([]byte, 16)
				_, err = file.Read(signature)
				if err != nil {
					file.Seek(currentPos, 0) // Restore position
					continue
				}

				// Check for expected efisys file signature
				expectedSig := []byte{0xEB, 0xEC, 0x90, 0x4D, 0x53, 0x44, 0x4F, 0x53, 0x35, 0x2E, 0x30, 0x00, 0x02, 0x02, 0x01, 0x00}
				if bytes.Equal(signature, expectedSig) {
					// Found efisys.bin! Replace it
					_, err = file.Seek(startPos, 0)
					if err != nil {
						return err
					}

					_, err = file.Write(nopromptData)
					if err != nil {
						return err
					}

					replacements++
					Status(fmt.Sprintf("Replaced efisys.bin instance %d", replacements))
				}

				file.Seek(currentPos, 0) // Restore position
			}
		}

		filePos += int64(n)
		// Seek back a bit to handle pattern spanning buffer boundaries
		if n == len(buffer) {
			file.Seek(filePos-int64(len(searchPattern)), 0)
			filePos -= int64(len(searchPattern))
		}
	}

	if replacements == 0 {
		return fmt.Errorf("no efisys.bin instances found to replace")
	}

	Status(fmt.Sprintf("Successfully replaced %d efisys.bin instances", replacements))
	return nil
}

func DownloadVirtioDrivers(vmdir string, arch string) error {
	// Check if target architecture is ARMv7
	if arch == "arm" || arch == "ARMv7" {
		// Unless you have a custom virtio-win.iso, ARMv7 is not supported for VirtIO drivers
		return fmt.Errorf("ARMv7 architecture is not supported for VirtIO drivers")
	} else if arch == "arm64" || arch == "aarch64" || arch == "ARM64" {
		// ARM64 is supported
	} else if arch == "x86_64" || arch == "amd64" || arch == "x64" {
		// x86_64 is supported
	} else {
		return fmt.Errorf("unsupported architecture: %s", arch)
	}

	url := "https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/stable-virtio/virtio-win.iso"
	outputPath := filepath.Join(vmdir, "virtio-win.iso")

	Status("Downloading VirtIO drivers...")

	// Use the existing downloadFile function which handles progress with bubbletea
	err := downloadFile(url, outputPath)
	if err != nil {
		return fmt.Errorf("failed to download VirtIO drivers: %w", err)
	}

	Status("VirtIO drivers downloaded successfully")

	// Extract VirtIO drivers
	Status("Extracting VirtIO drivers...")
	err = extractVirtioDrivers(outputPath, vmdir, arch)
	if err != nil {
		return fmt.Errorf("failed to extract VirtIO drivers: %w", err)
	}

	// Remove the ISO file after extraction
	err = os.Remove(outputPath)
	if err != nil {
		Warning("Failed to remove virtio-win.iso after extraction: " + err.Error())
	}

	Status("VirtIO drivers extracted successfully")
	return nil
}

// extractVirtioDrivers extracts ARM64 VirtIO drivers from the ISO
func extractVirtioDrivers(isoPath, vmdir, arch string) error {
	// Create temporary directory for extraction
	tempDir, err := os.MkdirTemp("", "virtio-extract-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Try to extract ISO using 7z first (no sudo required)
	Status("Extracting VirtIO ISO contents...")
	err = runCommandWithSpinner("Extracting VirtIO ISO", "7z", "x", isoPath, "-o"+tempDir, "-y")
	if err != nil {
		// If 7z fails, try unzip as second option
		Status("7z extraction failed, trying unzip method...")
		err = runCommandWithSpinner("Extracting VirtIO ISO with unzip", "unzip", "-q", isoPath, "-d", tempDir)
		if err != nil {
			// If unzip also fails, try mounting with sudo as final fallback
			Status("Unzip extraction failed, trying mount method...")

			// Check if we can mount without password prompt first
			testCmd := exec.Command("sudo", "-n", "mount", "--help")
			if err := testCmd.Run(); err != nil {
				// Passwordless sudo not available, prompt for password upfront
				Status("Sudo password required for mounting ISO. Please enter your password:")
				fmt.Print("Password: ")

				// Use a simple approach - run sudo with a simple command to cache credentials
				sudoTestCmd := exec.Command("sudo", "-v")
				sudoTestCmd.Stdin = os.Stdin
				sudoTestCmd.Stdout = os.Stdout
				sudoTestCmd.Stderr = os.Stderr
				if err := sudoTestCmd.Run(); err != nil {
					return fmt.Errorf("failed to authenticate with sudo: %w", err)
				}
				Status("Authentication successful, proceeding with mount...")
			}

			// Mount the ISO as fallback
			err = runCommandWithSpinner("Mounting VirtIO ISO", "sudo", "mount", "-r", isoPath, tempDir)
			if err != nil {
				return fmt.Errorf("failed to mount ISO: %w", err)
			}
			defer func() {
				// Unmount the ISO
				for i := 0; i < 3; i++ {
					if err := exec.Command("sudo", "umount", tempDir).Run(); err == nil {
						break
					}
					time.Sleep(time.Second)
				}
			}()
		}
	}

	// Create unattended directory
	unattendedDir := filepath.Join(vmdir, "unattended")
	err = os.MkdirAll(unattendedDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create unattended directory: %w", err)
	}

	// Determine the architecture directory to look for
	archDir := "ARM64"
	if arch == "x86_64" || arch == "amd64" || arch == "x64" {
		archDir = "amd64"
	}

	// Find and copy drivers for Windows 11
	err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Look for architecture-specific directories under w11
		if info.IsDir() && info.Name() == archDir {
			// Check if this is under a w11 directory
			if strings.Contains(path, "/w11/") || strings.Contains(path, "\\w11\\") {
				// Extract driver name from path
				pathParts := strings.Split(filepath.ToSlash(path), "/")
				var driverName string
				for i, part := range pathParts {
					if part == "w11" && i > 0 {
						driverName = pathParts[i-1]
						break
					}
				}

				if driverName != "" {
					Status("  - " + driverName)
					destDir := filepath.Join(unattendedDir, driverName)
					err := os.MkdirAll(destDir, 0755)
					if err != nil {
						return fmt.Errorf("failed to create driver directory %s: %w", destDir, err)
					}

					// Copy all files from architecture directory
					err = copyDir(path, destDir)
					if err != nil {
						return fmt.Errorf("failed to copy %s drivers: %w", driverName, err)
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to extract drivers: %w", err)
	}

	// Copy guest agent
	guestAgentSrc := filepath.Join(tempDir, "guest-agent", "qemu-ga-x86_64.msi")
	guestAgentDir := filepath.Join(unattendedDir, "guest-agent")
	err = os.MkdirAll(guestAgentDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create guest-agent directory: %w", err)
	}

	if _, err := os.Stat(guestAgentSrc); err == nil {
		guestAgentDest := filepath.Join(guestAgentDir, "qemu-ga-x86_64.msi")
		err = copyFile(guestAgentSrc, guestAgentDest)
		if err != nil {
			return fmt.Errorf("failed to copy guest agent: %w", err)
		}
	}

	// Copy certificates
	certSrc := filepath.Join(tempDir, "cert")
	certDest := filepath.Join(unattendedDir, "cert")
	if _, err := os.Stat(certSrc); err == nil {
		err = copyDir(certSrc, certDest)
		if err != nil {
			// Don't fail the entire process if certificate copying fails
			Warning("Failed to copy certificates (non-critical): " + err.Error())
		}
	}

	return nil
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate destination path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		return copyFile(path, destPath)
	})
}

func DownloadDebloatingScript(vmdir string) error {
	//download the debloating script from https://github.com/pi-apps-go/bvm-go/blob/main/resources/debloat.ps1
	//copy the file to the vmdir/debloat.ps1
	//run the script to debloat the Windows ISO
	err := os.MkdirAll(filepath.Join(vmdir, "unattended"), 0755)
	if err != nil {
		return fmt.Errorf("failed to create unattended directory: %w", err)
	}

	// This debloat script is run by the autounattend.xml file on first login
	debloatPath := filepath.Join(vmdir, "unattended", "Win11Debloat")
	if _, err := os.Stat(debloatPath); err != nil {
		err = gitClone("https://github.com/Raphire/Win11Debloat", debloatPath)
		if err != nil {
			return err
		}
		Status("Win11Debloat repository cloned successfully")
	} else {
		Status("Win11Debloat repository already exists")
	}
	return nil
}

// helper function to clone a git repository
func gitClone(url, dir string) error {
	err := runCommandWithSpinner("Cloning Git repository", "git", "clone", url, dir)
	if err != nil {
		return fmt.Errorf("git clone of %s repository failed: %w", url, err)
	}
	return nil
}

// ValidateCustomWindowsISO validates a custom Windows ISO file
// This function is kept for backward compatibility but now only handles basic validation
// For remote URLs, validation is done during the download process to avoid downloading twice
func ValidateCustomWindowsISO(isoPath string) error {
	// Clean and resolve the path - remove quotes and trim spaces
	cleanedPath := strings.TrimSpace(isoPath)
	cleanedPath = strings.Trim(cleanedPath, "'\"") // Remove surrounding quotes
	Debug("Original ISO path: " + isoPath)
	Debug("Cleaned ISO path: " + cleanedPath)

	// For remote URLs, we can't validate without downloading, so just check URL format
	if strings.HasPrefix(cleanedPath, "http://") || strings.HasPrefix(cleanedPath, "https://") {
		Status("Remote ISO URL detected - validation will be performed during download")
		return nil
	}

	// For local files, resolve to absolute path and validate
	absPath, err := filepath.Abs(cleanedPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path %s: %v", cleanedPath, err)
	}

	// Check if file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("ISO file does not exist: %s", absPath)
	} else if err != nil {
		return fmt.Errorf("error accessing ISO file %s: %v", absPath, err)
	}

	// Validate the local ISO file
	return validateCustomWindowsISOFile(absPath)
}

// ProcessCustomWindowsISO handles a custom Windows ISO (local or remote)
func ProcessCustomWindowsISO(isoPath, vmdir string, customVirtioPath ...string) error {
	Status("Processing custom Windows ISO...")

	// Clean and resolve the path - remove quotes and trim spaces
	cleanedPath := strings.TrimSpace(isoPath)
	cleanedPath = strings.Trim(cleanedPath, "'\"") // Remove surrounding quotes
	Debug("Original ISO path: " + isoPath)
	Debug("Cleaned ISO path: " + cleanedPath)

	// Create VM directory if it doesn't exist
	if err := os.MkdirAll(vmdir, 0755); err != nil {
		return fmt.Errorf("failed to create VM directory: %v", err)
	}

	targetPath := filepath.Join(vmdir, "installer.iso")
	var localISOPath string

	if strings.HasPrefix(cleanedPath, "http://") || strings.HasPrefix(cleanedPath, "https://") {
		// For remote URLs, download directly to target location
		Status("Downloading custom Windows ISO...")
		if err := downloadFile(cleanedPath, targetPath); err != nil {
			return fmt.Errorf("failed to download custom ISO: %v", err)
		}
		localISOPath = targetPath
	} else {
		// For local files, resolve to absolute path and validate first
		absPath, err := filepath.Abs(cleanedPath)
		if err != nil {
			return fmt.Errorf("failed to resolve path %s: %v", cleanedPath, err)
		}
		cleanedPath = absPath
		Debug("Resolved ISO path to: " + cleanedPath)

		// Check if file exists
		if _, err := os.Stat(cleanedPath); os.IsNotExist(err) {
			return fmt.Errorf("ISO file does not exist: %s", cleanedPath)
		} else if err != nil {
			return fmt.Errorf("error accessing ISO file %s: %v", cleanedPath, err)
		}

		localISOPath = cleanedPath
	}

	// Validate the ISO (now we have it locally)
	Status("Validating custom Windows ISO...")
	if err := validateCustomWindowsISOFile(localISOPath); err != nil {
		// If we downloaded the file and validation failed, clean up
		if strings.HasPrefix(cleanedPath, "http://") || strings.HasPrefix(cleanedPath, "https://") {
			os.Remove(targetPath)
		}
		return err
	}

	// If it was a local file, copy it to the target location
	if !strings.HasPrefix(cleanedPath, "http://") && !strings.HasPrefix(cleanedPath, "https://") {
		Status("Copying custom Windows ISO...")
		if err := copyFile(cleanedPath, targetPath); err != nil {
			return fmt.Errorf("failed to copy custom ISO: %v", err)
		}
	}

	// Patch the ISO to boot without user intervention
	Status("Patching ISO for automatic boot...")
	if err := PatchWindowsISO(targetPath); err != nil {
		Warning("Failed to patch ISO for automatic boot: " + err.Error())
		Status("ISO will require manual keypress to boot")
	}

	// Handle VirtIO drivers
	if len(customVirtioPath) > 0 && customVirtioPath[0] != "" {
		// Custom VirtIO path provided
		customVirtio := strings.TrimSpace(customVirtioPath[0])
		customVirtio = strings.Trim(customVirtio, "'\"") // Remove surrounding quotes

		Status("Processing custom VirtIO drivers...")
		if err := processCustomVirtioDrivers(customVirtio, vmdir); err != nil {
			Warning("Failed to process custom VirtIO drivers: " + err.Error())
			Status("Falling back to default VirtIO drivers...")
			// Fall back to default VirtIO drivers
			if err := handleDefaultVirtioDrivers(vmdir); err != nil {
				return fmt.Errorf("failed to get VirtIO drivers: %v", err)
			}
		}
	} else {
		// Use default VirtIO drivers
		Status("Using default VirtIO drivers...")
		if err := handleDefaultVirtioDrivers(vmdir); err != nil {
			return fmt.Errorf("failed to get default VirtIO drivers: %v", err)
		}
	}

	return nil
}

// processCustomVirtioDrivers handles custom VirtIO drivers (ISO file, directory, or URL)
func processCustomVirtioDrivers(virtioPath, vmdir string) error {
	// Check if it's a URL
	if strings.HasPrefix(virtioPath, "http://") || strings.HasPrefix(virtioPath, "https://") {
		// Download custom VirtIO ISO
		virtioISOPath := filepath.Join(vmdir, "custom-virtio-win.iso")
		Status("Downloading custom VirtIO drivers...")
		if err := downloadFile(virtioPath, virtioISOPath); err != nil {
			return fmt.Errorf("failed to download custom VirtIO ISO: %v", err)
		}
		return extractVirtioDrivers(virtioISOPath, vmdir, runtime.GOARCH)
	}

	// Check if it's a local file or directory
	absPath, err := filepath.Abs(virtioPath)
	if err != nil {
		return fmt.Errorf("failed to resolve VirtIO path %s: %v", virtioPath, err)
	}

	stat, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("VirtIO path does not exist: %s", absPath)
	} else if err != nil {
		return fmt.Errorf("error accessing VirtIO path %s: %v", absPath, err)
	}

	if stat.IsDir() {
		// It's a directory - copy directly
		Status("Copying custom VirtIO drivers from directory...")
		unattendedDir := filepath.Join(vmdir, "unattended")
		if err := os.MkdirAll(unattendedDir, 0755); err != nil {
			return fmt.Errorf("failed to create unattended directory: %v", err)
		}
		return copyDir(absPath, unattendedDir)
	} else {
		// It's a file - assume it's an ISO
		Status("Extracting custom VirtIO drivers from ISO...")
		return extractVirtioDrivers(absPath, vmdir, runtime.GOARCH)
	}
}

// handleDefaultVirtioDrivers downloads and extracts default VirtIO drivers
func handleDefaultVirtioDrivers(vmdir string) error {
	// Use the existing DownloadVirtioDrivers function which already downloads and extracts
	return DownloadVirtioDrivers(vmdir, runtime.GOARCH)
}

// validateCustomWindowsISOFile performs all validation checks on a local ISO file
func validateCustomWindowsISOFile(isoPath string) error {
	// 1. Verify the ISO is valid
	Status("Checking ISO file integrity...")
	if err := validateISOFileStructure(isoPath); err != nil {
		return fmt.Errorf("invalid ISO file structure: %v", err)
	}

	// 2. Verify the ISO contains Windows files and check architecture
	Status("Verifying Windows files and architecture...")
	arch, err := validateWindowsFilesAndArchitecture(isoPath)
	if err != nil {
		return fmt.Errorf("failed Windows files validation: %v", err)
	}

	// 3. Verify the ISO is bootable
	Status("Checking if ISO is bootable...")
	if err := validateBootableISO(isoPath); err != nil {
		return fmt.Errorf("ISO is not bootable: %v", err)
	}

	StatusGreen(fmt.Sprintf("Custom Windows ISO validation successful! Detected architecture: %s", arch))
	return nil
}

// validateISOFileStructure checks if the file is a valid ISO
func validateISOFileStructure(isoPath string) error {
	file, err := os.Open(isoPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Check for ISO 9660 signature at offset 32769
	_, err = file.Seek(32769, 0)
	if err != nil {
		return err
	}

	signature := make([]byte, 5)
	_, err = file.Read(signature)
	if err != nil {
		return err
	}

	if string(signature) != "CD001" {
		return fmt.Errorf("not a valid ISO 9660 file")
	}

	return nil
}

// validateWindowsFilesAndArchitecture checks for Windows files and detects architecture
func validateWindowsFilesAndArchitecture(isoPath string) (string, error) {
	// Mount the ISO temporarily to check contents
	tempMount := filepath.Join(os.TempDir(), fmt.Sprintf("bvm-iso-mount-%d", time.Now().Unix()))
	if err := os.MkdirAll(tempMount, 0755); err != nil {
		return "", err
	}
	defer os.RemoveAll(tempMount)

	// Mount the ISO (Linux)
	cmd := exec.Command("sudo", "mount", "-o", "loop,ro", isoPath, tempMount)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to mount ISO for validation")
	}
	defer func() {
		exec.Command("sudo", "umount", tempMount).Run()
	}()

	// Check for essential Windows files
	requiredFiles := []string{
		"sources/boot.wim",
		"sources/install.wim",
		"bootmgr",
	}

	for _, file := range requiredFiles {
		fullPath := filepath.Join(tempMount, file)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			// Try alternative file names
			if file == "sources/install.wim" {
				// Some ISOs use install.esd instead
				altPath := filepath.Join(tempMount, "sources/install.esd")
				if _, err := os.Stat(altPath); os.IsNotExist(err) {
					return "", fmt.Errorf("missing essential Windows file: %s (also checked for install.esd)", file)
				}
			} else {
				return "", fmt.Errorf("missing essential Windows file: %s", file)
			}
		}
	}

	// Detect architecture by examining the boot.wim file
	arch, err := detectWindowsArchitecture(tempMount)
	if err != nil {
		return "", fmt.Errorf("failed to detect architecture: %v", err)
	}

	// Verify architecture compatibility with host
	hostArch := runtime.GOARCH
	if !isArchitectureCompatible(arch, hostArch) {
		return "", fmt.Errorf("architecture mismatch: ISO is %s but host is %s", arch, hostArch)
	}

	return arch, nil
}

// detectWindowsArchitecture examines Windows files to determine architecture
func detectWindowsArchitecture(mountPoint string) (string, error) {
	// Try to use wiminfo to get architecture information from boot.wim
	bootWimPath := filepath.Join(mountPoint, "sources", "boot.wim")

	cmd := exec.Command("wiminfo", bootWimPath)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to analyze boot.wim with wiminfo")
	}

	outputStr := string(output)

	// Look for architecture information in wiminfo output
	if strings.Contains(outputStr, "Architecture") {
		lines := strings.Split(outputStr, "\n")
		for _, line := range lines {
			if strings.Contains(line, "Architecture") {
				if strings.Contains(line, "ARM64") || strings.Contains(line, "aarch64") {
					return "arm64", nil
				} else if strings.Contains(line, "ARM") {
					return "arm", nil
				} else if strings.Contains(line, "x64") || strings.Contains(line, "AMD64") {
					return "amd64", nil
				} else if strings.Contains(line, "x86") {
					return "386", nil
				}
			}
		}
	}

	// Fallback: check file extensions and paths for architecture hints
	efiPath := filepath.Join(mountPoint, "efi", "boot")
	if _, err := os.Stat(efiPath); err == nil {
		// Check for EFI boot files
		files, err := os.ReadDir(efiPath)
		if err == nil {
			for _, file := range files {
				name := strings.ToLower(file.Name())
				if strings.Contains(name, "bootaa64") || strings.Contains(name, "arm64") {
					return "arm64", nil
				} else if strings.Contains(name, "bootx64") || strings.Contains(name, "x64") {
					return "amd64", nil
				} else if strings.Contains(name, "bootarm") {
					return "arm", nil
				}
			}
		}
	}

	return "", fmt.Errorf("could not determine Windows architecture from ISO contents")
}

// isArchitectureCompatible checks if the ISO architecture is compatible with the host
func isArchitectureCompatible(isoArch, hostArch string) bool {
	// Direct matches
	if isoArch == hostArch {
		return true
	}

	// ARM64 hosts can run ARM32 Windows
	if hostArch == "arm64" && isoArch == "arm" {
		return true
	}

	// AMD64 hosts can run x86 Windows (but rarely needed for VMs)
	if hostArch == "amd64" && isoArch == "386" {
		return true
	}

	return false
}

// validateBootableISO checks if the ISO has proper boot structures
func validateBootableISO(isoPath string) error {
	file, err := os.Open(isoPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Check for El Torito boot record (bootable CD signature)
	// El Torito signature is typically at sector 17 (offset 34816)
	_, err = file.Seek(34816, 0)
	if err != nil {
		return err
	}

	bootRecord := make([]byte, 32)
	_, err = file.Read(bootRecord)
	if err != nil {
		return err
	}

	// Look for El Torito signature or EFI boot structures
	bootRecordStr := string(bootRecord)
	if !strings.Contains(bootRecordStr, "EL TORITO") {
		// Check for UEFI boot structures as fallback
		if err := checkUEFIBootStructures(isoPath); err != nil {
			return fmt.Errorf("ISO does not appear to be bootable (missing El Torito or UEFI boot records)")
		}
	}

	return nil
}

// checkUEFIBootStructures looks for UEFI boot files as an alternative boot method check
func checkUEFIBootStructures(isoPath string) error {
	// Mount the ISO temporarily to check for UEFI boot files
	tempMount := filepath.Join(os.TempDir(), fmt.Sprintf("bvm-uefi-check-%d", time.Now().Unix()))
	if err := os.MkdirAll(tempMount, 0755); err != nil {
		return err
	}
	defer os.RemoveAll(tempMount)

	// Mount the ISO
	cmd := exec.Command("sudo", "mount", "-o", "loop,ro", isoPath, tempMount)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to mount ISO for UEFI check")
	}
	defer func() {
		exec.Command("sudo", "umount", tempMount).Run()
	}()

	// Check for common UEFI boot files
	uefiPaths := []string{
		"efi/boot/bootx64.efi",
		"efi/boot/bootaa64.efi",
		"efi/boot/bootarm.efi",
		"efi/microsoft/boot/cdboot.efi",
	}

	for _, path := range uefiPaths {
		fullPath := filepath.Join(tempMount, path)
		if _, err := os.Stat(fullPath); err == nil {
			return nil // Found at least one UEFI boot file
		}
	}

	return fmt.Errorf("no UEFI boot files found")
}
