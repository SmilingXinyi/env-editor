//go:build !unix

package envfile

import "os"

// chownLike is a no-op where files have no Unix owner.
func chownLike(*os.File, os.FileInfo) error { return nil }
