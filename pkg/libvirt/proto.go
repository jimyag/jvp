package libvirt

import "encoding/xml"

// DomainXML represents the root domain XML structure
// Reference: https://libvirt.org/formatdomain.html
type DomainXML struct {
	XMLName xml.Name `xml:"domain"`
	Type    string   `xml:"type,attr"`

	// Basic metadata
	// Source: https://libvirt.org/formatdomain.html#general-metadata
	Name        string `xml:"name"`
	UUID        string `xml:"uuid,omitempty"`        // RFC 4122 compliant UUID
	Title       string `xml:"title,omitempty"`       // Short description without newlines
	Description string `xml:"description,omitempty"` // Detailed human-readable description

	// Memory configuration
	// Source: https://libvirt.org/formatdomain.html#memory-allocation
	Memory        DomainMemory `xml:"memory"`
	CurrentMemory DomainMemory `xml:"currentMemory,omitempty"` // Actual memory allocation, can be less than max for ballooning

	// CPU configuration
	// Source: https://libvirt.org/formatdomain.html#cpu-model-and-topology
	VCPU DomainVCPU `xml:"vcpu"`
	CPU  *DomainCPU `xml:"cpu,omitempty"` // Detailed CPU requirements (model, topology, features)

	// OS and boot
	// Source: https://libvirt.org/formatdomain.html#operating-system-booting
	OS DomainOS `xml:"os"`

	// Hypervisor features
	// Source: https://libvirt.org/formatdomain.html#hypervisor-features
	Features *DomainFeatures `xml:"features,omitempty"` // ACPI, APIC, PAE, etc.

	// Clock and timers
	// Source: https://libvirt.org/formatdomain.html#time-keeping
	Clock *DomainClock `xml:"clock,omitempty"`

	// Lifecycle management
	// Source: https://libvirt.org/formatdomain.html#events-configuration
	OnPoweroff string `xml:"on_poweroff,omitempty"` // Action on poweroff: destroy, restart, preserve, rename-restart
	OnReboot   string `xml:"on_reboot,omitempty"`   // Action on reboot: destroy, restart, preserve, rename-restart
	OnCrash    string `xml:"on_crash,omitempty"`    // Action on crash: destroy, restart, preserve, rename-restart, coredump-destroy, coredump-restart

	// Devices
	// Source: https://libvirt.org/formatdomain.html#devices
	Devices DomainDevices `xml:"devices"`
}

// DomainMemory represents memory configuration
type DomainMemory struct {
	Unit  string `xml:"unit,attr"`
	Value uint64 `xml:",chardata"`
}

// DomainVCPU represents virtual CPU configuration
type DomainVCPU struct {
	Placement string `xml:"placement,attr"`
	Value     int    `xml:",chardata"`
}

// DomainOS represents operating system configuration
type DomainOS struct {
	Type DomainOSType `xml:"type"`
	Boot DomainBoot   `xml:"boot"`
}

// DomainOSType represents OS type details
type DomainOSType struct {
	Arch    string `xml:"arch,attr"`
	Machine string `xml:"machine,attr"`
	Value   string `xml:",chardata"`
}

// DomainBoot represents boot configuration
type DomainBoot struct {
	Dev string `xml:"dev,attr"`
}

// DomainDevices represents all devices in the domain
// Source: https://libvirt.org/formatdomain.html#devices
type DomainDevices struct {
	Emulator   string            `xml:"emulator"`
	Disks      []DomainDisk      `xml:"disk"`
	Interfaces []DomainInterface `xml:"interface"`
	Graphics   DomainGraphics    `xml:"graphics"`
	Serial     DomainSerial      `xml:"serial"`
	Console    DomainConsole     `xml:"console"`

	// Additional device types
	// Source: https://libvirt.org/formatdomain.html#hard-drives-floppy-disks-cdroms
	Controllers []DomainController `xml:"controller,omitempty"` // USB, PCI, SCSI, IDE controllers
	Videos      []DomainVideo      `xml:"video,omitempty"`      // Video devices
	Inputs      []DomainInput      `xml:"input,omitempty"`      // Input devices (keyboard, mouse, tablet)
	Sounds      []DomainSound      `xml:"sound,omitempty"`      // Audio devices
	Hostdevs    []DomainHostdev    `xml:"hostdev,omitempty"`    // Host device passthrough
	Watchdogs   []DomainWatchdog   `xml:"watchdog,omitempty"`   // Watchdog devices
	Channels    []DomainChannel    `xml:"channel,omitempty"`    // Communication channels (guest agent)
	MemBalloon  *DomainMemBalloon  `xml:"memballoon,omitempty"` // Memory balloon device
	RNG         *DomainRNG         `xml:"rng,omitempty"`        // Random number generator
	TPM         *DomainTPM         `xml:"tpm,omitempty"`        // TPM device
}

// DomainDisk represents a disk device
type DomainDisk struct {
	Type   string           `xml:"type,attr"`
	Device string           `xml:"device,attr"`
	Driver DomainDiskDriver `xml:"driver"`
	Source DomainDiskSource `xml:"source"`
	Target DomainDiskTarget `xml:"target"`
}

// DomainDiskDriver represents disk driver configuration
type DomainDiskDriver struct {
	Name string `xml:"name,attr"`
	Type string `xml:"type,attr"`
}

// DomainDiskSource represents disk source configuration
type DomainDiskSource struct {
	Pool   string `xml:"pool,attr,omitempty"`
	Volume string `xml:"volume,attr,omitempty"`
	File   string `xml:"file,attr,omitempty"`
}

// DomainDiskTarget represents disk target configuration
type DomainDiskTarget struct {
	Dev string `xml:"dev,attr"`
	Bus string `xml:"bus,attr"`
}

// DomainInterface represents a network interface
type DomainInterface struct {
	Type   string                `xml:"type,attr"`
	Source DomainInterfaceSource `xml:"source"`
	Model  DomainInterfaceModel  `xml:"model"`
}

// DomainInterfaceSource represents network interface source
type DomainInterfaceSource struct {
	Network string `xml:"network,attr,omitempty"`
	Bridge  string `xml:"bridge,attr,omitempty"`
	Dev     string `xml:"dev,attr,omitempty"`
	Mode    string `xml:"mode,attr,omitempty"`
	MAC     string `xml:"mac,attr,omitempty"`
}

// DomainInterfaceModel represents network interface model
type DomainInterfaceModel struct {
	Type string `xml:"type,attr"`
}

// DomainGraphics represents graphics configuration
type DomainGraphics struct {
	Type     string                `xml:"type,attr"`
	Port     int                   `xml:"port,attr,omitempty"`
	Autoport string                `xml:"autoport,attr,omitempty"`
	Socket   string                `xml:"socket,attr,omitempty"` // Unix socket path for VNC
	Listen   *DomainGraphicsListen `xml:"listen,omitempty"`      // Use pointer so omitempty works correctly
}

// DomainGraphicsListen represents graphics listen configuration
type DomainGraphicsListen struct {
	Type string `xml:"type,attr"` // Required when listen element is present
}

// DomainSerial represents serial device configuration
type DomainSerial struct {
	Type   string             `xml:"type,attr"`
	Target DomainSerialTarget `xml:"target"`
}

// DomainSerialTarget represents serial target configuration
type DomainSerialTarget struct {
	Type  string                  `xml:"type,attr"`
	Port  int                     `xml:"port,attr"`
	Model DomainSerialTargetModel `xml:"model"`
}

// DomainSerialTargetModel represents serial target model
type DomainSerialTargetModel struct {
	Name string `xml:"name,attr"`
}

// DomainConsole represents console device configuration
type DomainConsole struct {
	Type   string              `xml:"type,attr"`
	Source DomainConsoleSource `xml:"source,omitempty"`
	Target DomainConsoleTarget `xml:"target"`
}

// DomainConsoleSource represents console source configuration (PTY path)
type DomainConsoleSource struct {
	Path string `xml:"path,attr,omitempty"` // PTY device path (assigned at runtime)
}

// DomainConsoleTarget represents console target configuration
type DomainConsoleTarget struct {
	Type string `xml:"type,attr"`
	Port int    `xml:"port,attr"`
}

// DomainCPU represents CPU configuration
// Source: https://libvirt.org/formatdomain.html#cpu-model-and-topology
type DomainCPU struct {
	Mode     string          `xml:"mode,attr,omitempty"`  // custom, host-model, host-passthrough
	Match    string          `xml:"match,attr,omitempty"` // minimum, exact, strict
	Check    string          `xml:"check,attr,omitempty"` // none, partial, full
	Model    *DomainCPUModel `xml:"model,omitempty"`      // CPU model name
	Vendor   string          `xml:"vendor,omitempty"`     // CPU vendor
	Topology *DomainTopology `xml:"topology,omitempty"`   // CPU topology (sockets, cores, threads)
	Features []DomainFeature `xml:"feature,omitempty"`    // CPU features to enable/disable
	Numa     *DomainNuma     `xml:"numa,omitempty"`       // NUMA topology
}

// DomainCPUModel represents CPU model configuration
type DomainCPUModel struct {
	Fallback string `xml:"fallback,attr,omitempty"` // allow, forbid
	Value    string `xml:",chardata"`               // Model name (e.g., "core2duo", "Haswell")
}

// DomainTopology represents CPU topology
type DomainTopology struct {
	Sockets int `xml:"sockets,attr"` // Number of CPU sockets
	Cores   int `xml:"cores,attr"`   // Number of cores per socket
	Threads int `xml:"threads,attr"` // Number of threads per core
}

// DomainFeature represents a CPU feature
type DomainFeature struct {
	Policy string `xml:"policy,attr"` // force, require, optional, disable, forbid
	Name   string `xml:"name,attr"`   // Feature name
}

// DomainNuma represents NUMA topology configuration
type DomainNuma struct {
	Cells []DomainCell `xml:"cell"` // NUMA cells
}

// DomainCell represents a NUMA cell
type DomainCell struct {
	ID     int    `xml:"id,attr"`             // Cell ID
	CPUs   string `xml:"cpus,attr"`           // CPU list (e.g., "0-3")
	Memory uint64 `xml:"memory,attr"`         // Memory in KiB
	Unit   string `xml:"unit,attr,omitempty"` // Memory unit
}

// DomainFeatures represents hypervisor features
// Source: https://libvirt.org/formatdomain.html#hypervisor-features
type DomainFeatures struct {
	ACPI     *DomainFeatureEnabled `xml:"acpi,omitempty"`     // ACPI support for power management
	APIC     *DomainFeatureEnabled `xml:"apic,omitempty"`     // APIC support
	PAE      *DomainFeatureEnabled `xml:"pae,omitempty"`      // Physical Address Extension
	HAP      *DomainFeatureEnabled `xml:"hap,omitempty"`      // Hardware Assisted Paging
	Viridian *DomainFeatureEnabled `xml:"viridian,omitempty"` // Hyper-V enlightenments
	PrivNet  *DomainFeatureEnabled `xml:"privnet,omitempty"`  // Private network namespace
	HyperV   *DomainHyperV         `xml:"hyperv,omitempty"`   // Hyper-V enlightenments for Windows guests
	KVM      *DomainKVM            `xml:"kvm,omitempty"`      // KVM specific features
	VMPort   *DomainFeatureState   `xml:"vmport,omitempty"`   // VMWare IO port emulation
}

// DomainFeatureEnabled represents a simple enabled feature
type DomainFeatureEnabled struct{}

// DomainFeatureState represents a feature with state
type DomainFeatureState struct {
	State string `xml:"state,attr,omitempty"` // on, off
}

// DomainHyperV represents Hyper-V enlightenments
type DomainHyperV struct {
	Relaxed         *DomainFeatureState `xml:"relaxed,omitempty"`
	VAPIC           *DomainFeatureState `xml:"vapic,omitempty"`
	Spinlocks       *DomainSpinlocks    `xml:"spinlocks,omitempty"`
	VPIndex         *DomainFeatureState `xml:"vpindex,omitempty"`
	Runtime         *DomainFeatureState `xml:"runtime,omitempty"`
	Synic           *DomainFeatureState `xml:"synic,omitempty"`
	STimer          *DomainFeatureState `xml:"stimer,omitempty"`
	Reset           *DomainFeatureState `xml:"reset,omitempty"`
	VendorID        *DomainVendorID     `xml:"vendor_id,omitempty"`
	Frequencies     *DomainFeatureState `xml:"frequencies,omitempty"`
	ReEnlightenment *DomainFeatureState `xml:"reenlightenment,omitempty"`
	TLBFlush        *DomainFeatureState `xml:"tlbflush,omitempty"`
	IPI             *DomainFeatureState `xml:"ipi,omitempty"`
	EVMCS           *DomainFeatureState `xml:"evmcs,omitempty"`
}

// DomainSpinlocks represents spinlock configuration
type DomainSpinlocks struct {
	State   string `xml:"state,attr,omitempty"`
	Retries int    `xml:"retries,attr,omitempty"`
}

// DomainVendorID represents vendor ID configuration
type DomainVendorID struct {
	State string `xml:"state,attr,omitempty"`
	Value string `xml:"value,attr,omitempty"`
}

// DomainKVM represents KVM specific features
type DomainKVM struct {
	Hidden        *DomainFeatureState `xml:"hidden,omitempty"`
	HintDedicated *DomainFeatureState `xml:"hint-dedicated,omitempty"`
	PollControl   *DomainFeatureState `xml:"poll-control,omitempty"`
	PVSpinlock    *DomainFeatureState `xml:"pv-spinlock,omitempty"`
}

// DomainClock represents clock and timer configuration
// Source: https://libvirt.org/formatdomain.html#time-keeping
type DomainClock struct {
	Offset     string        `xml:"offset,attr"`               // utc, localtime, timezone, variable
	Timezone   string        `xml:"timezone,attr,omitempty"`   // Timezone name when offset=timezone
	Adjustment string        `xml:"adjustment,attr,omitempty"` // Time adjustment
	Timers     []DomainTimer `xml:"timer,omitempty"`           // Individual timers
}

// DomainTimer represents a timer device
type DomainTimer struct {
	Name       string              `xml:"name,attr"`                 // platform, pit, rtc, hpet, tsc, hypervclock, kvmclock, etc.
	Track      string              `xml:"track,attr,omitempty"`      // boot, guest, wall
	TickPolicy string              `xml:"tickpolicy,attr,omitempty"` // delay, catchup, merge, discard
	CatchUp    *DomainTimerCatchUp `xml:"catchup,omitempty"`
	Frequency  uint64              `xml:"frequency,attr,omitempty"`
	Mode       string              `xml:"mode,attr,omitempty"`    // auto, native, emulate, paravirt, smpsafe
	Present    string              `xml:"present,attr,omitempty"` // yes, no
}

// DomainTimerCatchUp represents timer catchup configuration
type DomainTimerCatchUp struct {
	Threshold uint `xml:"threshold,attr,omitempty"`
	Slew      uint `xml:"slew,attr,omitempty"`
	Limit     uint `xml:"limit,attr,omitempty"`
}

// DomainController represents a device controller
// Source: https://libvirt.org/formatdomain.html#controllers
type DomainController struct {
	Type    string                  `xml:"type,attr"`            // usb, pci, scsi, ide, fdc, virtio-serial, ccid
	Index   int                     `xml:"index,attr"`           // Controller index
	Model   string                  `xml:"model,attr,omitempty"` // Controller model
	Driver  *DomainControllerDriver `xml:"driver,omitempty"`
	Master  *DomainControllerMaster `xml:"master,omitempty"`
	Address *DomainAddress          `xml:"address,omitempty"`
}

// DomainControllerDriver represents controller driver configuration
type DomainControllerDriver struct {
	Queues     int `xml:"queues,attr,omitempty"`
	CmdPerLun  int `xml:"cmd_per_lun,attr,omitempty"`
	MaxSectors int `xml:"max_sectors,attr,omitempty"`
	IOThread   int `xml:"iothread,attr,omitempty"`
}

// DomainControllerMaster represents USB companion controller
type DomainControllerMaster struct {
	StartPort int `xml:"startport,attr"`
}

// DomainAddress represents device address
type DomainAddress struct {
	Type          string `xml:"type,attr,omitempty"` // pci, drive, virtio-serial, ccid, usb, spapr-vio, ccw, isa, dimm
	Domain        string `xml:"domain,attr,omitempty"`
	Bus           string `xml:"bus,attr,omitempty"`
	Slot          string `xml:"slot,attr,omitempty"`
	Function      string `xml:"function,attr,omitempty"`
	Controller    int    `xml:"controller,attr,omitempty"`
	Target        int    `xml:"target,attr,omitempty"`
	Unit          int    `xml:"unit,attr,omitempty"`
	Port          int    `xml:"port,attr,omitempty"`
	MultiFunction string `xml:"multifunction,attr,omitempty"`
}

// DomainVideo represents a video device
// Source: https://libvirt.org/formatdomain.html#video-devices
type DomainVideo struct {
	Model   DomainVideoModel `xml:"model"`
	Address *DomainAddress   `xml:"address,omitempty"`
}

// DomainVideoModel represents video model configuration
type DomainVideoModel struct {
	Type         string                   `xml:"type,attr"`              // vga, cirrus, vmvga, qxl, virtio, gop, bochs, ramfb
	VRam         uint                     `xml:"vram,attr,omitempty"`    // Video RAM in KiB
	Heads        uint                     `xml:"heads,attr,omitempty"`   // Number of screens
	Primary      string                   `xml:"primary,attr,omitempty"` // yes, no
	Ram          uint                     `xml:"ram,attr,omitempty"`     // Bar size for QXL
	VGAMem       uint                     `xml:"vgamem,attr,omitempty"`  // VGA framebuffer size
	Acceleration *DomainVideoAcceleration `xml:"acceleration,omitempty"`
}

// DomainVideoAcceleration represents video acceleration
type DomainVideoAcceleration struct {
	Accel2D string `xml:"accel2d,attr,omitempty"` // yes, no
	Accel3D string `xml:"accel3d,attr,omitempty"` // yes, no
}

// DomainInput represents an input device
// Source: https://libvirt.org/formatdomain.html#input-devices
type DomainInput struct {
	Type    string             `xml:"type,attr"`            // mouse, tablet, keyboard, passthrough, evdev
	Bus     string             `xml:"bus,attr,omitempty"`   // ps2, usb, virtio, xen
	Model   string             `xml:"model,attr,omitempty"` // virtio, virtio-transitional, virtio-non-transitional
	Address *DomainAddress     `xml:"address,omitempty"`
	Source  *DomainInputSource `xml:"source,omitempty"`
}

// DomainInputSource represents input device source
type DomainInputSource struct {
	Dev string `xml:"dev,attr,omitempty"` // Device path for passthrough/evdev
}

// DomainSound represents a sound device
// Source: https://libvirt.org/formatdomain.html#sound-devices
type DomainSound struct {
	Model   string         `xml:"model,attr"` // es1370, sb16, ac97, ich6, ich9, usb
	Address *DomainAddress `xml:"address,omitempty"`
	Codec   []DomainCodec  `xml:"codec,omitempty"`
}

// DomainCodec represents audio codec configuration
type DomainCodec struct {
	Type string `xml:"type,attr"` // duplex, micro, output
}

// DomainHostdev represents host device passthrough
// Source: https://libvirt.org/formatdomain.html#host-device-assignment
type DomainHostdev struct {
	Mode    string               `xml:"mode,attr"`              // subsystem, capabilities
	Type    string               `xml:"type,attr"`              // usb, pci, scsi, scsi_host, mdev, storage, misc, net
	Managed string               `xml:"managed,attr,omitempty"` // yes, no
	Source  *DomainHostdevSource `xml:"source,omitempty"`
	Address *DomainAddress       `xml:"address,omitempty"`
	Boot    *DomainBootOrder     `xml:"boot,omitempty"`
	ROM     *DomainROM           `xml:"rom,omitempty"`
}

// DomainHostdevSource represents hostdev source
type DomainHostdevSource struct {
	Address  *DomainHostdevAddress `xml:"address,omitempty"`
	Vendor   *DomainHostdevVendor  `xml:"vendor,omitempty"`
	Product  *DomainHostdevProduct `xml:"product,omitempty"`
	Adapter  string                `xml:"adapter,attr,omitempty"`
	Protocol string                `xml:"protocol,attr,omitempty"`
}

// DomainHostdevAddress represents hostdev address
type DomainHostdevAddress struct {
	Domain   string `xml:"domain,attr,omitempty"`
	Bus      string `xml:"bus,attr,omitempty"`
	Slot     string `xml:"slot,attr,omitempty"`
	Function string `xml:"function,attr,omitempty"`
}

// DomainHostdevVendor represents USB vendor ID
type DomainHostdevVendor struct {
	ID string `xml:"id,attr"`
}

// DomainHostdevProduct represents USB product ID
type DomainHostdevProduct struct {
	ID string `xml:"id,attr"`
}

// DomainBootOrder represents boot order
type DomainBootOrder struct {
	Order int `xml:"order,attr"`
}

// DomainROM represents ROM configuration
type DomainROM struct {
	Bar     string `xml:"bar,attr,omitempty"`     // on, off
	File    string `xml:"file,attr,omitempty"`    // ROM file path
	Enabled string `xml:"enabled,attr,omitempty"` // yes, no
}

// DomainWatchdog represents a watchdog device
// Source: https://libvirt.org/formatdomain.html#watchdog-device
type DomainWatchdog struct {
	Model   string         `xml:"model,attr"`            // i6300esb, ib700, diag288
	Action  string         `xml:"action,attr,omitempty"` // reset, shutdown, poweroff, pause, none, dump, inject-nmi
	Address *DomainAddress `xml:"address,omitempty"`
}

// DomainChannel represents a channel device
// Source: https://libvirt.org/formatdomain.html#channel
type DomainChannel struct {
	Type    string               `xml:"type,attr"` // unix, pty, dev, file, pipe, tcp, udp, spicevmc, spiceport
	Source  *DomainChannelSource `xml:"source,omitempty"`
	Target  *DomainChannelTarget `xml:"target,omitempty"`
	Address *DomainAddress       `xml:"address,omitempty"`
}

// DomainChannelSource represents channel source
type DomainChannelSource struct {
	Mode string `xml:"mode,attr,omitempty"` // bind, connect
	Path string `xml:"path,attr,omitempty"` // Unix socket path
	Host string `xml:"host,attr,omitempty"` // Hostname for TCP
	Port string `xml:"port,attr,omitempty"` // Port for TCP/UDP
}

// DomainChannelTarget represents channel target
type DomainChannelTarget struct {
	Type    string `xml:"type,attr,omitempty"`    // virtio, guestfwd, xen
	Name    string `xml:"name,attr,omitempty"`    // Channel name (e.g., org.qemu.guest_agent.0)
	Address string `xml:"address,attr,omitempty"` // IP address for guestfwd
	Port    int    `xml:"port,attr,omitempty"`    // Port for guestfwd
}

// DomainMemBalloon represents memory balloon device
// Source: https://libvirt.org/formatdomain.html#memory-balloon-device
type DomainMemBalloon struct {
	Model       string                 `xml:"model,attr"`                 // virtio, xen, none
	AutoDeflate string                 `xml:"autodeflate,attr,omitempty"` // on, off
	Address     *DomainAddress         `xml:"address,omitempty"`
	Stats       *DomainMemBalloonStats `xml:"stats,omitempty"`
}

// DomainMemBalloonStats represents balloon statistics
type DomainMemBalloonStats struct {
	Period int `xml:"period,attr"` // Statistics collection period in seconds
}

// DomainRNG represents random number generator device
// Source: https://libvirt.org/formatdomain.html#random-number-generator-device
type DomainRNG struct {
	Model   string            `xml:"model,attr"` // virtio, virtio-transitional, virtio-non-transitional
	Backend *DomainRNGBackend `xml:"backend,omitempty"`
	Rate    *DomainRNGRate    `xml:"rate,omitempty"`
	Address *DomainAddress    `xml:"address,omitempty"`
}

// DomainRNGBackend represents RNG backend configuration
type DomainRNGBackend struct {
	Model   string               `xml:"model,attr"`          // random, egd, builtin
	Type    string               `xml:"type,attr,omitempty"` // For EGD backend (udp, tcp, etc.)
	Value   string               `xml:",chardata"`           // For random backend: device path like /dev/random
	Sources []DomainRNGEGDSource `xml:"source,omitempty"`    // For EGD backend: connection sources
}

// DomainRNGEGDSource represents EGD backend source configuration
type DomainRNGEGDSource struct {
	Mode    string `xml:"mode,attr"`              // bind, connect
	Host    string `xml:"host,attr,omitempty"`    // For connect mode
	Service string `xml:"service,attr,omitempty"` // Port number or service name
}

// DomainRNGRate represents RNG rate limiting
type DomainRNGRate struct {
	Bytes  int `xml:"bytes,attr"`  // Bytes per period
	Period int `xml:"period,attr"` // Period in milliseconds
}

// DomainTPM represents TPM device
// Source: https://libvirt.org/formatdomain.html#tpm-device
type DomainTPM struct {
	Model   string            `xml:"model,attr,omitempty"` // tpm-tis, tpm-crb, tpm-spapr
	Backend *DomainTPMBackend `xml:"backend,omitempty"`
}

// DomainTPMBackend represents TPM backend
type DomainTPMBackend struct {
	Type            string               `xml:"type,attr"`              // passthrough, emulator
	Version         string               `xml:"version,attr,omitempty"` // 1.2, 2.0
	Device          *DomainTPMDevice     `xml:"device,omitempty"`
	Encryption      *DomainTPMEncryption `xml:"encryption,omitempty"`
	PersistentState string               `xml:"persistent_state,attr,omitempty"` // yes, no
}

// DomainTPMDevice represents TPM device path
type DomainTPMDevice struct {
	Path string `xml:"path,attr"`
}

// DomainTPMEncryption represents TPM encryption
type DomainTPMEncryption struct {
	Secret *DomainSecret `xml:"secret,omitempty"`
}

// DomainSecret represents secret configuration
type DomainSecret struct {
	Type string `xml:"type,attr"` // passphrase, vtpm
	UUID string `xml:"uuid,attr,omitempty"`
}
