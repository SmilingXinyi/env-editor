// Package envfile reads and writes .env files while preserving comments,
// blank lines, line endings and the formatting of lines that were not edited.
package envfile

import (
	"errors"
	"strings"
)

// Errors returned by the package. Match them with errors.Is.
var (
	ErrInvalidKey        = errors.New("invalid key")
	ErrDuplicateKey      = errors.New("key already exists")
	ErrNotEntry          = errors.New("line is not a variable")
	ErrNotCommentedEntry = errors.New("comment is not a commented-out variable")
	ErrMultiline         = errors.New("not supported for multi-line values")
	ErrUnquotable        = errors.New("value cannot be quoted without changing its meaning")
	ErrModified          = errors.New("file was modified by another program")
)

const bom = "\ufeff"

// Item is one line of a .env file. A quoted value that spans several lines
// is a single Item whose Raw contains the inner line breaks.
type Item struct {
	Raw    string
	Entry  bool
	Export bool
	Key    string
	// Value is the text between the quotes as written in the file: escapes
	// such as \n are kept in double-quoted values, other values are literal.
	Value  string
	Quote  byte   // 0, '\'' or '"'
	Suffix string // trailing inline comment, including leading whitespace
	dirty  bool
	eol    string // line ending after the item; "" means Doc.EOL
}

// Doc is a parsed .env file.
type Doc struct {
	Items      []*Item
	EOL        string // line ending for new lines, the most common one in the file
	BOM        bool   // the file starts with a UTF-8 byte order mark
	Missing    bool   // the file did not exist when loaded
	noFinalEOL bool   // the last line has no line ending
	disk       []byte // file content as last loaded or saved
	loaded     bool   // disk and Missing describe a real file
}

// ValidKey reports whether key is acceptable as a variable name.
func ValidKey(key string) bool {
	if key == "" {
		return false
	}
	for i, r := range key {
		switch {
		case r == '_', r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z':
		case i > 0 && (r >= '0' && r <= '9' || r == '.' || r == '-'):
		default:
			return false
		}
	}
	return true
}

// Parse parses the content of a .env file.
func Parse(data []byte) *Doc {
	d := &Doc{}
	s, hasBOM := strings.CutPrefix(string(data), bom)
	d.BOM = hasBOM
	crlf, lf := 0, 0
	for p := 0; p < len(s); {
		end := lineEnd(s, p)
		it, q := parseItem(lineText(s, p, end))
		if q >= 0 {
			// A quoted value is still open: it ends on the line holding
			// the closing quote. Without one the line stays plain text.
			if c := closingQuote(s[p+q:]); c > 0 {
				end = lineEnd(s, p+q+c)
				it, _ = parseItem(lineText(s, p, end))
			}
		}
		switch {
		case end == len(s):
			d.noFinalEOL = true
		case end > p && s[end-1] == '\r':
			it.eol = "\r\n"
			crlf++
		default:
			it.eol = "\n"
			lf++
		}
		d.Items = append(d.Items, &it)
		p = end + 1
	}
	d.EOL = "\n"
	if crlf > lf {
		d.EOL = "\r\n"
	}
	return d
}

// lineEnd returns the index of the line break ending the line at i, or len(s).
func lineEnd(s string, i int) int {
	if n := strings.IndexByte(s[i:], '\n'); n >= 0 {
		return i + n
	}
	return len(s)
}

// lineText returns s[p:end] without the carriage return of a CRLF ending.
func lineText(s string, p, end int) string {
	if end < len(s) {
		return strings.TrimSuffix(s[p:end], "\r")
	}
	return s[p:end]
}

// parseItem parses the text of one item. When a quoted value is opened but
// not closed it returns a plain item and the offset of the opening quote;
// otherwise the offset is -1.
func parseItem(raw string) (Item, int) {
	plain := Item{Raw: raw}
	s := strings.TrimLeft(raw, " \t")
	if s == "" || s[0] == '#' {
		return plain, -1
	}
	it := Item{Raw: raw, Entry: true}
	if len(s) > 7 && strings.HasPrefix(s, "export") && (s[6] == ' ' || s[6] == '\t') {
		it.Export = true
		s = strings.TrimLeft(s[7:], " \t")
	}
	eq := strings.IndexByte(s, '=')
	if eq <= 0 {
		return plain, -1
	}
	it.Key = strings.TrimRight(s[:eq], " \t")
	if !ValidKey(it.Key) {
		return plain, -1
	}
	rest := strings.TrimLeft(s[eq+1:], " \t")

	if rest != "" && (rest[0] == '"' || rest[0] == '\'') {
		end := closingQuote(rest)
		if end < 0 {
			return plain, len(raw) - len(rest)
		}
		it.Quote = rest[0]
		it.Value = rest[1:end]
		it.Suffix = rest[end+1:]
		return it, -1
	}
	if strings.HasPrefix(rest, "#") {
		it.Suffix = " " + rest
		return it, -1
	}
	for i := 1; i < len(rest); i++ {
		if rest[i] == '#' && (rest[i-1] == ' ' || rest[i-1] == '\t') {
			it.Value = strings.TrimRight(rest[:i], " \t")
			it.Suffix = rest[len(it.Value):]
			return it, -1
		}
	}
	it.Value = strings.TrimRight(rest, " \t")
	return it, -1
}

func closingQuote(s string) int {
	q := s[0]
	for i := 1; i < len(s); i++ {
		if q == '"' && s[i] == '\\' {
			i++
			continue
		}
		if s[i] == q {
			return i
		}
	}
	return -1
}

// needsQuote reports whether an unquoted value would be misread.
func needsQuote(v string) bool {
	return v != strings.TrimSpace(v) || strings.ContainsAny(v, " \t#\"'")
}

// escapeDoubleQuotes makes v valid between double quotes. Existing escape
// pairs are kept, so text that is already valid comes back unchanged.
func escapeDoubleQuotes(v string) string {
	var b strings.Builder
	for i := 0; i < len(v); i++ {
		c := v[i]
		switch {
		case c == '\\' && i+1 < len(v):
			b.WriteByte(c)
			i++
			b.WriteByte(v[i])
		case c == '\\':
			b.WriteString(`\\`)
		case c == '"':
			b.WriteString(`\"`)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// quoteFor picks the quoting for value v of an item written with quote q,
// such that v keeps its meaning. A double-quoted v is taken as written (with
// escapes); other values are literal. Switching quote style is only done
// when it cannot change how backslashes or $ are interpreted.
func quoteFor(q byte, v string) (byte, string, error) {
	switch q {
	case '"':
		return '"', escapeDoubleQuotes(v), nil
	case '\'':
		if !strings.Contains(v, "'") {
			return '\'', v, nil
		}
		if !strings.ContainsAny(v, `\$`) {
			return '"', escapeDoubleQuotes(v), nil
		}
	default:
		if !needsQuote(v) {
			return 0, v, nil
		}
		if !strings.ContainsAny(v, "'$") {
			return '\'', v, nil
		}
		if !strings.Contains(v, `\`) {
			return '"', escapeDoubleQuotes(v), nil
		}
	}
	return 0, "", ErrUnquotable
}

func (it *Item) format() string {
	var b strings.Builder
	if it.Export {
		b.WriteString("export ")
	}
	b.WriteString(it.Key)
	b.WriteByte('=')
	if it.Quote != 0 {
		b.WriteByte(it.Quote)
		b.WriteString(it.Value)
		b.WriteByte(it.Quote)
	} else {
		b.WriteString(it.Value)
	}
	b.WriteString(it.Suffix)
	return b.String()
}

// String returns the item as it will be written. Untouched items are
// returned verbatim.
func (it *Item) String() string {
	if !it.Entry || !it.dirty {
		return it.Raw
	}
	return it.format()
}

// Multiline reports whether the item spans several lines.
func (it *Item) Multiline() bool {
	return strings.ContainsAny(it.String(), "\r\n")
}

// SetValue changes the value of an entry. See Item.Value for how v is read.
// The value is quoted as needed; it is refused if that would change its
// meaning.
func (it *Item) SetValue(v string) error {
	if !it.Entry {
		return ErrNotEntry
	}
	if it.Multiline() {
		return ErrMultiline
	}
	q, out, err := quoteFor(it.Quote, v)
	if err != nil {
		return err
	}
	if q == it.Quote && out == it.Value {
		return nil
	}
	it.Quote, it.Value, it.dirty = q, out, true
	return nil
}

// IsBlank reports whether the line is empty or whitespace only.
func (it *Item) IsBlank() bool {
	return !it.Entry && strings.TrimSpace(it.Raw) == ""
}

// IsComment reports whether the line is a # comment.
func (it *Item) IsComment() bool {
	return !it.Entry && strings.HasPrefix(strings.TrimLeft(it.Raw, " \t"), "#")
}

// NewBlank returns an empty line.
func NewBlank() *Item { return &Item{} }

// NewComment returns a comment line with the given raw text.
func NewComment(raw string) *Item { return &Item{Raw: raw} }

// Clone returns a deep copy of the document.
func (d *Doc) Clone() *Doc {
	c := *d
	c.Items = make([]*Item, len(d.Items))
	for i, it := range d.Items {
		cp := *it
		c.Items[i] = &cp
	}
	return &c
}

// Restore replaces the content with that of snapshot s, which was taken
// with Clone. What is known about the file on disk is kept.
func (d *Doc) Restore(s *Doc) {
	c := s.Clone()
	d.Items, d.EOL, d.BOM, d.noFinalEOL = c.Items, c.EOL, c.BOM, c.noFinalEOL
}

// Insert places it at index i of the document.
func (d *Doc) Insert(i int, it *Item) {
	d.Items = append(d.Items, nil)
	copy(d.Items[i+1:], d.Items[i:])
	d.Items[i] = it
}

// Remove deletes the line at index i.
func (d *Doc) Remove(i int) {
	d.Items = append(d.Items[:i], d.Items[i+1:]...)
}

// ToggleComment comments out an entry, or restores a commented-out entry.
func (d *Doc) ToggleComment(i int) error {
	it := d.Items[i]
	if it.Entry {
		if it.Multiline() {
			return ErrMultiline
		}
		*it = Item{Raw: "# " + it.String(), eol: it.eol}
		return nil
	}
	if !it.IsComment() {
		return ErrNotEntry
	}
	rest := strings.TrimPrefix(strings.TrimLeft(it.Raw, " \t"), "#")
	rest = strings.TrimPrefix(rest, " ")
	restored, _ := parseItem(rest)
	if !restored.Entry {
		return ErrNotCommentedEntry
	}
	if d.Find(restored.Key) != nil {
		return ErrDuplicateKey
	}
	restored.eol = it.eol
	*it = restored
	return nil
}

// Rename changes the key of the entry at index i. The rest of the line is
// kept as written.
func (d *Doc) Rename(i int, key string) error {
	it := d.Items[i]
	if !it.Entry {
		return ErrNotEntry
	}
	if key == it.Key {
		return nil
	}
	if !ValidKey(key) {
		return ErrInvalidKey
	}
	if d.Find(key) != nil {
		return ErrDuplicateKey
	}
	if !it.dirty {
		s := strings.TrimLeft(it.Raw, " \t")
		if it.Export {
			s = strings.TrimLeft(s[len("export"):], " \t")
		}
		off := len(it.Raw) - len(s)
		it.Raw = it.Raw[:off] + key + it.Raw[off+len(it.Key):]
	}
	it.Key = key
	return nil
}

// CollapseBlanks merges runs of consecutive blank lines into one. It returns
// the new index of the line that was at sel (-1 stays -1) and the number of
// lines removed.
func (d *Doc) CollapseBlanks(sel int) (int, int) {
	out := make([]*Item, 0, len(d.Items))
	newSel, removed := sel, 0
	for i, it := range d.Items {
		if it.IsBlank() && len(out) > 0 && out[len(out)-1].IsBlank() {
			removed++
			if i == sel {
				newSel = len(out) - 1
			}
			continue
		}
		if i == sel {
			newSel = len(out)
		}
		out = append(out, it)
	}
	d.Items = out
	return newSel, removed
}

// Entries returns the KEY=VALUE lines in file order.
func (d *Doc) Entries() []*Item {
	var out []*Item
	for _, it := range d.Items {
		if it.Entry {
			out = append(out, it)
		}
	}
	return out
}

// Find returns the entry with the given key, or nil.
func (d *Doc) Find(key string) *Item {
	for _, it := range d.Items {
		if it.Entry && it.Key == key {
			return it
		}
	}
	return nil
}

// EndOfEntries is the index just after the last entry, where new entries
// go by default so they stay with the others rather than after trailing
// comments.
func (d *Doc) EndOfEntries() int {
	for i := len(d.Items) - 1; i >= 0; i-- {
		if d.Items[i].Entry {
			return i + 1
		}
	}
	return len(d.Items)
}

// Add appends a new entry. See AddAt.
func (d *Doc) Add(key, value string) (*Item, error) {
	return d.AddAt(len(d.Items), key, value)
}

// AddAt inserts a new entry at index i. The key must be valid and not yet
// present; value is written as typed and quoted only where an unquoted
// value would be misread.
func (d *Doc) AddAt(i int, key, value string) (*Item, error) {
	if !ValidKey(key) {
		return nil, ErrInvalidKey
	}
	if d.Find(key) != nil {
		return nil, ErrDuplicateKey
	}
	q, v, err := quoteFor(0, value)
	if err != nil {
		return nil, err
	}
	it := &Item{Entry: true, Key: key, Value: v, Quote: q, dirty: true}
	d.Insert(i, it)
	return it, nil
}

// Import inserts the entries found in data at index i, as written. Entries
// whose key is already present are skipped; other lines are ignored.
func (d *Doc) Import(i int, data []byte) (added, skipped int) {
	for _, it := range Parse(data).Items {
		if !it.Entry {
			continue
		}
		if d.Find(it.Key) != nil {
			skipped++
			continue
		}
		it.eol = ""
		d.Insert(i+added, it)
		added++
	}
	return added, skipped
}
