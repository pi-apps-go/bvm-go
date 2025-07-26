package cli

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pi-apps-go/bvm-go/internal"
)

type item struct {
	title string
	desc  string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type selectionState int

const (
	selectingVersion selectionState = iota
	selectingArch
	selectingLanguage
	selectingEdition
	selectingCustomISO
	selectingCustomVirtio
	downloading
)

type downloadModel struct {
	state  selectionState
	vmName string

	// Selection lists
	versionList  list.Model
	archList     list.Model
	languageList list.Model
	editionList  list.Model

	// Custom ISO input
	customISOInput    textinput.Model
	customVirtioInput textinput.Model

	// Selected values
	selectedVersion      string
	selectedArch         string
	selectedLanguage     string
	selectedEdition      string
	selectedCustomISO    string
	selectedCustomVirtio string

	width  int
	height int

	hostCaps HostCapabilities
}

// HostCapabilities represents what the current hardware can run
type HostCapabilities struct {
	Architecture    string // "arm64", "amd64", "arm"
	SupportsAtomics bool   // For ARM64: whether CPU supports atomics instruction
	IsARMv9         bool   // For ARM64: whether this is an ARMv9 CPU (no 32-bit support)
	HostDescription string // Human-readable description
}

// detectHostCapabilities determines what Windows versions this hardware can run
func detectHostCapabilities() HostCapabilities {
	caps := HostCapabilities{
		Architecture: runtime.GOARCH,
	}

	switch runtime.GOARCH {
	case "arm64":
		// Check for atomics support and ARMv9 on ARM64
		caps.SupportsAtomics = internal.IsAtomicsCPU()
		caps.IsARMv9 = internal.IsARMv9CPU()

		if caps.IsARMv9 {
			caps.HostDescription = "ARMv9 (modern ARM64 with no 32-bit support)"
		} else if caps.SupportsAtomics {
			caps.HostDescription = "ARM64 with atomics support (Pi 5, modern SBCs, Apple Silicon)"
		} else {
			caps.HostDescription = "ARM64 without atomics (Pi 4, older ARMv8.0 CPUs)"
		}
	case "amd64":
		caps.HostDescription = "x64/AMD64 (Intel, AMD processors)"
	case "arm":
		caps.HostDescription = "ARMv7 (Pi 2/3, older ARM devices)"
	default:
		caps.HostDescription = "Unknown architecture: " + runtime.GOARCH
	}

	return caps
}

// getCompatibleVersions returns Windows versions compatible with this hardware
func getCompatibleVersions(caps HostCapabilities) []list.Item {
	var items []list.Item

	switch caps.Architecture {
	case "arm64":
		if caps.IsARMv9 {
			// ARMv9 processors - can run latest Windows 11, but NO ARMv7 builds
			items = []list.Item{
				item{title: "Windows 11", desc: "Latest Windows version (recommended for your ARMv9 CPU)"},
				item{title: "Windows 10", desc: "Previous Windows version (ARM64 only - no ARMv7 support)"},
			}
		} else if caps.SupportsAtomics {
			// Modern ARM64 with atomics - can run latest Windows 11
			items = []list.Item{
				item{title: "Windows 11", desc: "Latest Windows version (recommended for your ARM64 CPU with atomics)"},
				item{title: "Windows 10", desc: "Previous Windows version (also compatible)"},
			}
		} else {
			// ARM64 without atomics - limited to older builds
			items = []list.Item{
				item{title: "Windows 11", desc: "Build 22631 only (compatible with your ARMv8.0 CPU)"},
				item{title: "Windows 10", desc: "Previous Windows version (recommended for your hardware)"},
			}
		}
	case "amd64":
		// x64 can run both Windows 10 and 11
		items = []list.Item{
			item{title: "Windows 11", desc: "Latest Windows version (recommended for x64)"},
			item{title: "Windows 10", desc: "Previous Windows version"},
		}
	case "arm":
		// ARMv7 only supports Windows 10 build 15035
		items = []list.Item{
			item{title: "Windows 10", desc: "Build 15035 only (only Windows version for ARMv7)"},
		}
	}

	// Always add Custom ISO option at the end
	items = append(items, item{title: "Custom ISO", desc: "Use your own Windows ISO file (local or remote)"})

	return items
}

// getCompatibleArchitectures returns architectures compatible with selected version and host
func getCompatibleArchitectures(selectedVersion string, caps HostCapabilities) []list.Item {
	var items []list.Item

	switch caps.Architecture {
	case "arm64":
		if selectedVersion == "Windows 11" {
			if caps.IsARMv9 || caps.SupportsAtomics {
				items = []list.Item{
					item{title: "ARM64 (Latest)", desc: "Latest build for your modern ARM64 processor"},
				}
			} else {
				items = []list.Item{
					item{title: "ARM64 (22631)", desc: "Build 22631 for your ARMv8.0 CPU (Pi 4 compatible)"},
				}
			}
		} else { // Windows 10
			if caps.IsARMv9 {
				// ARMv9 processors don't support 32-bit, so only ARM64 builds
				items = []list.Item{
					item{title: "ARM64", desc: "ARM64 build for your ARMv9 processor (no ARMv7 support)"},
				}
			} else {
				// ARMv8 processors can run both ARM64 and ARMv7 builds
				items = []list.Item{
					item{title: "ARM64", desc: "ARM64 build for your processor"},
					item{title: "ARMv7", desc: "ARMv7 build 15035 (32-bit compatibility)"},
				}
			}
		}
	case "amd64":
		items = []list.Item{
			item{title: "x64", desc: "x64 build for your processor"},
		}
	case "arm":
		if selectedVersion == "Windows 10" {
			items = []list.Item{
				item{title: "ARMv7", desc: "ARMv7 build 15035 for your processor"},
			}
		}
		// Windows 11 not supported on ARMv7
	}

	return items
}

func DownloadCLI(vmName string) tea.Model {
	// Detect host capabilities
	caps := detectHostCapabilities()

	// Get compatible Windows versions for this hardware
	versionItems := getCompatibleVersions(caps)

	if len(versionItems) == 0 {
		// This should never happen, but handle gracefully
		versionItems = []list.Item{
			item{title: "No compatible versions", desc: "Your architecture is not supported"},
		}
	}

	versionList := list.New(versionItems, list.NewDefaultDelegate(), 0, 0)
	versionList.Title = fmt.Sprintf("Select Windows Version (Host: %s)", caps.HostDescription)
	versionList.SetShowStatusBar(false)
	versionList.SetFilteringEnabled(false)
	versionList.Styles.Title = titleStyle

	return downloadModel{
		state:       selectingVersion,
		vmName:      vmName,
		versionList: versionList,
		hostCaps:    caps,
	}
}

func (m downloadModel) Init() tea.Cmd {
	return nil
}

func (m downloadModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.versionList.SetWidth(msg.Width)
		m.versionList.SetHeight(msg.Height - 4)

		// Only update dimensions for lists that have been initialized
		if m.state >= selectingArch && len(m.archList.Items()) > 0 {
			m.archList.SetWidth(msg.Width)
			m.archList.SetHeight(msg.Height - 4)
		}
		if m.state >= selectingLanguage && len(m.languageList.Items()) > 0 {
			m.languageList.SetWidth(msg.Width)
			m.languageList.SetHeight(msg.Height - 4)
		}
		if m.state >= selectingEdition && len(m.editionList.Items()) > 0 {
			m.editionList.SetWidth(msg.Width)
			m.editionList.SetHeight(msg.Height - 4)
		}
		if m.state == selectingCustomISO {
			m.customISOInput.Width = msg.Width - 4
		}
		if m.state == selectingCustomVirtio {
			m.customVirtioInput.Width = msg.Width - 4
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			return m.handleSelection()
		case "esc":
			return m.goBack()
		}
	}

	var cmd tea.Cmd
	switch m.state {
	case selectingVersion:
		m.versionList, cmd = m.versionList.Update(msg)
	case selectingArch:
		m.archList, cmd = m.archList.Update(msg)
	case selectingLanguage:
		m.languageList, cmd = m.languageList.Update(msg)
	case selectingEdition:
		m.editionList, cmd = m.editionList.Update(msg)
	case selectingCustomISO:
		m.customISOInput, cmd = m.customISOInput.Update(msg)
	case selectingCustomVirtio:
		m.customVirtioInput, cmd = m.customVirtioInput.Update(msg)
	}

	return m, cmd
}

func (m downloadModel) handleSelection() (tea.Model, tea.Cmd) {
	switch m.state {
	case selectingVersion:
		if m.versionList.SelectedItem() == nil {
			return m, nil
		}
		selectedItem := m.versionList.SelectedItem().(item)
		m.selectedVersion = selectedItem.title

		// Handle Custom ISO selection
		if m.selectedVersion == "Custom ISO" {
			return m.setupCustomISOInput()
		}

		return m.setupArchSelection()

	case selectingArch:
		if m.archList.SelectedItem() == nil {
			return m, nil
		}
		selectedItem := m.archList.SelectedItem().(item)
		m.selectedArch = selectedItem.title

		// Skip language selection for ARMv7 (only English supported)
		if m.selectedArch == "ARMv7" {
			m.selectedLanguage = "English (United States)"
			return m.startDownload()
		}

		return m.setupLanguageSelection()

	case selectingLanguage:
		if m.languageList.SelectedItem() == nil {
			return m, nil
		}
		selectedItem := m.languageList.SelectedItem().(item)
		m.selectedLanguage = selectedItem.title

		// Check if we need edition selection (Windows 11 build 22631)
		if m.selectedVersion == "Windows 11" && m.selectedArch == "ARM64 (22631)" {
			return m.setupEditionSelection()
		}

		return m.startDownload()

	case selectingEdition:
		if m.editionList.SelectedItem() == nil {
			return m, nil
		}
		selectedItem := m.editionList.SelectedItem().(item)
		m.selectedEdition = selectedItem.title
		return m.startDownload()

	case selectingCustomISO:
		isoPath := strings.TrimSpace(m.customISOInput.Value())
		if isoPath == "" {
			return m, nil
		}
		m.selectedCustomISO = isoPath
		return m.setupCustomVirtioInput()

	case selectingCustomVirtio:
		virtioPath := strings.TrimSpace(m.customVirtioInput.Value())
		m.selectedCustomVirtio = virtioPath // Can be empty for no custom VirtIO
		return m.startDownload()
	}

	return m, nil
}

func (m downloadModel) goBack() (tea.Model, tea.Cmd) {
	switch m.state {
	case selectingArch:
		m.state = selectingVersion
		return m, nil
	case selectingLanguage:
		m.state = selectingArch
		return m, nil
	case selectingEdition:
		m.state = selectingLanguage
		return m, nil
	case selectingCustomISO:
		m.state = selectingVersion
		return m, nil
	case selectingCustomVirtio:
		m.state = selectingCustomISO
		return m, nil
	}
	return m, nil
}

func (m downloadModel) setupArchSelection() (tea.Model, tea.Cmd) {
	// Get compatible architectures for the selected version and host hardware
	archItems := getCompatibleArchitectures(m.selectedVersion, m.hostCaps)

	if len(archItems) == 0 {
		// This should never happen with proper filtering, but handle gracefully
		archItems = []list.Item{
			item{title: "No compatible architectures", desc: "Selected version not supported on your hardware"},
		}
	}

	m.archList = list.New(archItems, list.NewDefaultDelegate(), m.width, m.height-4)
	m.archList.Title = fmt.Sprintf("Select Architecture (Host: %s)", m.hostCaps.HostDescription)
	m.archList.SetShowStatusBar(false)
	m.archList.SetFilteringEnabled(false)
	m.archList.Styles.Title = titleStyle
	m.state = selectingArch

	return m, nil
}

func (m downloadModel) setupLanguageSelection() (tea.Model, tea.Cmd) {
	// Parse available languages from internal.ListDownloadLanguages()
	languagesStr := internal.ListDownloadLanguages()
	languageLines := strings.Split(languagesStr, "\n")

	var languageItems []list.Item
	for _, line := range languageLines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) == 2 {
			code := parts[0]
			name := parts[1]

			// Add special handling for Windows 11 build 22631
			desc := fmt.Sprintf("Language code: %s", code)
			if m.selectedVersion == "Windows 11" && m.selectedArch == "ARM64 (22631)" {
				desc += " (Build 22631 - requires edition selection)"
			}

			languageItems = append(languageItems, item{
				title: name,
				desc:  desc,
			})
		}
	}

	m.languageList = list.New(languageItems, list.NewDefaultDelegate(), m.width, m.height-4)
	m.languageList.Title = "Select Language"
	m.languageList.SetShowStatusBar(false)
	m.languageList.SetFilteringEnabled(true)
	m.languageList.Styles.Title = titleStyle
	m.state = selectingLanguage

	return m, nil
}

func (m downloadModel) setupEditionSelection() (tea.Model, tea.Cmd) {
	// Standard editions available worldwide
	editionItems := []list.Item{
		item{title: "Home", desc: "Windows Home edition (recommended for personal use)"},
		item{title: "Pro", desc: "Windows Professional edition (recommended for advanced users)"},
		item{title: "Pro Education", desc: "Windows Pro Education edition (for educational institutions)"},
		item{title: "Pro for Workstations", desc: "Windows Pro for Workstations edition (for high-end hardware)"},
		item{title: "Education", desc: "Windows Education edition (for educational institutions)"},
	}

	// Add N variants only for European languages (due to EU antitrust rulings)
	if m.isEuropeanLanguage(m.selectedLanguage) {
		nVariants := []list.Item{
			item{title: "Home N", desc: "Windows Home N edition (without Media Player)"},
			item{title: "Pro N", desc: "Windows Professional N edition (without Media Player)"},
			item{title: "Pro N for Workstations", desc: "Windows Pro N for Workstations (without Media Player)"},
			item{title: "Pro Education N", desc: "Windows Pro Education N edition (without Media Player)"},
			item{title: "Education N", desc: "Windows Education N edition (without Media Player)"},
		}
		editionItems = append(editionItems, nVariants...)
	}

	m.editionList = list.New(editionItems, list.NewDefaultDelegate(), m.width, m.height-4)
	if m.isEuropeanLanguage(m.selectedLanguage) {
		m.editionList.Title = "Select Windows Edition (includes N variants)"
	} else {
		m.editionList.Title = "Select Windows Edition"
	}
	m.editionList.SetShowStatusBar(false)
	m.editionList.SetFilteringEnabled(true)
	m.editionList.Styles.Title = titleStyle
	m.state = selectingEdition

	return m, nil
}

func (m downloadModel) setupCustomISOInput() (tea.Model, tea.Cmd) {
	m.customISOInput = textinput.New()
	m.customISOInput.Placeholder = "Enter path to local ISO file or URL to remote ISO"
	m.customISOInput.Focus()
	m.customISOInput.CharLimit = 512
	m.customISOInput.Width = m.width - 4
	m.state = selectingCustomISO
	return m, nil
}

func (m downloadModel) setupCustomVirtioInput() (tea.Model, tea.Cmd) {
	m.customVirtioInput = textinput.New()
	m.customVirtioInput.Placeholder = "Enter path to custom VirtIO drivers (optional - leave blank for default)"
	m.customVirtioInput.Focus()
	m.customVirtioInput.CharLimit = 512
	m.customVirtioInput.Width = m.width - 4
	m.state = selectingCustomVirtio
	return m, nil
}

// isEuropeanLanguage determines if a language qualifies for N variants due to EU antitrust rulings
func (m downloadModel) isEuropeanLanguage(language string) bool {
	europeanLanguages := map[string]bool{
		"German":                  true,  // de-de
		"French":                  true,  // fr-fr
		"French Canadian":         false, // fr-ca (not EU)
		"Italian":                 true,  // it-it
		"Spanish":                 true,  // es-es
		"Spanish (Mexico)":        false, // es-mx (not EU)
		"Dutch":                   true,  // nl-nl
		"Polish":                  true,  // pl-pl
		"Portuguese":              true,  // pt-pt
		"Swedish":                 true,  // sv-se
		"Danish":                  true,  // da-dk
		"Finnish":                 true,  // fi-fi
		"Norwegian":               true,  // nb-no
		"Czech":                   true,  // cs-cz
		"Slovak":                  true,  // sk-sk
		"Hungarian":               true,  // hu-hu
		"Romanian":                true,  // ro-ro
		"Bulgarian":               true,  // bg-bg
		"Croatian":                true,  // hr-hr
		"Slovenian":               true,  // sl-si
		"Estonian":                true,  // et-ee
		"Latvian":                 true,  // lv-lv
		"Lithuanian":              true,  // lt-lt
		"Greek":                   true,  // el-gr
		"Serbian Latin":           false, // sr-latn-rs (not EU member)
		"English (United States)": false, // en-us (not EU)
		"English International":   false, // en-gb (UK left EU, and N variants don't apply to UK anyway)
		"Brazilian Portuguese":    false, // pt-br (not EU)
		"Chinese (Simplified)":    false, // zh-cn
		"Chinese (Traditional)":   false, // zh-tw
		"Japanese":                false, // ja-jp
		"Korean":                  false, // ko-kr
		"Arabic":                  false, // ar-sa
		"Russian":                 false, // ru-ru (not EU member)
		"Turkish":                 false, // tr-tr (not EU member)
		"Ukrainian":               false, // uk-ua (not EU member, and currently in conflict)
		"Thai":                    false, // th-th
		"Hebrew":                  false, // he-il
	}

	return europeanLanguages[language]
}

type DownloadSelections struct {
	SelectedVersion      string
	SelectedArch         string
	SelectedLanguage     string
	SelectedEdition      string
	SelectedCustomISO    string
	SelectedCustomVirtio string
	VmName               string
}

type DownloadModel interface {
	tea.Model
	GetSelections() *DownloadSelections
}

func (m downloadModel) startDownload() (tea.Model, tea.Cmd) {
	m.state = downloading
	return m, tea.Quit
}

func (m downloadModel) GetSelections() *DownloadSelections {
	if m.state == downloading {
		return &DownloadSelections{
			SelectedVersion:      m.selectedVersion,
			SelectedArch:         m.selectedArch,
			SelectedLanguage:     m.selectedLanguage,
			SelectedEdition:      m.selectedEdition,
			SelectedCustomISO:    m.selectedCustomISO,
			SelectedCustomVirtio: m.selectedCustomVirtio,
			VmName:               m.vmName,
		}
	}
	return nil
}

func (m downloadModel) View() string {
	switch m.state {
	case selectingVersion:
		return fmt.Sprintf("%s\n%s\n\n%s",
			headerStyle.Render("BVM - Download Windows"),
			infoStyle.Render("ℹ Only showing Windows versions compatible with your hardware"),
			m.versionList.View())

	case selectingArch:
		return fmt.Sprintf("%s\n%s\n%s\n\n%s",
			headerStyle.Render("BVM - Download Windows"),
			fmt.Sprintf("Selected: %s", m.selectedVersion),
			infoStyle.Render("ℹ Only showing architectures compatible with your hardware"),
			m.archList.View())

	case selectingLanguage:
		return fmt.Sprintf("%s\n%s\n\n%s",
			headerStyle.Render("BVM - Download Windows"),
			fmt.Sprintf("Selected: %s %s", m.selectedVersion, m.selectedArch),
			m.languageList.View())

	case selectingEdition:
		return fmt.Sprintf("%s\n%s\n\n%s",
			headerStyle.Render("BVM - Download Windows"),
			fmt.Sprintf("Selected: %s %s - %s", m.selectedVersion, m.selectedArch, m.selectedLanguage),
			m.editionList.View())

	case selectingCustomISO:
		return fmt.Sprintf("%s\n%s\n\n%s\n%s\n\n%s\n%s",
			headerStyle.Render("BVM - Download Windows"),
			fmt.Sprintf("Selected: %s", m.selectedVersion),
			infoStyle.Render("Enter the path to a local Windows ISO file or a URL to a remote ISO file."),
			infoStyle.Render("The ISO will be validated to ensure it's a bootable Windows image with the correct architecture."),
			"ISO Path or URL:",
			m.customISOInput.View())

	case selectingCustomVirtio:
		return fmt.Sprintf("%s\n%s\n\n%s\n%s\n%s\n\n%s\n%s",
			headerStyle.Render("BVM - Download Windows"),
			fmt.Sprintf("Selected: %s", m.selectedVersion),
			infoStyle.Render("Enter the path to custom VirtIO drivers for your Windows version."),
			infoStyle.Render("This can be a local ISO file, a directory with extracted drivers, or a URL to a remote ISO."),
			infoStyle.Render("Leave blank to use the default VirtIO drivers (recommended for most cases)."),
			"Custom VirtIO Path (optional):",
			m.customVirtioInput.View())

	case downloading:
		if m.selectedVersion == "Custom ISO" {
			virtioInfo := ""
			if m.selectedCustomVirtio != "" {
				virtioInfo = fmt.Sprintf("\nCustom VirtIO: %s", m.selectedCustomVirtio)
			}
			return fmt.Sprintf("%s\n\nValidating and preparing custom ISO...\n%s%s",
				headerStyle.Render("BVM - Download Windows"),
				m.selectedCustomISO,
				virtioInfo)
		}
		return fmt.Sprintf("%s\n\nStarting download...\n%s %s - %s",
			headerStyle.Render("BVM - Download Windows"),
			m.selectedVersion, m.selectedArch, m.selectedLanguage)
	}

	return ""
}

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(1).
			MarginBottom(1)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Italic(true)
)
