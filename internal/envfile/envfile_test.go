package envfile

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRoundTripUntouched(t *testing.T) {
	src := "# comment\n\nA=1\nexport B=\"x y\" # note\nC='z'\nweird line\n"
	d := Parse([]byte(src))
	if got := string(d.Bytes()); got != src {
		t.Fatalf("round trip changed content:\n%q\n%q", got, src)
	}
}

func TestParseEntries(t *testing.T) {
	d := Parse([]byte("A=1 # c\nB=\"q # not comment\"\nC=\nexport D=x\n"))
	es := d.Entries()
	want := []struct{ k, v string }{{"A", "1"}, {"B", "q # not comment"}, {"C", ""}, {"D", "x"}}
	if len(es) != len(want) {
		t.Fatalf("got %d entries", len(es))
	}
	for i, w := range want {
		if es[i].Key != w.k || es[i].Value != w.v {
			t.Errorf("entry %d = %q=%q, want %q=%q", i, es[i].Key, es[i].Value, w.k, w.v)
		}
	}
}

func TestEditKeepsFormatting(t *testing.T) {
	d := Parse([]byte("A=1 # note\nB=\"x\"\n"))
	d.Find("A").SetValue("2")
	d.Find("B").SetValue("hello world")
	want := "A=2 # note\nB=\"hello world\"\n"
	if got := string(d.Bytes()); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestAddQuoting(t *testing.T) {
	d := Parse(nil)
	for _, kv := range [][2]string{{"A", "plain"}, {"B", "has space"}, {"C", "it's here"}, {"D", ""}} {
		if _, err := d.Add(kv[0], kv[1]); err != nil {
			t.Fatal(err)
		}
	}
	want := "A=plain\nB='has space'\nC=\"it's here\"\nD=\n"
	if got := string(d.Bytes()); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if _, err := d.Add("A", "x"); err == nil {
		t.Error("duplicate key accepted")
	}
	if _, err := d.Add("1bad", "x"); err == nil {
		t.Error("invalid key accepted")
	}
}

func TestCRLFPreserved(t *testing.T) {
	d := Parse([]byte("A=1\r\nB=2\r\n"))
	d.Find("B").SetValue("3")
	if got, want := string(d.Bytes()), "A=1\r\nB=3\r\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	d, err := Load(path)
	if err != nil || !d.Missing {
		t.Fatalf("missing file: doc=%v err=%v", d, err)
	}
	d.Add("A", "1")
	if err := d.Save(path); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "A=1\n" {
		t.Fatalf("saved %q", data)
	}
	if st, _ := os.Stat(path); st.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v", st.Mode().Perm())
	}
}

func TestMultilineValue(t *testing.T) {
	src := "KEY=\"-----BEGIN\nMIIBq==\n-----END\"\nB=1\n"
	d := Parse([]byte(src))
	es := d.Entries()
	if len(es) != 2 || es[0].Key != "KEY" || es[1].Key != "B" {
		t.Fatalf("entries %+v", es)
	}
	if !es[0].Multiline() || es[0].Value != "-----BEGIN\nMIIBq==\n-----END" {
		t.Fatalf("value %q", es[0].Value)
	}
	if err := es[0].SetValue("x"); !errors.Is(err, ErrMultiline) {
		t.Fatalf("SetValue err = %v", err)
	}
	if err := d.ToggleComment(0); !errors.Is(err, ErrMultiline) {
		t.Fatalf("ToggleComment err = %v", err)
	}
	if err := d.Rename(0, "CERT"); err != nil {
		t.Fatal(err)
	}
	es[1].SetValue("2")
	if got, want := string(d.Bytes()), "CERT=\"-----BEGIN\nMIIBq==\n-----END\"\nB=2\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestUnclosedQuoteStaysText(t *testing.T) {
	src := "A=\"open\nB=2\n"
	d := Parse([]byte(src))
	if d.Find("A") != nil || d.Find("B") == nil {
		t.Fatal("unclosed quote should leave the line as text")
	}
	if got := string(d.Bytes()); got != src {
		t.Fatalf("got %q", got)
	}
}

func TestSetValueKeepsMeaning(t *testing.T) {
	cases := []struct {
		src, value, want string
		err              error
	}{
		{`P='C:\temp'`, `C:\new`, `P='C:\new'`, nil},
		{`P='a'`, `it's`, `P="it's"`, nil},
		{`P='a'`, `it's C:\temp`, "", ErrUnquotable},
		{`P='a'`, `it's $HOME`, "", ErrUnquotable},
		{`P=a`, `x y`, `P='x y'`, nil},
		{`P=a`, `$HOME x`, `P="$HOME x"`, nil},
		{`P=a`, `C:\a b`, `P='C:\a b'`, nil},
		{`P=a`, `it's C:\a`, "", ErrUnquotable},
		{`P="a"`, `say "hi"\n`, `P="say \"hi\"\n"`, nil},
		{`P=a`, ` padded`, `P=' padded'`, nil},
	}
	for _, c := range cases {
		d := Parse([]byte(c.src))
		err := d.Items[0].SetValue(c.value)
		if !errors.Is(err, c.err) {
			t.Errorf("%s <- %s: err %v, want %v", c.src, c.value, err, c.err)
			continue
		}
		if c.err != nil {
			if got := string(d.Bytes()); got != c.src {
				t.Errorf("%s: changed after error: %q", c.src, got)
			}
			continue
		}
		if got := d.Items[0].String(); got != c.want {
			t.Errorf("%s <- %s: got %s, want %s", c.src, c.value, got, c.want)
		}
	}
}

func TestPreservesFileLayout(t *testing.T) {
	for _, src := range []string{
		"A=1",
		"A=1\r\nB=2\nC=3\r\n",
		"\ufeffA=1\n",
		"export\tA=1\n",
		"",
	} {
		d := Parse([]byte(src))
		if got := string(d.Bytes()); got != src {
			t.Errorf("round trip %q -> %q", src, got)
		}
	}
	d := Parse([]byte("\ufeffexport\tA=1\r\nB=2\n"))
	if d.Find("A") == nil || !d.Find("A").Export {
		t.Fatal("BOM or export<TAB> not handled")
	}
	d.Find("A").SetValue("9")
	d.Add("C", "3")
	if got, want := string(d.Bytes()), "\ufeffexport A=9\r\nB=2\nC=3\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestErrorKinds(t *testing.T) {
	d := Parse([]byte("weird line\n# just text\n# A=1\nA=2\n"))
	if err := d.ToggleComment(0); !errors.Is(err, ErrNotEntry) {
		t.Errorf("weird line: %v", err)
	}
	if err := d.ToggleComment(1); !errors.Is(err, ErrNotCommentedEntry) {
		t.Errorf("plain comment: %v", err)
	}
	if err := d.ToggleComment(2); !errors.Is(err, ErrDuplicateKey) {
		t.Errorf("duplicate: %v", err)
	}
	if err := d.Rename(3, "1x"); !errors.Is(err, ErrInvalidKey) {
		t.Errorf("rename: %v", err)
	}
}

func TestImport(t *testing.T) {
	d := Parse([]byte("A=1\n# end\n"))
	added, skipped := d.Import(d.EndOfEntries(), []byte("A=2\r\nB='x y'\nnot a var\nC=3"))
	if added != 2 || skipped != 1 {
		t.Fatalf("added %d skipped %d", added, skipped)
	}
	if got, want := string(d.Bytes()), "A=1\nB='x y'\nC=3\n# end\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestUndoSnapshot(t *testing.T) {
	d := Parse([]byte("A=1\n"))
	s := d.Clone()
	d.Find("A").SetValue("2")
	d.Add("B", "3")
	d.Restore(s)
	if got := string(d.Bytes()); got != "A=1\n" {
		t.Fatalf("got %q", got)
	}
}

func TestSaveDetectsExternalChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	os.WriteFile(path, []byte("A=1\n"), 0o640)
	d, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(path, []byte("A=1\nB=2\n"), 0o640)
	d.Find("A").SetValue("9")
	if err := d.Save(path); !errors.Is(err, ErrModified) {
		t.Fatalf("err = %v", err)
	}
	if data, _ := os.ReadFile(path); string(data) != "A=1\nB=2\n" {
		t.Fatalf("file overwritten: %q", data)
	}

	d, _ = Load(path)
	d.Find("A").SetValue("9")
	if err := d.Save(path); err != nil {
		t.Fatal(err)
	}
	d.Find("A").SetValue("8")
	if err := d.Save(path); err != nil {
		t.Fatalf("second save: %v", err)
	}
	if st, _ := os.Stat(path); st.Mode().Perm() != 0o640 {
		t.Errorf("mode = %v", st.Mode().Perm())
	}
}

func TestSaveMissingFileCreatedMeanwhile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	d, _ := Load(path)
	os.WriteFile(path, []byte("X=1\n"), 0o600)
	d.Add("A", "1")
	if err := d.Save(path); !errors.Is(err, ErrModified) {
		t.Fatalf("err = %v", err)
	}
}

func TestSaveInReadOnlyDir(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	os.WriteFile(path, []byte("A=1\n"), 0o600)
	d, _ := Load(path)
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(dir, 0o700)
	d.Find("A").SetValue("2")
	if err := d.Save(path); err != nil {
		t.Fatalf("save with read-only dir: %v", err)
	}
	if data, _ := os.ReadFile(path); string(data) != "A=2\n" {
		t.Fatalf("saved %q", data)
	}
}
