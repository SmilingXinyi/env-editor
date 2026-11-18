//go:build unix

package envfile

import (
	"os"
	"syscall"
)

// chownLike gives f the owner and group of the file described by st.
func chownLike(f *os.File, st os.FileInfo) error {
	want, ok := st.Sys().(*syscall.Stat_t)
	if !ok {
		return nil
	}
	cur, err := f.Stat()
	if err != nil {
		return err
	}
	if have, ok := cur.Sys().(*syscall.Stat_t); ok && have.Uid == want.Uid && have.Gid == want.Gid {
		return nil
	}
	return f.Chown(int(want.Uid), int(want.Gid))
}
