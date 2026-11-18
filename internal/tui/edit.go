package tui

import (
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"env-editor/internal/envfile"
	"env-editor/internal/i18n"
)

func (m *Model) updateEdit(msg tea.KeyMsg) {
	switch msg.String() {
	case "esc":
		m.cancelEdit()
		return
	case "enter":
		m.commit()
		return
	case "ctrl+t":
		if m.mode == modeNewKey {
			m.keepCase = !m.keepCase
			if !m.keepCase {
				pos := m.in.pos
				m.in = newInput(strings.ToUpper(m.in.String()))
				m.in.pos = pos
			}
		}
		return
	case "down", "ctrl+n":
		if m.mode == modeSearch {
			m.search(m.cursor, 1)
		}
		return
	case "up", "ctrl+p":
		if m.mode == modeSearch {
			m.search(m.cursor-2, -1)
		}
		return
	}
	if msg.Type == tea.KeyRunes && msg.Paste && strings.ContainsAny(string(msg.Runes), "\r\n") {
		m.pasteLines(string(msg.Runes))
		return
	}
	if msg.Type == tea.KeyRunes && !msg.Alt && m.mode == modeNewKey {
		key, rest, hasEq := strings.Cut(string(msg.Runes), "=")
		if !m.keepCase {
			key = strings.ToUpper(key)
		}
		m.in.insert(key)
		if hasEq {
			m.commit()
			if m.mode == modeNewValue {
				m.in.insert(rest)
			}
		}
		return
	}
	if m.in.handleKey(msg) && m.mode == modeSearch {
		m.search(max(0, m.prevCursor-1), 1)
	}
}

// pasteLines handles a paste that spans several lines. In the KEY field it
// imports the pasted variables; elsewhere it is refused, as values and
// comments are single lines.
func (m *Model) pasteLines(text string) {
	if m.mode != modeNewKey {
		m.status, m.isErr = i18n.PasteSingleLine.String(), true
		return
	}
	added, skipped := m.doc.Import(m.newAt, []byte(text))
	if added == 0 {
		m.status, m.isErr = i18n.ImportNone.String(), true
		if skipped > 0 {
			m.status = i18n.ImportNoneSkipped.Fmt(skipped)
		}
		return
	}
	m.mode = modeList
	m.cursor = m.newAt + 1
	msg := i18n.Imported.Fmt(added)
	if skipped > 0 {
		msg = i18n.ImportedSkipped.Fmt(added, skipped)
	}
	m.persist(msg)
}

func (m *Model) cancelEdit() {
	switch m.mode {
	case modeNewValue:
		m.mode = modeNewKey
		m.in = newInput(m.newKey)
		return
	case modeNewKey, modeSearch:
		m.cursor = m.prevCursor
	case modeEditComment:
		if m.inserting {
			m.doc.Remove(m.cursor - 1)
			m.cursor, m.inserting = m.prevCursor, false
		}
	}
	m.mode = modeList
	m.status = i18n.Cancelled.String()
}

// search selects the first item matching the query, looking from item index
// from in direction dir and wrapping around.
func (m *Model) search(from, dir int) {
	q := strings.ToLower(m.in.String())
	n := len(m.doc.Items)
	if q == "" || n == 0 {
		m.cursor = m.prevCursor
		return
	}
	for k := 0; k < n; k++ {
		i := ((from+dir*k)%n + n) % n
		it := m.doc.Items[i]
		text := it.Raw
		if it.Entry {
			text = it.Key + "\x00" + it.Value
		}
		if strings.Contains(strings.ToLower(text), q) {
			m.cursor = i + 1
			return
		}
	}
	m.status, m.isErr = i18n.NoMatch.String(), true
}

func (m *Model) commit() {
	switch m.mode {
	case modeSearch:
		m.mode = modeList
	case modeEditValue:
		e := m.doc.Items[m.cursor-1]
		old := e.String()
		if err := e.SetValue(m.in.String()); err != nil {
			m.fail(err)
			return
		}
		m.mode = modeList
		if e.String() == old {
			m.status = i18n.Unchanged.String()
			return
		}
		m.persist(i18n.Saved.Fmt(e.Key))
	case modeEditComment:
		text := strings.TrimLeft(m.in.String(), " \t")
		if !strings.HasPrefix(text, "#") {
			text = "# " + text
		}
		m.doc.Items[m.cursor-1].Raw = text
		m.mode = modeList
		if m.inserting {
			m.cursor = m.prevCursor
			if m.prevCursor > 0 {
				m.cursor++
			}
			m.inserting = false
		}
		m.persist(i18n.SavedComment.String())
	case modeRename:
		idx := m.cursor - 1
		old, key := m.doc.Items[idx].Key, strings.TrimSpace(m.in.String())
		if key == old {
			m.mode = modeList
			m.status = i18n.Unchanged.String()
			return
		}
		if err := m.doc.Rename(idx, key); err != nil {
			m.fail(err)
			return
		}
		m.mode = modeList
		m.persist(i18n.Renamed.Fmt(old, key))
	case modeNewKey:
		key := strings.TrimSpace(m.in.String())
		if !envfile.ValidKey(key) {
			m.fail(envfile.ErrInvalidKey)
			return
		}
		if m.doc.Find(key) != nil {
			m.fail(envfile.ErrDuplicateKey)
			return
		}
		m.newKey = key
		m.mode = modeNewValue
		m.in = input{}
	case modeNewValue:
		if _, err := m.doc.AddAt(m.newAt, m.newKey, m.in.String()); err != nil {
			m.fail(err)
			return
		}
		m.cursor = m.newAt + 1
		m.mode = modeList
		m.persist(i18n.Added.Fmt(m.newKey))
	}
}

// persist saves the document. If that fails the command is rolled back so
// that what is shown matches the file.
func (m *Model) persist(ok string) {
	if err := m.doc.Save(m.path); err != nil {
		m.doc.Restore(m.base.doc)
		m.cursor = min(m.base.cursor, len(m.doc.Items))
		m.mode, m.inserting = modeList, false
		m.fail(err)
		return
	}
	m.saved = true
	m.status = ok
}

func (m *Model) fail(err error) {
	m.status, m.isErr = describe(err), true
}

// describe turns an error into a message for the status line.
func describe(err error) string {
	switch {
	case errors.Is(err, envfile.ErrInvalidKey):
		return i18n.ErrInvalidKey.String()
	case errors.Is(err, envfile.ErrDuplicateKey):
		return i18n.ErrDuplicateKey.String()
	case errors.Is(err, envfile.ErrNotEntry):
		return i18n.ErrNotEntry.String()
	case errors.Is(err, envfile.ErrNotCommentedEntry):
		return i18n.ErrNotCommented.String()
	case errors.Is(err, envfile.ErrMultiline):
		return i18n.ErrMultiline.String()
	case errors.Is(err, envfile.ErrUnquotable):
		return i18n.ErrUnquotable.String()
	case errors.Is(err, envfile.ErrModified):
		return i18n.ErrModified.String()
	}
	return i18n.SaveFailed.Fmt(err.Error())
}
