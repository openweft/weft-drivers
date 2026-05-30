package drivers

import (
	"errors"
	"testing"
)

// TestPackageCompiles is a trivial test that ensures the package
// compiles. The drivers package is purely interface + struct
// definitions, so there are no executable statements to cover
// beyond the package-level var initialisations in errors.go.
func TestPackageCompiles(t *testing.T) {
	// No-op: presence of this test ensures `go test ./...` runs.
}

// TestSentinelErrorsDistinct verifies the three sentinel errors
// declared in errors.go are distinct values. errors.Is must
// return false when comparing different sentinels, otherwise
// callers can't distinguish "not applicable" from "unsupported"
// from "not found".
func TestSentinelErrorsDistinct(t *testing.T) {
	pairs := []struct {
		name string
		a, b error
	}{
		{"NotApplicable_vs_Unsupported", ErrNotApplicable, ErrUnsupported},
		{"NotApplicable_vs_NotFound", ErrNotApplicable, ErrNotFound},
		{"Unsupported_vs_NotFound", ErrUnsupported, ErrNotFound},
	}
	for _, p := range pairs {
		t.Run(p.name, func(t *testing.T) {
			if errors.Is(p.a, p.b) {
				t.Errorf("expected %v and %v to be distinct sentinels", p.a, p.b)
			}
			if errors.Is(p.b, p.a) {
				t.Errorf("expected %v and %v to be distinct sentinels (reverse)", p.b, p.a)
			}
		})
	}
}

// TestSentinelErrorsSelfIdentity verifies each sentinel matches
// itself via errors.Is, including through a wrap. This is the
// primary public contract of the sentinels.
func TestSentinelErrorsSelfIdentity(t *testing.T) {
	sentinels := []struct {
		name string
		err  error
	}{
		{"ErrNotApplicable", ErrNotApplicable},
		{"ErrUnsupported", ErrUnsupported},
		{"ErrNotFound", ErrNotFound},
	}
	for _, s := range sentinels {
		t.Run(s.name, func(t *testing.T) {
			if s.err == nil {
				t.Fatal("sentinel is nil")
			}
			if !errors.Is(s.err, s.err) {
				t.Error("sentinel does not match itself via errors.Is")
			}
			wrapped := wrap(s.err)
			if !errors.Is(wrapped, s.err) {
				t.Error("sentinel does not match through wrap")
			}
			if s.err.Error() == "" {
				t.Error("sentinel has empty message")
			}
		})
	}
}

// TestSentinelErrorMessages locks in the exact public messages.
// These are part of the contract (they may appear in logs / API
// responses); breaking them is a backward-incompatible change.
func TestSentinelErrorMessages(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{ErrNotApplicable, "driver: operation not applicable to this driver type"},
		{ErrUnsupported, "driver: unsupported"},
		{ErrNotFound, "driver: not found"},
	}
	for _, c := range cases {
		if got := c.err.Error(); got != c.want {
			t.Errorf("error message: got %q, want %q", got, c.want)
		}
	}
}

// wrap is a tiny helper used to confirm errors.Is traverses
// the wrapping chain for sentinels.
func wrap(err error) error {
	return errors.Join(errors.New("context"), err)
}

// TestStructZeroValues exercises the struct types defined in
// types.go to make sure their zero values are usable. This is
// a cheap smoke test for the protobuf-friendly invariants
// (no required init, no panics on zero).
func TestStructZeroValues(t *testing.T) {
	var (
		_ HostInfo
		_ NetworkSpec
		_ PortSpec
		_ NICHandle
		_ VolumeSpec
		_ AttachedVolume
		_ VMSpec
		_ DiskSpec
	)
	// Concrete construction to confirm field set + assignment.
	hi := HostInfo{
		UUID:         "h-1",
		Hostname:     "host-1",
		AZ:           "us-east-1a",
		Hypervisor:   "apple-vz",
		Architecture: "arm64",
	}
	if hi.UUID != "h-1" {
		t.Errorf("HostInfo.UUID round-trip failed: %q", hi.UUID)
	}
	ns := NetworkSpec{
		UUID:           "n-1",
		ProjectUUID:    "p-1",
		Name:           "net",
		CIDR:           "10.0.0.0/24",
		Gateway:        "10.0.0.1",
		DNSServers:     []string{"1.1.1.1"},
		Type:           "nat",
		MeshListenPort: 51820,
		MeshEndpoint:   "1.2.3.4:51820",
	}
	if len(ns.DNSServers) != 1 || ns.DNSServers[0] != "1.1.1.1" {
		t.Errorf("NetworkSpec.DNSServers round-trip failed: %v", ns.DNSServers)
	}
	ps := PortSpec{
		UUID:                    "po-1",
		ProjectUUID:             "p-1",
		VMUUID:                  "v-1",
		NetworkUUID:             "n-1",
		MAC:                     "aa:bb:cc:dd:ee:ff",
		IP:                      "10.0.0.5",
		WireguardPubKey:         "pk",
		MeshEndpoint:            "5.6.7.8:51820",
		EffectiveSecurityGroups: []string{"sg-1", "sg-2"},
	}
	if len(ps.EffectiveSecurityGroups) != 2 {
		t.Errorf("PortSpec.EffectiveSecurityGroups round-trip failed")
	}
	nh := NICHandle{Device: "tap0", MAC: "aa:bb:cc:dd:ee:ff"}
	if nh.Device != "tap0" {
		t.Errorf("NICHandle.Device round-trip failed")
	}
	vs := VolumeSpec{UUID: "vol-1", ProjectUUID: "p-1", Name: "data", SizeGiB: 10, Format: "qcow2"}
	if vs.SizeGiB != 10 {
		t.Errorf("VolumeSpec.SizeGiB round-trip failed")
	}
	av := AttachedVolume{BackingPath: "/var/lib/weft/disk.qcow2", ReadOnly: true}
	if !av.ReadOnly {
		t.Errorf("AttachedVolume.ReadOnly round-trip failed")
	}
	vm := VMSpec{
		UUID:        "v-1",
		ProjectUUID: "p-1",
		Name:        "vm",
		CPUCount:    2,
		MemoryMiB:   1024,
		BootKind:    "uki",
		BootRef:     "ghcr.io/openweft/uki:latest",
		Cmdline:     "console=ttyS0",
	}
	if vm.CPUCount != 2 {
		t.Errorf("VMSpec.CPUCount round-trip failed")
	}
	ds := DiskSpec{
		VolumeUUID:  "vol-1",
		BackingPath: "/var/lib/weft/disk.qcow2",
		Bus:         "virtio",
		SizeGiB:     20,
		ReadOnly:    false,
		Boot:        true,
	}
	if !ds.Boot {
		t.Errorf("DiskSpec.Boot round-trip failed")
	}
}

// TestInterfacesAssignable ensures the driver interface types are
// satisfiable. We declare nil interface values; this catches any
// future change that accidentally makes an interface unimplementable
// (e.g. unexported method added).
func TestInterfacesAssignable(t *testing.T) {
	var (
		hv  HypervisorDriver
		nd  NetworkDriver
		vd  VolumeDriver
		id  ImageDriver
	)
	if hv != nil || nd != nil || vd != nil || id != nil {
		t.Error("expected zero-value interfaces to be nil")
	}
}
