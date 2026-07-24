// Package stdlib bundles the Slang standard library source into the compiler
// binary. The .sl sources under std/ are embedded at build time so that
// `import "std/..."` resolves without any files on disk — the toolchain ships
// as a single self-contained binary.
package stdlib

import "embed"

// FS holds the embedded standard-library source tree. Package directories live
// under the "std/" prefix (e.g. "std/math/math.sl"), matching the import paths
// used in Slang source.
//
//go:embed std
var FS embed.FS
