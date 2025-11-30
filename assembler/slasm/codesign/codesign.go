// Package codesign provides ad-hoc code signing for Mach-O files.
//
// This package implements the same ad-hoc signing algorithm used by the
// Darwin linker and Go toolchain, allowing slasm to sign binaries without
// relying on the external codesign tool.
package codesign

import (
	"crypto/sha256"
	"debug/macho"
	"encoding/binary"
	"io"
)

// Page size constants for code signing
const (
	pageSizeBits = 12
	pageSize     = 1 << pageSizeBits // 4096 bytes
)

// Hash constants
const (
	hashSize = 32 // SHA256 hash size
)

// LC_CODE_SIGNATURE load command type
const LC_CODE_SIGNATURE = 0x1d

// Code signature magic numbers
// From https://opensource.apple.com/source/xnu/xnu-4903.270.47/osfmk/kern/cs_blobs.h
const (
	CSMAGIC_REQUIREMENT        = 0xfade0c00 // single Requirement blob
	CSMAGIC_REQUIREMENTS       = 0xfade0c01 // Requirements vector
	CSMAGIC_CODEDIRECTORY      = 0xfade0c02 // CodeDirectory blob
	CSMAGIC_EMBEDDED_SIGNATURE = 0xfade0cc0 // embedded signature
	CSMAGIC_DETACHED_SIGNATURE = 0xfade0cc1 // detached signature

	CSSLOT_CODEDIRECTORY = 0 // slot index for CodeDirectory
)

// Hash type constants
const (
	CS_HASHTYPE_SHA1             = 1
	CS_HASHTYPE_SHA256           = 2
	CS_HASHTYPE_SHA256_TRUNCATED = 3
	CS_HASHTYPE_SHA384           = 4
)

// Executable segment flags
const (
	CS_EXECSEG_MAIN_BINARY     = 0x1   // executable segment denotes main binary
	CS_EXECSEG_ALLOW_UNSIGNED  = 0x10  // allow unsigned pages (debugging)
	CS_EXECSEG_DEBUGGER        = 0x20  // main binary is debugger
	CS_EXECSEG_JIT             = 0x40  // JIT enabled
	CS_EXECSEG_SKIP_LV         = 0x80  // skip library validation
	CS_EXECSEG_CAN_LOAD_CDHASH = 0x100 // can bless cdhash for execution
	CS_EXECSEG_CAN_EXEC_CDHASH = 0x200 // can execute blessed cdhash
)

// Blob represents an index entry in the SuperBlob
type Blob struct {
	typ    uint32 // type of entry
	offset uint32 // offset of entry from start of SuperBlob
}

func (b *Blob) put(out []byte) []byte {
	out = put32be(out, b.typ)
	out = put32be(out, b.offset)
	return out
}

const blobSize = 2 * 4 // 8 bytes

// SuperBlob is the container for code signature blobs
type SuperBlob struct {
	magic  uint32 // magic number
	length uint32 // total length of SuperBlob
	count  uint32 // number of index entries following
}

func (s *SuperBlob) put(out []byte) []byte {
	out = put32be(out, s.magic)
	out = put32be(out, s.length)
	out = put32be(out, s.count)
	return out
}

const superBlobSize = 3 * 4 // 12 bytes

// CodeDirectory contains the code signature metadata and hashes
type CodeDirectory struct {
	magic         uint32 // magic number (CSMAGIC_CODEDIRECTORY)
	length        uint32 // total length of CodeDirectory blob
	version       uint32 // compatibility version
	flags         uint32 // setup and mode flags
	hashOffset    uint32 // offset of hash slot element at index zero
	identOffset   uint32 // offset of identifier string
	nSpecialSlots uint32 // number of special hash slots
	nCodeSlots    uint32 // number of ordinary (code) hash slots
	codeLimit     uint32 // limit to main image signature range
	hashSize      uint8  // size of each hash in bytes
	hashType      uint8  // type of hash (CS_HASHTYPE_* constants)
	_pad1         uint8  // unused (must be zero)
	pageSize      uint8  // log2(page size in bytes); 0 => infinite
	_pad2         uint32 // unused (must be zero)
	scatterOffset uint32 // offset of scatter vector
	teamOffset    uint32 // offset of team identifier
	_pad3         uint32 // unused (must be zero)
	codeLimit64   uint64 // limit to main image signature range (64-bit)
	execSegBase   uint64 // offset of executable segment
	execSegLimit  uint64 // limit of executable segment
	execSegFlags  uint64 // executable segment flags
}

func (c *CodeDirectory) put(out []byte) []byte {
	out = put32be(out, c.magic)
	out = put32be(out, c.length)
	out = put32be(out, c.version)
	out = put32be(out, c.flags)
	out = put32be(out, c.hashOffset)
	out = put32be(out, c.identOffset)
	out = put32be(out, c.nSpecialSlots)
	out = put32be(out, c.nCodeSlots)
	out = put32be(out, c.codeLimit)
	out = put8(out, c.hashSize)
	out = put8(out, c.hashType)
	out = put8(out, c._pad1)
	out = put8(out, c.pageSize)
	out = put32be(out, c._pad2)
	out = put32be(out, c.scatterOffset)
	out = put32be(out, c.teamOffset)
	out = put32be(out, c._pad3)
	out = put64be(out, c.codeLimit64)
	out = put64be(out, c.execSegBase)
	out = put64be(out, c.execSegLimit)
	out = put64be(out, c.execSegFlags)
	return out
}

const codeDirectorySize = 13*4 + 4 + 4*8 // 88 bytes

// CodeSigCmd represents the LC_CODE_SIGNATURE load command
type CodeSigCmd struct {
	Cmd      uint32 // LC_CODE_SIGNATURE
	Cmdsize  uint32 // sizeof this command (16)
	Dataoff  uint32 // file offset of signature data in __LINKEDIT
	Datasize uint32 // file size of signature data
}

// FindCodeSigCmd searches for LC_CODE_SIGNATURE in a Mach-O file
func FindCodeSigCmd(f *macho.File) (CodeSigCmd, bool) {
	get32 := f.ByteOrder.Uint32
	for _, l := range f.Loads {
		data := l.Raw()
		cmd := get32(data)
		if cmd == LC_CODE_SIGNATURE {
			return CodeSigCmd{
				Cmd:      cmd,
				Cmdsize:  get32(data[4:]),
				Dataoff:  get32(data[8:]),
				Datasize: get32(data[12:]),
			}, true
		}
	}
	return CodeSigCmd{}, false
}

// Size computes the size of the code signature.
// id is the identifier used for signing (a field in CodeDirectory blob,
// which has no significance in ad-hoc signing).
func Size(codeSize int64, id string) int64 {
	nhashes := (codeSize + pageSize - 1) / pageSize
	idOff := int64(codeDirectorySize)
	hashOff := idOff + int64(len(id)+1) // +1 for null terminator
	cdirSz := hashOff + nhashes*hashSize
	return int64(superBlobSize+blobSize) + cdirSz
}

// Sign generates an ad-hoc code signature and writes it to out.
// out must have length at least Size(codeSize, id).
// data is the file content without the signature, of size codeSize.
// textOff and textSize are the file offset and size of the text segment.
// isMain is true if this is a main executable.
// id is the identifier used for signing (a field in CodeDirectory blob,
// which has no significance in ad-hoc signing).
func Sign(out []byte, data io.Reader, id string, codeSize, textOff, textSize int64, isMain bool) {
	nhashes := (codeSize + pageSize - 1) / pageSize
	idOff := int64(codeDirectorySize)
	hashOff := idOff + int64(len(id)+1)
	sz := Size(codeSize, id)

	// Emit SuperBlob header
	sb := SuperBlob{
		magic:  CSMAGIC_EMBEDDED_SIGNATURE,
		length: uint32(sz),
		count:  1,
	}

	// Emit Blob index entry
	blob := Blob{
		typ:    CSSLOT_CODEDIRECTORY,
		offset: superBlobSize + blobSize,
	}

	// Emit CodeDirectory
	cdir := CodeDirectory{
		magic:        CSMAGIC_CODEDIRECTORY,
		length:       uint32(sz) - (superBlobSize + blobSize),
		version:      0x20400,
		flags:        0x20002, // adhoc | linkerSigned
		hashOffset:   uint32(hashOff),
		identOffset:  uint32(idOff),
		nCodeSlots:   uint32(nhashes),
		codeLimit:    uint32(codeSize),
		hashSize:     hashSize,
		hashType:     CS_HASHTYPE_SHA256,
		pageSize:     uint8(pageSizeBits),
		execSegBase:  uint64(textOff),
		execSegLimit: uint64(textSize),
	}
	if isMain {
		cdir.execSegFlags = CS_EXECSEG_MAIN_BINARY
	}

	// Write headers
	outp := out
	outp = sb.put(outp)
	outp = blob.put(outp)
	outp = cdir.put(outp)

	// Write the identifier (null-terminated C string)
	outp = puts(outp, []byte(id+"\000"))

	// Hash each page and write the hashes
	var buf [pageSize]byte
	p := 0
	for p < int(codeSize) {
		n, err := io.ReadFull(data, buf[:])
		if err == io.EOF {
			break
		}
		if err != nil && err != io.ErrUnexpectedEOF {
			panic(err)
		}
		if p+n > int(codeSize) {
			n = int(codeSize) - p
		}
		p += n
		h := sha256.Sum256(buf[:n])
		outp = puts(outp, h[:])
	}
}

// Helper functions for big-endian serialization

func put32be(b []byte, x uint32) []byte {
	binary.BigEndian.PutUint32(b, x)
	return b[4:]
}

func put64be(b []byte, x uint64) []byte {
	binary.BigEndian.PutUint64(b, x)
	return b[8:]
}

func put8(b []byte, x uint8) []byte {
	b[0] = x
	return b[1:]
}

func puts(b, s []byte) []byte {
	n := copy(b, s)
	return b[n:]
}
