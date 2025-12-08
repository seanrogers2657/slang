package arch

import (
	"testing"
)

func TestEmitterBuilder(t *testing.T) {
	t.Run("NewEmitterBuilder", func(t *testing.T) {
		eb := NewEmitterBuilder()
		if eb == nil {
			t.Fatal("NewEmitterBuilder returned nil")
		}
		if eb.Builder == nil {
			t.Fatal("EmitterBuilder.Builder is nil")
		}
	})

	t.Run("Write", func(t *testing.T) {
		eb := NewEmitterBuilder()
		eb.Write("hello")
		eb.Write(" world")

		got := eb.Builder.String()
		want := "hello world"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("Writef", func(t *testing.T) {
		eb := NewEmitterBuilder()
		eb.Writef("mov %s, #%d\n", "x0", 42)

		got := eb.Builder.String()
		want := "mov x0, #42\n"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("String resets builder", func(t *testing.T) {
		eb := NewEmitterBuilder()
		eb.Write("first")

		first := eb.String()
		if first != "first" {
			t.Errorf("first call: got %q, want %q", first, "first")
		}

		// After String(), builder should be reset
		eb.Write("second")
		second := eb.String()
		if second != "second" {
			t.Errorf("second call: got %q, want %q", second, "second")
		}
	})

	t.Run("combined usage", func(t *testing.T) {
		eb := NewEmitterBuilder()
		eb.Write("    ")
		eb.Writef("add %s, %s, %s\n", "x2", "x0", "x1")
		eb.Write("    ret\n")

		got := eb.String()
		want := "    add x2, x0, x1\n    ret\n"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestArchitectureConstants(t *testing.T) {
	// Verify architecture constants are defined correctly
	if ArchARM64 != "arm64" {
		t.Errorf("ArchARM64 = %q, want %q", ArchARM64, "arm64")
	}
	if ArchAMD64 != "amd64" {
		t.Errorf("ArchAMD64 = %q, want %q", ArchAMD64, "amd64")
	}
}

func TestPlatformConstants(t *testing.T) {
	// Verify platform constants are defined correctly
	if PlatformDarwin != "darwin" {
		t.Errorf("PlatformDarwin = %q, want %q", PlatformDarwin, "darwin")
	}
	if PlatformLinux != "linux" {
		t.Errorf("PlatformLinux = %q, want %q", PlatformLinux, "linux")
	}
}

func TestTarget(t *testing.T) {
	target := Target{
		Arch:     ArchARM64,
		Platform: PlatformDarwin,
	}

	if target.Arch != ArchARM64 {
		t.Errorf("target.Arch = %q, want %q", target.Arch, ArchARM64)
	}
	if target.Platform != PlatformDarwin {
		t.Errorf("target.Platform = %q, want %q", target.Platform, PlatformDarwin)
	}
}
