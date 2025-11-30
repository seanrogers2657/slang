package codesign

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestSize(t *testing.T) {
	tests := []struct {
		name     string
		codeSize int64
		id       string
		wantMin  int64 // minimum expected size
	}{
		{
			name:     "small binary",
			codeSize: 4096, // 1 page
			id:       "test",
			wantMin:  superBlobSize + blobSize + codeDirectorySize + 5 + 32, // header + id + 1 hash
		},
		{
			name:     "medium binary",
			codeSize: 16384, // 4 pages
			id:       "test-binary",
			wantMin:  superBlobSize + blobSize + codeDirectorySize + 12 + 4*32, // header + id + 4 hashes
		},
		{
			name:     "large binary",
			codeSize: 1024 * 1024, // 1MB = 256 pages
			id:       "large",
			wantMin:  superBlobSize + blobSize + codeDirectorySize + 6 + 256*32,
		},
		{
			name:     "partial page",
			codeSize: 5000, // just over 1 page
			id:       "x",
			wantMin:  superBlobSize + blobSize + codeDirectorySize + 2 + 2*32, // 2 pages worth of hashes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Size(tt.codeSize, tt.id)
			if got < tt.wantMin {
				t.Errorf("Size(%d, %q) = %d, want >= %d", tt.codeSize, tt.id, got, tt.wantMin)
			}
		})
	}
}

func TestSizeConsistency(t *testing.T) {
	// Size should increase with code size and id length
	size1 := Size(4096, "a")
	size2 := Size(8192, "a")
	size3 := Size(4096, "abc")

	if size2 <= size1 {
		t.Errorf("Size should increase with code size: Size(8192) = %d <= Size(4096) = %d", size2, size1)
	}

	if size3 <= size1 {
		t.Errorf("Size should increase with id length: Size(4096, 'abc') = %d <= Size(4096, 'a') = %d", size3, size1)
	}
}

func TestSign(t *testing.T) {
	tests := []struct {
		name     string
		codeSize int64
		id       string
		textOff  int64
		textSize int64
		isMain   bool
	}{
		{
			name:     "simple main binary",
			codeSize: 4096,
			id:       "test",
			textOff:  0,
			textSize: 4096,
			isMain:   true,
		},
		{
			name:     "library",
			codeSize: 8192,
			id:       "libtest",
			textOff:  0,
			textSize: 8192,
			isMain:   false,
		},
		{
			name:     "slasm binary",
			codeSize: 16896, // typical slasm output size
			id:       "slasm-binary",
			textOff:  0,
			textSize: 16384,
			isMain:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake file content
			content := make([]byte, tt.codeSize)
			for i := range content {
				content[i] = byte(i % 256)
			}

			// Calculate signature size
			sigSize := Size(tt.codeSize, tt.id)

			// Create output buffer
			out := make([]byte, sigSize)

			// Sign
			Sign(out, bytes.NewReader(content), tt.id, tt.codeSize, tt.textOff, tt.textSize, tt.isMain)

			// Verify SuperBlob header
			magic := binary.BigEndian.Uint32(out[0:4])
			if magic != CSMAGIC_EMBEDDED_SIGNATURE {
				t.Errorf("SuperBlob magic = 0x%x, want 0x%x", magic, CSMAGIC_EMBEDDED_SIGNATURE)
			}

			length := binary.BigEndian.Uint32(out[4:8])
			if length != uint32(sigSize) {
				t.Errorf("SuperBlob length = %d, want %d", length, sigSize)
			}

			count := binary.BigEndian.Uint32(out[8:12])
			if count != 1 {
				t.Errorf("SuperBlob count = %d, want 1", count)
			}

			// Verify Blob entry
			blobType := binary.BigEndian.Uint32(out[12:16])
			if blobType != CSSLOT_CODEDIRECTORY {
				t.Errorf("Blob type = %d, want %d", blobType, CSSLOT_CODEDIRECTORY)
			}

			blobOffset := binary.BigEndian.Uint32(out[16:20])
			expectedOffset := uint32(superBlobSize + blobSize)
			if blobOffset != expectedOffset {
				t.Errorf("Blob offset = %d, want %d", blobOffset, expectedOffset)
			}

			// Verify CodeDirectory header
			cdirMagic := binary.BigEndian.Uint32(out[20:24])
			if cdirMagic != CSMAGIC_CODEDIRECTORY {
				t.Errorf("CodeDirectory magic = 0x%x, want 0x%x", cdirMagic, CSMAGIC_CODEDIRECTORY)
			}

			// Check hash size is 32 (offset 36 in CodeDirectory, which starts at offset 20)
			hashSizeVal := out[20+36]
			if hashSizeVal != 32 {
				t.Errorf("Hash size = %d, want 32", hashSizeVal)
			}

			// Check hash type is SHA256 (offset 37 in CodeDirectory)
			hashTypeVal := out[20+37]
			if hashTypeVal != CS_HASHTYPE_SHA256 {
				t.Errorf("Hash type = %d, want %d (SHA256)", hashTypeVal, CS_HASHTYPE_SHA256)
			}
		})
	}
}

func TestSignDeterministic(t *testing.T) {
	// Same input should produce same output
	content := make([]byte, 4096)
	for i := range content {
		content[i] = byte(i)
	}

	id := "test"
	size := Size(4096, id)

	out1 := make([]byte, size)
	out2 := make([]byte, size)

	Sign(out1, bytes.NewReader(content), id, 4096, 0, 4096, true)
	Sign(out2, bytes.NewReader(content), id, 4096, 0, 4096, true)

	if !bytes.Equal(out1, out2) {
		t.Error("Sign should be deterministic: same input produced different output")
	}
}

func TestSignDifferentContent(t *testing.T) {
	// Different content should produce different signatures
	content1 := make([]byte, 4096)
	content2 := make([]byte, 4096)
	content2[0] = 1 // Just one byte different

	id := "test"
	size := Size(4096, id)

	out1 := make([]byte, size)
	out2 := make([]byte, size)

	Sign(out1, bytes.NewReader(content1), id, 4096, 0, 4096, true)
	Sign(out2, bytes.NewReader(content2), id, 4096, 0, 4096, true)

	if bytes.Equal(out1, out2) {
		t.Error("Different content should produce different signatures")
	}
}

func TestBlobPut(t *testing.T) {
	blob := Blob{
		typ:    CSSLOT_CODEDIRECTORY,
		offset: 0x14,
	}

	out := make([]byte, blobSize)
	blob.put(out)

	// Check big-endian encoding
	if binary.BigEndian.Uint32(out[0:4]) != CSSLOT_CODEDIRECTORY {
		t.Errorf("Blob type encoding failed")
	}
	if binary.BigEndian.Uint32(out[4:8]) != 0x14 {
		t.Errorf("Blob offset encoding failed")
	}
}

func TestSuperBlobPut(t *testing.T) {
	sb := SuperBlob{
		magic:  CSMAGIC_EMBEDDED_SIGNATURE,
		length: 256,
		count:  1,
	}

	out := make([]byte, superBlobSize)
	sb.put(out)

	if binary.BigEndian.Uint32(out[0:4]) != CSMAGIC_EMBEDDED_SIGNATURE {
		t.Errorf("SuperBlob magic encoding failed")
	}
	if binary.BigEndian.Uint32(out[4:8]) != 256 {
		t.Errorf("SuperBlob length encoding failed")
	}
	if binary.BigEndian.Uint32(out[8:12]) != 1 {
		t.Errorf("SuperBlob count encoding failed")
	}
}

func TestCodeDirectoryPut(t *testing.T) {
	cd := CodeDirectory{
		magic:        CSMAGIC_CODEDIRECTORY,
		length:       200,
		version:      0x20400,
		flags:        0x20002,
		hashOffset:   100,
		identOffset:  88,
		nSpecialSlots: 0,
		nCodeSlots:   4,
		codeLimit:    16384,
		hashSize:     32,
		hashType:     CS_HASHTYPE_SHA256,
		pageSize:     12,
		execSegFlags: CS_EXECSEG_MAIN_BINARY,
	}

	out := make([]byte, codeDirectorySize)
	cd.put(out)

	// Verify magic
	if binary.BigEndian.Uint32(out[0:4]) != CSMAGIC_CODEDIRECTORY {
		t.Errorf("CodeDirectory magic encoding failed")
	}

	// Verify version
	if binary.BigEndian.Uint32(out[8:12]) != 0x20400 {
		t.Errorf("CodeDirectory version encoding failed")
	}

	// Verify flags
	if binary.BigEndian.Uint32(out[12:16]) != 0x20002 {
		t.Errorf("CodeDirectory flags encoding failed")
	}

	// Verify hashSize (at offset 36)
	if out[36] != 32 {
		t.Errorf("CodeDirectory hashSize encoding failed: got %d, want 32", out[36])
	}

	// Verify hashType (at offset 37)
	if out[37] != CS_HASHTYPE_SHA256 {
		t.Errorf("CodeDirectory hashType encoding failed: got %d, want %d", out[37], CS_HASHTYPE_SHA256)
	}
}

func TestConstants(t *testing.T) {
	// Verify important constants
	if pageSize != 4096 {
		t.Errorf("pageSize = %d, want 4096", pageSize)
	}
	if pageSizeBits != 12 {
		t.Errorf("pageSizeBits = %d, want 12", pageSizeBits)
	}
	if hashSize != 32 {
		t.Errorf("hashSize = %d, want 32", hashSize)
	}
	if LC_CODE_SIGNATURE != 0x1d {
		t.Errorf("LC_CODE_SIGNATURE = 0x%x, want 0x1d", LC_CODE_SIGNATURE)
	}
	if blobSize != 8 {
		t.Errorf("blobSize = %d, want 8", blobSize)
	}
	if superBlobSize != 12 {
		t.Errorf("superBlobSize = %d, want 12", superBlobSize)
	}
	if codeDirectorySize != 88 {
		t.Errorf("codeDirectorySize = %d, want 88", codeDirectorySize)
	}
}
