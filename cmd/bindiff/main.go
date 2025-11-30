package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
)

const (
	MH_MAGIC_64    = 0xfeedfacf
	LC_SEGMENT_64  = 0x19
	LC_SYMTAB      = 0x2
	LC_DYSYMTAB    = 0xb
	LC_LOAD_DYLINKER = 0xe
	LC_LOAD_DYLIB  = 0xc
	LC_UUID        = 0x1b
	LC_BUILD_VERSION = 0x32
	LC_SOURCE_VERSION = 0x2a
	LC_CODE_SIGNATURE = 0x1d
	LC_MAIN        = 0x80000028
	LC_DYLD_CHAINED_FIXUPS = 0x80000034
	LC_DYLD_INFO_ONLY = 0x80000022
	LC_FUNCTION_STARTS = 0x26
	LC_DATA_IN_CODE = 0x29
)

var lcNames = map[uint32]string{
	LC_SEGMENT_64: "LC_SEGMENT_64",
	LC_SYMTAB: "LC_SYMTAB",
	LC_DYSYMTAB: "LC_DYSYMTAB",
	LC_LOAD_DYLINKER: "LC_LOAD_DYLINKER",
	LC_LOAD_DYLIB: "LC_LOAD_DYLIB",
	LC_UUID: "LC_UUID",
	LC_BUILD_VERSION: "LC_BUILD_VERSION",
	LC_SOURCE_VERSION: "LC_SOURCE_VERSION",
	LC_CODE_SIGNATURE: "LC_CODE_SIGNATURE",
	LC_MAIN: "LC_MAIN",
	LC_DYLD_CHAINED_FIXUPS: "LC_DYLD_CHAINED_FIXUPS",
	LC_DYLD_INFO_ONLY: "LC_DYLD_INFO_ONLY",
	LC_FUNCTION_STARTS: "LC_FUNCTION_STARTS",
	LC_DATA_IN_CODE: "LC_DATA_IN_CODE",
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <file1> <file2>\n", os.Args[0])
		os.Exit(1)
	}

	file1Path := os.Args[1]
	file2Path := os.Args[2]

	data1, err := os.ReadFile(file1Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", file1Path, err)
		os.Exit(1)
	}

	data2, err := os.ReadFile(file2Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", file2Path, err)
		os.Exit(1)
	}

	// Parse and compare Mach-O structure
	fmt.Println("=" + string(make([]byte, 60)) + "=")
	fmt.Println("MACH-O STRUCTURE COMPARISON")
	fmt.Println("=" + string(make([]byte, 60)) + "=")
	fmt.Println()
	compareMachO(file1Path, file2Path, data1, data2)

	fmt.Println()
	compare(file1Path, file2Path, data1, data2)
}

func compareMachO(name1, name2 string, data1, data2 []byte) {
	m1, err1 := parseMachO(data1)
	m2, err2 := parseMachO(data2)

	if err1 != nil {
		fmt.Printf("File 1 (%s): Not a valid Mach-O: %v\n", name1, err1)
		return
	}
	if err2 != nil {
		fmt.Printf("File 2 (%s): Not a valid Mach-O: %v\n", name2, err2)
		return
	}

	fmt.Printf("%-35s | %-35s\n", "FILE 1", "FILE 2")
	fmt.Printf("%-35s | %-35s\n", name1, name2)
	fmt.Println(string(make([]byte, 73)))

	// Header comparison
	fmt.Printf("\nMACH-O HEADER:\n")
	fmt.Printf("  NCmds:      %-22d | %-22d %s\n", m1.ncmds, m2.ncmds, diffMark(m1.ncmds, m2.ncmds))
	fmt.Printf("  SizeofCmds: %-22d | %-22d %s\n", m1.sizeofcmds, m2.sizeofcmds, diffMark(m1.sizeofcmds, m2.sizeofcmds))
	fmt.Printf("  Flags:      %-22s | %-22s %s\n", fmt.Sprintf("0x%08x", m1.flags), fmt.Sprintf("0x%08x", m2.flags), diffMark(m1.flags, m2.flags))

	// Load commands comparison
	fmt.Printf("\nLOAD COMMANDS:\n")
	fmt.Printf("  %-30s | %-30s\n", fmt.Sprintf("Count: %d", len(m1.loadCmds)), fmt.Sprintf("Count: %d", len(m2.loadCmds)))

	// Build a map of load commands by type for comparison
	lc1 := make(map[uint32][]loadCmd)
	lc2 := make(map[uint32][]loadCmd)
	for _, lc := range m1.loadCmds {
		lc1[lc.cmd] = append(lc1[lc.cmd], lc)
	}
	for _, lc := range m2.loadCmds {
		lc2[lc.cmd] = append(lc2[lc.cmd], lc)
	}

	// Show which commands are present/missing
	allCmds := make(map[uint32]bool)
	for cmd := range lc1 { allCmds[cmd] = true }
	for cmd := range lc2 { allCmds[cmd] = true }

	for cmd := range allCmds {
		name := lcNames[cmd]
		if name == "" {
			name = fmt.Sprintf("0x%08x", cmd)
		}
		has1 := len(lc1[cmd]) > 0
		has2 := len(lc2[cmd]) > 0

		mark := ""
		if has1 != has2 {
			mark = " <-- MISSING"
		}

		str1 := "-"
		str2 := "-"
		if has1 {
			str1 = "✓"
		}
		if has2 {
			str2 = "✓"
		}
		fmt.Printf("  %-28s %-3s | %-3s%s\n", name+":", str1, str2, mark)
	}

	// Segment comparison
	fmt.Printf("\nSEGMENTS:\n")
	for i := 0; i < max(len(m1.segments), len(m2.segments)); i++ {
		var s1, s2 *segment
		if i < len(m1.segments) {
			s1 = &m1.segments[i]
		}
		if i < len(m2.segments) {
			s2 = &m2.segments[i]
		}

		name1 := "-"
		name2 := "-"
		if s1 != nil { name1 = s1.name }
		if s2 != nil { name2 = s2.name }

		fmt.Printf("\n  [%d] %-12s vs %-12s\n", i, name1, name2)

		if s1 != nil && s2 != nil && s1.name == s2.name {
			fmt.Printf("      vmaddr:   0x%-18x | 0x%-18x %s\n", s1.vmaddr, s2.vmaddr, diffMark(s1.vmaddr, s2.vmaddr))
			fmt.Printf("      vmsize:   0x%-18x | 0x%-18x %s\n", s1.vmsize, s2.vmsize, diffMark(s1.vmsize, s2.vmsize))
			fmt.Printf("      fileoff:  0x%-18x | 0x%-18x %s\n", s1.fileoff, s2.fileoff, diffMark(s1.fileoff, s2.fileoff))
			fmt.Printf("      filesize: 0x%-18x | 0x%-18x %s\n", s1.filesize, s2.filesize, diffMark(s1.filesize, s2.filesize))
			fmt.Printf("      maxprot:  0x%-18x | 0x%-18x %s\n", s1.maxprot, s2.maxprot, diffMark(uint64(s1.maxprot), uint64(s2.maxprot)))
			fmt.Printf("      initprot: 0x%-18x | 0x%-18x %s\n", s1.initprot, s2.initprot, diffMark(uint64(s1.initprot), uint64(s2.initprot)))

			// Sections
			for j := 0; j < max(len(s1.sections), len(s2.sections)); j++ {
				var sec1, sec2 *section
				if j < len(s1.sections) { sec1 = &s1.sections[j] }
				if j < len(s2.sections) { sec2 = &s2.sections[j] }

				secName1 := "-"
				secName2 := "-"
				if sec1 != nil { secName1 = sec1.name }
				if sec2 != nil { secName2 = sec2.name }

				fmt.Printf("      Section: %-15s vs %-15s\n", secName1, secName2)
				if sec1 != nil && sec2 != nil {
					fmt.Printf("        addr:   0x%-16x | 0x%-16x %s\n", sec1.addr, sec2.addr, diffMark(sec1.addr, sec2.addr))
					fmt.Printf("        size:   0x%-16x | 0x%-16x %s\n", sec1.size, sec2.size, diffMark(sec1.size, sec2.size))
					fmt.Printf("        offset: 0x%-16x | 0x%-16x %s\n", sec1.offset, sec2.offset, diffMark(uint64(sec1.offset), uint64(sec2.offset)))
					fmt.Printf("        align:  %-18d | %-18d %s\n", sec1.align, sec2.align, diffMark(uint64(sec1.align), uint64(sec2.align)))
				}
			}
		}
	}

	// Entry point
	fmt.Printf("\nENTRY POINT:\n")
	fmt.Printf("  entryoff:   0x%-18x | 0x%-18x %s\n", m1.entryoff, m2.entryoff, diffMark(m1.entryoff, m2.entryoff))
}

func diffMark[T comparable](a, b T) string {
	if a != b {
		return "<-- DIFF"
	}
	return ""
}

type machO struct {
	ncmds      uint32
	sizeofcmds uint32
	flags      uint32
	loadCmds   []loadCmd
	segments   []segment
	entryoff   uint64
}

type loadCmd struct {
	cmd     uint32
	cmdsize uint32
}

type segment struct {
	name     string
	vmaddr   uint64
	vmsize   uint64
	fileoff  uint64
	filesize uint64
	maxprot  uint32
	initprot uint32
	sections []section
}

type section struct {
	name   string
	addr   uint64
	size   uint64
	offset uint32
	align  uint32
}

func parseMachO(data []byte) (*machO, error) {
	if len(data) < 32 {
		return nil, fmt.Errorf("file too small")
	}

	magic := binary.LittleEndian.Uint32(data[0:4])
	if magic != MH_MAGIC_64 {
		return nil, fmt.Errorf("not a Mach-O 64-bit file")
	}

	m := &machO{
		ncmds:      binary.LittleEndian.Uint32(data[16:20]),
		sizeofcmds: binary.LittleEndian.Uint32(data[20:24]),
		flags:      binary.LittleEndian.Uint32(data[24:28]),
	}

	// Parse load commands
	offset := uint32(32) // After mach_header_64
	for i := uint32(0); i < m.ncmds && int(offset) < len(data); i++ {
		if int(offset)+8 > len(data) {
			break
		}

		cmd := binary.LittleEndian.Uint32(data[offset : offset+4])
		cmdsize := binary.LittleEndian.Uint32(data[offset+4 : offset+8])

		m.loadCmds = append(m.loadCmds, loadCmd{cmd: cmd, cmdsize: cmdsize})

		// Parse segment details
		if cmd == LC_SEGMENT_64 && int(offset)+72 <= len(data) {
			seg := segment{
				name:     string(bytes.TrimRight(data[offset+8:offset+24], "\x00")),
				vmaddr:   binary.LittleEndian.Uint64(data[offset+24 : offset+32]),
				vmsize:   binary.LittleEndian.Uint64(data[offset+32 : offset+40]),
				fileoff:  binary.LittleEndian.Uint64(data[offset+40 : offset+48]),
				filesize: binary.LittleEndian.Uint64(data[offset+48 : offset+56]),
				maxprot:  binary.LittleEndian.Uint32(data[offset+56 : offset+60]),
				initprot: binary.LittleEndian.Uint32(data[offset+60 : offset+64]),
			}

			nsects := binary.LittleEndian.Uint32(data[offset+64 : offset+68])

			// Parse sections
			secOffset := offset + 72
			for s := uint32(0); s < nsects && int(secOffset)+80 <= len(data); s++ {
				sec := section{
					name:   string(bytes.TrimRight(data[secOffset:secOffset+16], "\x00")),
					addr:   binary.LittleEndian.Uint64(data[secOffset+32 : secOffset+40]),
					size:   binary.LittleEndian.Uint64(data[secOffset+40 : secOffset+48]),
					offset: binary.LittleEndian.Uint32(data[secOffset+48 : secOffset+52]),
					align:  binary.LittleEndian.Uint32(data[secOffset+52 : secOffset+56]),
				}
				seg.sections = append(seg.sections, sec)
				secOffset += 80
			}

			m.segments = append(m.segments, seg)
		}

		// Parse LC_MAIN
		if cmd == LC_MAIN && int(offset)+24 <= len(data) {
			m.entryoff = binary.LittleEndian.Uint64(data[offset+8 : offset+16])
		}

		offset += cmdsize
	}

	return m, nil
}

func compare(name1, name2 string, data1, data2 []byte) {
	fmt.Println("Binary Comparison Summary")
	fmt.Println("=========================")
	fmt.Printf("File 1: %s (%d bytes)\n", name1, len(data1))
	fmt.Printf("File 2: %s (%d bytes)\n", name2, len(data2))
	fmt.Println()

	if bytes.Equal(data1, data2) {
		fmt.Println("Result: Files are identical")
		return
	}

	// Size difference
	sizeDiff := len(data2) - len(data1)
	if sizeDiff != 0 {
		if sizeDiff > 0 {
			fmt.Printf("Size difference: File 2 is %d bytes larger\n", sizeDiff)
		} else {
			fmt.Printf("Size difference: File 1 is %d bytes larger\n", -sizeDiff)
		}
	} else {
		fmt.Println("Size difference: None (same size)")
	}

	// Find differences
	minLen := min(len(data1), len(data2))
	maxLen := max(len(data1), len(data2))

	var diffCount int
	var firstDiff int = -1
	var lastDiff int = -1
	diffRegions := []struct{ start, end int }{}
	inRegion := false
	regionStart := 0

	for i := 0; i < minLen; i++ {
		if data1[i] != data2[i] {
			diffCount++
			if firstDiff == -1 {
				firstDiff = i
			}
			lastDiff = i
			if !inRegion {
				inRegion = true
				regionStart = i
			}
		} else {
			if inRegion {
				diffRegions = append(diffRegions, struct{ start, end int }{regionStart, i})
				inRegion = false
			}
		}
	}
	if inRegion {
		diffRegions = append(diffRegions, struct{ start, end int }{regionStart, minLen})
	}

	// Count bytes only in longer file
	if maxLen > minLen {
		diffCount += maxLen - minLen
		if firstDiff == -1 {
			firstDiff = minLen
		}
		lastDiff = maxLen - 1
	}

	fmt.Printf("Differing bytes: %d (%.2f%%)\n", diffCount, float64(diffCount)*100/float64(maxLen))
	fmt.Printf("First difference at offset: 0x%X (%d)\n", firstDiff, firstDiff)
	fmt.Printf("Last difference at offset: 0x%X (%d)\n", lastDiff, lastDiff)
	fmt.Printf("Contiguous diff regions: %d\n", len(diffRegions))
	fmt.Println()

	// Show first few diff regions with hex dump
	maxRegions := 5
	if len(diffRegions) > 0 {
		fmt.Println("Difference Details (first few regions):")
		fmt.Println("-----------------------------------------")

		for i, region := range diffRegions {
			if i >= maxRegions {
				fmt.Printf("... and %d more regions\n", len(diffRegions)-maxRegions)
				break
			}

			regionLen := region.end - region.start
			fmt.Printf("\nRegion %d: offset 0x%X-0x%X (%d bytes)\n", i+1, region.start, region.end-1, regionLen)

			// Show context around the diff (up to 16 bytes before and after)
			contextStart := max(0, region.start-16)
			contextEnd := min(minLen, region.end+16)
			showLen := min(64, contextEnd-contextStart) // Limit display

			fmt.Printf("  File 1: %s\n", formatHexLine(data1[contextStart:contextStart+showLen], contextStart, region.start, region.end))
			fmt.Printf("  File 2: %s\n", formatHexLine(data2[contextStart:contextStart+showLen], contextStart, region.start, region.end))
		}
	}

	// If one file is longer, show the extra bytes
	if len(data1) != len(data2) {
		fmt.Println()
		if len(data1) > len(data2) {
			extraBytes := data1[len(data2):]
			fmt.Printf("Extra bytes in File 1 (offset 0x%X, %d bytes):\n", len(data2), len(extraBytes))
			printHexDump(extraBytes, len(data2), 64)
		} else {
			extraBytes := data2[len(data1):]
			fmt.Printf("Extra bytes in File 2 (offset 0x%X, %d bytes):\n", len(data1), len(extraBytes))
			printHexDump(extraBytes, len(data1), 64)
		}
	}
}

func formatHexLine(data []byte, baseOffset, diffStart, diffEnd int) string {
	var result bytes.Buffer
	for i, b := range data {
		offset := baseOffset + i
		if offset >= diffStart && offset < diffEnd {
			// Highlight differences with brackets
			result.WriteString(fmt.Sprintf("[%02X]", b))
		} else {
			result.WriteString(fmt.Sprintf(" %02X ", b))
		}
	}
	return result.String()
}

func printHexDump(data []byte, baseOffset int, maxBytes int) {
	if len(data) > maxBytes {
		data = data[:maxBytes]
		defer fmt.Printf("  ... (%d more bytes)\n", len(data)-maxBytes)
	}

	for i := 0; i < len(data); i += 16 {
		end := min(i+16, len(data))
		line := data[i:end]

		// Offset
		fmt.Printf("  %08X: ", baseOffset+i)

		// Hex
		fmt.Print(hex.EncodeToString(line))
		// Padding if short line
		if len(line) < 16 {
			for j := len(line); j < 16; j++ {
				fmt.Print("  ")
			}
		}

		// ASCII
		fmt.Print("  |")
		for _, b := range line {
			if b >= 32 && b < 127 {
				fmt.Printf("%c", b)
			} else {
				fmt.Print(".")
			}
		}
		fmt.Println("|")
	}
}
