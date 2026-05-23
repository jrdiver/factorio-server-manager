package factorio

import (
	"encoding/binary"
	"fmt"
	"os"
	"testing"
)

// readSave2x opens a level-init.dat directly (not from a zip) and parses the header.
func readSave2x(t *testing.T, path string) SaveHeader {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Cannot open %s: %v", path, err)
	}
	defer f.Close()
	var h SaveHeader
	if err := h.ReadFrom(f); err != nil {
		t.Fatalf("ReadFrom failed for %s: %v", path, err)
	}
	return h
}

// modNames returns the set of mod names from a SaveHeader.
func modNames(h SaveHeader) map[string]bool {
	out := make(map[string]bool, len(h.Mods))
	for _, m := range h.Mods {
		out[m.Name] = true
	}
	return out
}

// Test2x_AgesOfSpace: vanilla + Space Age DLC, expected 4 built-in mods.
func Test2x_AgesOfSpace(t *testing.T) {
	h := readSave2x(t, "../../The Ages of Space 1.275/level-init.dat")

	if h.FactorioVersion[0] != 2 {
		t.Errorf("expected Factorio 2.x, got %s", h.FactorioVersion)
	}
	wantMods := []string{"base", "elevated-rails", "quality", "space-age"}
	if len(h.Mods) != len(wantMods) {
		t.Errorf("expected %d mods, got %d", len(wantMods), len(h.Mods))
	}
	names := modNames(h)
	for _, name := range wantMods {
		if !names[name] {
			t.Errorf("expected mod %q not found in %v", name, names)
		}
	}
}

// Test2x_ModDev1: 5-mod save (base DLC + 1 extra).
func Test2x_ModDev1(t *testing.T) {
	h := readSave2x(t, "../../ModDev1/level-init.dat")

	if h.FactorioVersion[0] != 2 {
		t.Errorf("expected Factorio 2.x, got %s", h.FactorioVersion)
	}
	if len(h.Mods) != 5 {
		t.Errorf("expected 5 mods, got %d: %v", len(h.Mods), modNames(h))
	}
	if !modNames(h)["base"] {
		t.Error("expected mod \"base\" in ModDev1 save")
	}
}

// Test2x_CamPackSMP: large modpack with 183 mods.
func Test2x_CamPackSMP(t *testing.T) {
	h := readSave2x(t, "../../CamPackSMP25.2/level-init.dat")

	if h.FactorioVersion[0] != 2 {
		t.Errorf("expected Factorio 2.x, got %s", h.FactorioVersion)
	}
	if len(h.Mods) != 183 {
		t.Errorf("expected 183 mods, got %d", len(h.Mods))
	}
	if !modNames(h)["base"] {
		t.Error("expected mod \"base\" in CamPackSMP save")
	}
}

// probeHeader reads a level-init.dat up through AllowedCommands using the same
// logic as ReadFrom, then dumps the next 32 raw bytes.  This lets us spot any
// unknown fields that Factorio 2.x inserted before the mod-count.
func probeHeader(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Cannot read %s: %v", path, err)
	}

	pos := 0
	read := func(n int) []byte {
		b := data[pos : pos+n]
		pos += n
		return b
	}
	readOptimUint16 := func() uint16 {
		b := data[pos]
		pos++
		if b != 0xFF {
			return uint16(b)
		}
		v := binary.LittleEndian.Uint16(data[pos : pos+2])
		pos += 2
		return v
	}
	readOptimUint32 := func() uint32 {
		b := data[pos]
		pos++
		if b != 0xFF {
			return uint32(b)
		}
		v := binary.LittleEndian.Uint32(data[pos : pos+4])
		pos += 4
		return v
	}
	readOptimStr := func() string {
		n := readOptimUint32()
		s := string(data[pos : pos+int(n)])
		pos += int(n)
		return s
	}

	ver := [4]uint16{
		binary.LittleEndian.Uint16(read(2)),
		binary.LittleEndian.Uint16(read(2)),
		binary.LittleEndian.Uint16(read(2)),
		binary.LittleEndian.Uint16(read(2)),
	}
	fmt.Printf("Version: %d.%d.%d.%d\n", ver[0], ver[1], ver[2], ver[3])
	fmt.Printf("0.17 byte:               0x%02X\n", read(1)[0])
	fmt.Printf("Campaign:                %q\n", readOptimStr())
	fmt.Printf("Name:                    %q\n", readOptimStr())
	fmt.Printf("BaseMod:                 %q\n", readOptimStr())
	fmt.Printf("Difficulty:              %d\n", read(1)[0])
	fmt.Printf("Finished:                %d\n", read(1)[0])
	fmt.Printf("PlayerWon:               %d\n", read(1)[0])
	fmt.Printf("NextLevel:               %q\n", readOptimStr())
	fmt.Printf("CanContinue:             %d\n", read(1)[0])
	fmt.Printf("FinishedButContinuing:   %d\n", read(1)[0])
	fmt.Printf("SavingReplay:            %d\n", read(1)[0])
	fmt.Printf("AllowNonAdminDebug:      %d\n", read(1)[0])
	lf0 := readOptimUint16()
	lf1 := readOptimUint16()
	lf2 := readOptimUint16()
	fmt.Printf("LoadedFrom:              %d.%d.%d\n", lf0, lf1, lf2)
	fmt.Printf("LoadedFromBuild:         %d\n", binary.LittleEndian.Uint16(read(2)))
	fmt.Printf("AllowedCommands:         %d\n", read(1)[0])

	fmt.Printf("\n=== Unknown bytes after AllowedCommands (offset %d, next 32 bytes) ===\n", pos)
	for i := 0; i < 32 && pos+i < len(data); i++ {
		fmt.Printf("  [%d] 0x%02X  %3d  %q\n", pos+i, data[pos+i], data[pos+i], rune(data[pos+i]))
	}
	fmt.Println()
}

// Byte-level probe: reads manually up to AllowedCommands then dumps the next
// 32 bytes so we can identify unknown 2.0 fields before the mod list.
func Test2x_ByteProbe(t *testing.T) {
	t.Run("CamPackSMP25.2", func(t *testing.T) {
		probeHeader(t, "../../CamPackSMP25.2/level-init.dat")
	})
	t.Run("AgesOfSpace", func(t *testing.T) {
		probeHeader(t, "../../The Ages of Space 1.275/level-init.dat")
	})
	t.Run("ModDev1", func(t *testing.T) {
		probeHeader(t, "../../ModDev1/level-init.dat")
	})
	t.Run("2.0NoMod", func(t *testing.T) {
		probeHeader(t, "../../2.0NoMod/level-init.dat")
	})
}

// Test2x_NoMod: base-only 2.0 save with no DLC, expected exactly 1 mod.
func Test2x_NoMod(t *testing.T) {
	h := readSave2x(t, "../../2.0NoMod/level-init.dat")

	if h.FactorioVersion[0] != 2 {
		t.Errorf("expected Factorio 2.x, got %s", h.FactorioVersion)
	}
	names := modNames(h)
	if !names["base"] {
		t.Errorf("expected mod \"base\" in 2.0NoMod save, got %v", names)
	}
	// Without Space Age DLC the only mod should be base.
	if len(h.Mods) != 1 {
		t.Errorf("expected 1 mod (base only), got %d: %v", len(h.Mods), names)
	}
}
