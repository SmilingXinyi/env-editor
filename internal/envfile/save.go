package envfile

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Bytes renders the document.
func (d *Doc) Bytes() []byte {
	var b strings.Builder
	if d.BOM {
		b.WriteString(bom)
	}
	for i, it := range d.Items {
		b.WriteString(it.String())
		if i == len(d.Items)-1 && d.noFinalEOL {
			break
		}
		if it.eol != "" {
			b.WriteString(it.eol)
		} else {
			b.WriteString(d.EOL)
		}
	}
	return []byte(b.String())
}

// Load reads path. A missing file yields an empty document.
func Load(path string) (*Doc, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		d := Parse(nil)
		d.Missing, d.loaded = true, true
		return d, nil
	}
	if err != nil {
		return nil, err
	}
	d := Parse(data)
	d.disk, d.loaded = data, true
	return d, nil
}

// Save writes the document to path. It returns ErrModified if the file was
// changed by someone else since it was loaded or last saved. The file is
// replaced atomically when the directory allows it, keeping its mode and
// owner; otherwise it is rewritten in place.
func (d *Doc) Save(path string) error {
	if real, err := filepath.EvalSymlinks(path); err == nil {
		path = real
	}
	if err := d.checkDisk(path); err != nil {
		return err
	}
	data := d.Bytes()
	st, statErr := os.Stat(path)
	err := replace(path, data, st)
	if errors.Is(err, errNeedInPlace) && statErr == nil {
		err = writeInPlace(path, data)
	}
	if err != nil {
		return err
	}
	d.disk, d.loaded, d.Missing = data, true, false
	return nil
}

func (d *Doc) checkDisk(path string) error {
	if !d.loaded {
		return nil
	}
	cur, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		if !d.Missing {
			return ErrModified
		}
	case err != nil:
		return err
	case d.Missing || !bytes.Equal(cur, d.disk):
		return ErrModified
	}
	return nil
}

// errNeedInPlace means the file cannot be replaced without losing its
// owner, or the directory does not allow creating the temporary file.
var errNeedInPlace = errors.New("cannot replace file")

// replace writes data to a temporary file next to path and renames it over
// path. st describes the current file, or is nil if there is none.
func replace(path string, data []byte, st os.FileInfo) error {
	mode := os.FileMode(0o600)
	if st != nil {
		mode = st.Mode().Perm()
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".env-editor-*")
	if errors.Is(err, os.ErrPermission) {
		return fmt.Errorf("%w: %w", errNeedInPlace, err)
	}
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := fill(tmp, data, mode, st); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return err
	}
	syncDir(dir)
	return nil
}

func fill(f *os.File, data []byte, mode os.FileMode, st os.FileInfo) error {
	if _, err := f.Write(data); err != nil {
		return err
	}
	if err := f.Chmod(mode); err != nil {
		return err
	}
	if st != nil {
		if err := chownLike(f, st); err != nil {
			return fmt.Errorf("%w: %w", errNeedInPlace, err)
		}
	}
	return f.Sync()
}

// writeInPlace truncates and rewrites path. It is not atomic but keeps the
// file's identity, mode and owner.
func writeInPlace(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// syncDir flushes the directory entry of a rename. Errors are ignored as
// not every platform can sync a directory.
func syncDir(dir string) {
	if f, err := os.Open(dir); err == nil {
		f.Sync()
		f.Close()
	}
}
