![BVM](https://raw.githubusercontent.com/pi-apps-go/bvm-go/refs/heads/main/resources/graphics/icon-128.png)  
## Botspot Virtual Machine (Go Rewrite) - Windows 11 QEMU (or Xen) on ARM Linux 
![20250304_01h53m41s_grim](https://github.com/user-attachments/assets/7b368dad-eb80-4579-be2e-12a737819f24)  
The Pi 5 just became a much more capable desktop replacement. [See how Botspot is using BVM for class.](https://github.com/Botspot/bvm/discussions/39)  
Please [report](https://github.com/pi-apps-go/bvm-go/issues) issues. Do not assume I am already aware of an issue, unless you can find an existing issue report, in which case please comment with something like "I'm having this problem too."  

### What to expect:
- A full Windows 11 ARM virtual machine on ARM Linux. Thanks to [KVM](https://en.wikipedia.org/wiki/Kernel-based_Virtual_Machine), this uses virtualization instead of emulation, resulting in no significant speed difference compared to installing Windows directly.
- A first-class setup experience. None of the usual sticking points are present here, thanks to how automated it is.
- It uses network passthrough to the Linux network stack, so Ethernet and WiFi all work out of the box.
- It uses audio passthrough to pipewire/pulseaudio/ALSA, so audio playback works out of the box.
- The VM uses less than 1GB of RAM and minimal CPU when not in use, leaving plenty of resources free for Linux applications.
- The `connect` mode gives the VM access to files stored on Linux, and any changes are immediately synchronized. (Go to This PC and look for the `home` network share)
- It is capable of USB passthrough - any USB device can be made to directly communicate with Windows.
- Thanks to Microsoft's built-in [Prism emulator](https://learn.microsoft.com/en-us/windows/arm/apps-on-arm-x86-emulation), all Windows applications should work, including x86 and x64. Compare that to [Wine](https://pi-apps.io/install-app/install-wine-x64-on-raspberry-pi/), which fails on everything but old, simple programs.
- The graphics are snappy and quicker than you would expect, at least with the `connect` mode on Wayland on a Pi 5. Youtube and lightweight web games are actually somewhat playable without any overclocking or extra tweaks.
- It supports both QEMU and Xen at your choice (QEMU is default).

### What not to expect:
- A gaming rig. For now there is no graphics acceleration, so 3D features and WebGL won't work. _That could change_ once somebody figures out virtualized graphics that talk to Vulkan. (see "Other Notes" below)

**BVM Go edition is very new and is a work in progress.** Please expect feature implementations to be missing/broken and [report](https://github.com/pi-apps-go/bvm-go/issues/new) any issues you encounter.

## Requirements

- Go 1.23 or later
- Linux system with KVM support
- Required system packages: `qemu-system-arm`, `wimtools`, `genisoimage`, `mount`
- For custom ISO validation: `sudo` access for mounting ISOs

### Get started:
```
git clone https://github.com/pi-apps-go/bvm-go
cd bvm-go
make install
./bvm help
```

<details>
<summary>Click to see what BVM does on first run.</summary>

BVM will install some dependencies. At the time of writing these are:
```
git genisoimage qemu-utils qemu-system-arm qemu-system-gui remmina remmina-plugin-rdp nmap seabios ipxe-qemu wimtools
#either
wlfreerdp
#or
xfreerdp
#depending on your desktop display setup
and either
qemu-system-aarch64/qemu-system-arm/qemu-system-x86_64
#or
xen-system-x86_64/xen-system-arm/xen-system-aarch64
depending on your choice of QEMU or Xen
```
If you are on Arch or some other non-standard OS, you will need to install these manually. ~~Hey Arch users: If you want to help out, make a plugin for Pi-Apps to replace the `InstallDependencies` function with a Arch specific variant if it detects Arch, and install these dependencies, then send it to me in a Pull Request or something. I would love to support Arch but I don't use Arch.~~ Arch support is in the works in Pi-Apps Go.
If Debian Bookworm is detected, BVM will set up the `bookworm-backports` APT repository in order to upgrade QEMU from 7.2 to 9.2. Version 7.2 does work, but it is missing pipewire audio output support, and besides, everyone knows newer is always better! If you have a good reason to not want bookworm-backports, please open an issue, explain why, and I may be willing to change this behavior.  

BVM also makes some icon symlinks in `~/.local/share/icons/hicolor/scalable/apps` to set the GUI's taskbar icon, and to override the QEMU and FreeRDP taskbar icons. If this does not work on your distro, or if you do not like this feature, please get in contact with me.  
BVM now adds a menu launcher to `~/.local/share/applications/bvm.desktop` in the Office category to run the GUI. Contact me if you can make an argument that it belongs in a different category.  
BVM symlinks itself to `~/.local/bin/bvm` so it can be run in a terminal with a simple invokation of `bvm`.

</details>

### Usage instructions
Read the help message and follow the instructions. BVM has simplified the VM-creation process to a tidy sequence of completely automated steps. Between each step you have the opportunity to change what is happening, modify the config file, retry a step, or do whatever else you want. This split-step approach helps enable better learning about how it works, adjusting how it works, and creating new features on top.

To get a fresh VM up and running, use a sequence like this:  
- `bvm/bvm new-vm ~/win11`  
    This makes a config file: `~/win11/bvm-config` <--- Please read the config file!  
- `bvm/bvm download ~/win11`  
    This downloads Windows and necessary drivers.  
- `bvm/bvm prepare ~/win11`  
    This bundles everything up to get ready for first boot.  
- `bvm/bvm firstboot ~/win11`  
    This boots the Windows installer, installs Windows and some drivers, sets up a local user account, and debloats the OS. Please allow it to complete all steps automatically. When Windows finishes installing, the VM will shut down. Once it's done, you can delete all .iso files and the `unattended` folder from `~/win11` to reclaim some storage space.  
- `bvm/bvm boot ~/win11`  
    Main command to use the VM. While this does work, it's a little laggy and lacks crucial features. No copy and paste transfer, no resizable screen. It is recommended to boot the VM headless and then connect to it via RDP. (keep reading)  
- Boot Windows headless, without a screen:  
    `bvm/bvm boot-nodisplay ~/win11`  
- Then in a second terminal, connect to the RDP service:  
    `bvm/bvm connect ~/win11`  
    The connect mode has:
  - Better audio.
  - Clipboard synchronization.
  - File sharing. (Your home folder is accessible from This PC)
  - Dynamic screen resizing.
  - Higher performance graphics.
- Mount the Windows main hard drive with:  
    `bvm/bvm mount ~/win11`  
    Direct file access can be useful for troubleshooting and further debloating. Be aware: if the VM was not shut down properly, the files will mount read-only.
- Expand the Windows main hard drive with:  
    `bvm/bvm expand ~/win11`
    It will ask how many gigabytes of space to add.

Full list of modes: `new-vm`, `download`, `prepare`, `firstboot`, `boot`, `connect`, `mount`, `help`, `list-languages`, `boot-nodisplay`, `boot-ramfb`, `boot-gtk`, `connect-freerdp`, `connect-remmina`, `expand`, `gui`  
That last one there deserves a mention. BVM has a graphical user interface.  
![20250304_01h55m15s_grim](https://github.com/user-attachments/assets/cc84632d-466d-4332-b6e2-382dd9277a7b)  
Run the GUI:
```
bvm/bvm gui
```
Or open the applications menu, go to Office, click Botspot Virtual Machine.  
Right now it is quite simplistic, but functional. It might stay that way, it might not. Much of BVM's future depends on how much of an impact it makes in the community. If nobody uses BVM, then I will stop working on it and find a new project.  

### Tips:
- Use an ARM/x86 64-bit Linux OS with the `kvm` kernel module enabled. This is a hard requirement.
- Use Wayland. This is not a hard requirement, but it makes a big difference.
- Use ZRAM to avoid running out of RAM as easily. This is basically a hard requirement if you have a 1GB or 2GB Pi model, but strongly recommended everywhere. [Instructions here.](https://pi-apps.io/install-app/install-more-ram-on-raspberry-pi/)
- Use Debian Bookworm or a new-ish Ubuntu image. Debian Bullseye may or may not work. If you try it, please let me know how it went. Maybe it works fine.
- Some questions keep being asked. "How to use my own Windows ISO?" "How to change the drive size or the language or the screen resolution?" [Find the answers here.](https://github.com/Botspot/bvm/discussions/35)
- Encounter an issue? [Open an issue.](https://github.com/pi-apps-go/bvm-go/issues) Deal? Deal. :)
- Please post how you use BVM in [the Show-And-Tell](https://github.com/pi-apps-go/bvm-go/discussions/categories/show-and-tell)!

### Other announcements:
- **Gratitude**: Thanks to [Jeff Geerling](https://www.youtube.com/watch?v=mkfILjKJ8nc) and [Leepspvideo](https://www.youtube.com/watch?v=b7puJhWLQkU) for featuring the original BVM in their videos just 2 days after [release day](https://github.com/Botspot/bvm/commit/038a5b452f3b9691cc64355c8272967c17ac2e9a)!
- **Other devices:** BVM Go edition has the potential to work on all modern ARM Linux distros/CPUs and on even x86 platforms. It should be easy to support more SBCs/x86, but the pi-apps-go team only owns Raspberry Pi boards in terms of ARM. Contact the team via the issues tab if you have an ARM system not mentioned here - getting it working should be fairly straightforward if the kernel has `kvm`.
- **Expertise wanted:** If you have past experience with libvirt, I would like to ask for your ongoing help to help build out more features. Surely somebody will want to do serial passthrough, bluetooth passthrough, or something really specific that I will have no idea how to implement.
- **Windows 10 support:** It's possible, but I see little value in adding an option for it. If you have some solid reasons, I would be willing to change my mind, so please get in touch.
- **GPU driver:** Full 2D/3D acceleration may be possible using virtualized graphics that talk to Vulkan or OpenGL. I found [this](https://github.com/virtio-win/kvm-guest-drivers-windows/pull/943) unfinished series of PRs spanning multiple projects that seems promising. Read about how various graphics options can work [here](https://wiki.archlinux.org/title/QEMU/Guest_graphics_acceleration#Virgil3d_virtio-gpu_paravirtualized_device_driver). Not all apply to ARM. [This](https://github.com/tenclass/mvisor-win-vgpu-driver) other virtualized graphics code repository may be more mature. If someone reading this is "actually good" with "real programming languages," (instead of only knowing shell script like me) feel free to compile it and check if it works on ARM. (And let the community know how it goes!)
- **RemoteApp**: One exciting possibility is the chance to break out individual windows programs and integrate them directly into the linux desktop, using a RDP mode called RemoteApp. See a screenshot of this [here.](https://forums.raspberrypi.com/viewtopic.php?t=384433) Unfortunately it is [very inconsistent](https://github.com/FreeRDP/FreeRDP/issues/11218) right now, and I deemed it too unstable to include as a feature at the moment.

### Ask me anything!
- Who made this?  
    I'm [Botspot](https://github.com/Botspot), a college student, bash scripter, Raspberry Pi user, and founder of [Pi-Apps](https://github.com/Botspot/pi-apps) and [WoR-Flasher](https://github.com/Botspot/wor-flasher). If you met me in real life you would just see a friendly kid. On a more personal note, I have had a lot of family drama lately and am in need of financial support. [Read more here.](https://github.com/sponsors/Botspot)  
- Is this legal?  
    Yes. This project does not distribute copyrighted material, and the license key used is a free one provided by Microsoft for virtual machines. Read the files in `resources/` for more details.
- Is this unique?  
    Yes. Before BVM there existed no decent tool or tutorial, to do much of this. Other projects share some similar ideas though. The [QuickEMU project](https://github.com/quickemu-project/quickemu) probably comes closest, but it is not ARM-compatible. There is also the [UTM project](https://getutm.app/), which is more MacOS ARM focused. Windows ARM64 can be run in docker with the [Dockur project](https://github.com/dockur/windows-arm/). Of all the unique features here, the most unique is likely its flexibility or automation. You could change a wide variety of parameters, run all the steps in sequence with a script, walk away, and come back in a few hours to a fully-installed VM ready to use.  
    Before this, you would need to watch over it and press a key in a 5-second time window to boot the installer, install drivers, deploy some registry workarounds, then navigate through the installation steps, and debloat the OS later. BVM bypasses the keypress by patching the Windows installer ISO in a very unconventional way, (see the `patch_iso_noprompt` function), bypasses the installation steps and registry workarounds with an autounattend.xml file, and removes the bloatware using a couple PowerShell scripts. To my knowledge, nothing else does that by default with zero human intervention.  
- Why did you make this?  
    First, it will benefit me personally quite a bit over the long term. As a die-hard Raspberry Pi user without convenient access to another computer, with BVM I now have an alternative to Wine when I need to run a Windows-only program, such as Lego Mindstorms, Microsoft Office, or [this specific coding IDE needed for class](https://github.com/Botspot/bvm/discussions/39).  
    But also, I genuinely want to help you all out wherever possible. The ARM desktop community is special, but small. The more daily users we can keep around, the better the community can become for everyone. Some will oppose the promotion of a Microsoft product, saying it defeats the point of FOSS. But for most folks, it's either Linux with compromise, or no Linux at all. If we want a growing userbase, we should strongly support any project that brings a great Linux desktop experience to all beginner users.
- What was the timeline for making this?  
    I have wanted to do a KVM Windows VM on a Pi ever since I was a high-schooler with a Pi 3, so 5+ years. The task is not trivial, and no good tutorials exist. Occasionally someone would get it working, but leave behind very incomplete instructions and not respond to questions. While I gave it a try every year or so, with partial success, I never "cracked" the code completely until February 2025 when I made up my mind to sit down and figure it out. This has been a multi-week effort of near total dedication, with a number of all-night coding sessions. It became an obsession.  
    Sidenote: I probably would have not succeeded without ChatGPT, especially for the Windows automation stuff. Anyone who says LLMs are useless is wrong.

~~If communication on github is not your thing, [join my Discord server!](https://discord.gg/RXSTvaUvuu)~~ We are planning our own Discord server once it's generally finished.
