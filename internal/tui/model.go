package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"env-editor/internal/envfile"
	"env-editor/internal/i18n"
)

type mode int

const (
	modeList mode = iota
	modeEditValue
	modeEditComment
	modeNewKey
	modeNewValue
	modeRename
	modeSearch
)

const maxUndo = 100

type snapshot struct {
	doc    *envfile.Doc
	cursor int
}

// Model is the Bubble Tea model of the editor.
type Model struct {
	path          string
	doc           *envfile.Doc
	cursor        int  // 0 is the "new variable" row, n is line n-1
	prevCursor    int  // row to return to when an insert, new entry or search is cancelled
	inserting     bool // the comment being edited was just inserted
	pendingDelete bool // the next "d" confirms deleting the selected entry
	offset        int
	mode          mode
	in            input
	newKey        string
	newAt         int      // index where the new entry is inserted
	keepCase      bool     // do not upper-case new keys
	masked        bool     // hide values in the list
	base          snapshot // document before the current command
	undo          []snapshot
	saved         bool // the current command wrote the file
	status        string
	isErr         bool
	width, height int
}

// New creates the editor for the document loaded from path. With masked set
// the values are hidden until "v" is pressed.
func New(path string, doc *envfile.Doc, masked bool) Model {
	m := Model{path: path, doc: doc, masked: masked, width: 80, height: 24}
	if doc.Missing {
		m.status = i18n.FileMissing.String()
	}
	return m
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.fixScroll()
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		m.status, m.isErr = "", false
		// An edit session starts in list mode, so base is the document
		// before the session even when a placeholder line was inserted.
		if m.mode == modeList {
			m.base = snapshot{m.doc.Clone(), m.cursor}
		}
		m.saved = false
		var cmd tea.Cmd
		if m.mode == modeList {
			cmd = m.updateList(msg)
		} else {
			m.updateEdit(msg)
		}
		if m.saved {
			m.undo = append(m.undo, m.base)
			if len(m.undo) > maxUndo {
				m.undo = m.undo[1:]
			}
		}
		m.fixScroll()
		return m, cmd
	}
	return m, nil
}

func (m *Model) updateList(msg tea.KeyMsg) tea.Cmd {
	items := m.doc.Items
	pending := m.pendingDelete
	m.pendingDelete = false
	switch msg.String() {
	case "q":
		return tea.Quit
	case "esc":
		m.status = i18n.QuitHint.String()
	case "up", "k":
		m.cursor = max(0, m.cursor-1)
	case "down", "j":
		m.cursor = min(len(items), m.cursor+1)
	case "n":
		m.insertBlank()
	case "f":
		sel, removed := m.doc.CollapseBlanks(m.cursor - 1)
		if removed == 0 {
			m.status = i18n.NoBlankRuns.String()
			break
		}
		m.cursor = sel + 1
		m.persist(i18n.BlanksMerged.Fmt(removed))
	case "a":
		at := m.doc.EndOfEntries()
		if m.cursor > 0 {
			at = m.cursor - 1
		}
		m.startNew(at)
	case "r":
		m.startRename()
	case "c":
		m.insertComment()
	case "#":
		m.toggleComment()
	case "/":
		m.prevCursor = m.cursor
		m.mode = modeSearch
		m.in = input{}
	case "d":
		m.deleteRow(pending)
	case "y":
		m.copyRow()
	case "u":
		m.undoLast()
	case "R":
		m.reload()
	case "v":
		m.masked = !m.masked
	case "enter":
		switch {
		case m.cursor == 0:
			m.startNew(m.doc.EndOfEntries())
		case items[m.cursor-1].Entry:
			if items[m.cursor-1].Multiline() {
				m.status, m.isErr = i18n.MultilineHint.String(), true
				break
			}
			m.mode = modeEditValue
			m.in = newInput(items[m.cursor-1].Value)
		case items[m.cursor-1].IsComment():
			m.mode = modeEditComment
			m.in = newInput(items[m.cursor-1].Raw)
		}
	}
	return nil
}

func (m *Model) startNew(at int) {
	m.prevCursor = m.cursor
	m.newAt = at
	m.cursor = 0
	m.mode = modeNewKey
	m.in = input{}
}

func (m *Model) startRename() {
	if m.cursor == 0 || !m.doc.Items[m.cursor-1].Entry {
		m.status, m.isErr = i18n.RenameOnlyVars.String(), true
		return
	}
	m.mode = modeRename
	m.in = newInput(m.doc.Items[m.cursor-1].Key)
}

func (m *Model) toggleComment() {
	if m.cursor == 0 {
		return
	}
	it := m.doc.Items[m.cursor-1]
	commenting, key := it.Entry, it.Key
	if err := m.doc.ToggleComment(m.cursor - 1); err != nil {
		m.fail(err)
		return
	}
	if commenting {
		m.persist(i18n.Commented.Fmt(key))
		return
	}
	m.persist(i18n.Uncommented.Fmt(m.doc.Items[m.cursor-1].Key))
}

func (m *Model) deleteRow(confirmed bool) {
	if m.cursor == 0 {
		return
	}
	it := m.doc.Items[m.cursor-1]
	if it.Entry && !confirmed {
		m.pendingDelete = true
		m.status = i18n.ConfirmDelete.Fmt(it.Key)
		return
	}
	m.doc.Remove(m.cursor - 1)
	m.cursor = min(m.cursor, len(m.doc.Items))
	if it.Entry {
		m.persist(i18n.DeletedVar.Fmt(it.Key))
		return
	}
	m.persist(i18n.Deleted.String())
}

// copyRow keeps a commented-out copy above a variable, so its old value is
// at hand while it is edited, and duplicates any other line below itself.
func (m *Model) copyRow() {
	if m.cursor == 0 {
		return
	}
	idx := m.cursor - 1
	it := m.doc.Items[idx]
	if it.Entry {
		if it.Multiline() {
			m.fail(envfile.ErrMultiline)
			return
		}
		m.doc.Insert(idx, envfile.NewComment("# "+it.String()))
		m.cursor++
		m.persist(i18n.CommentBackup.String())
		return
	}
	clone := *it
	m.doc.Insert(idx+1, &clone)
	m.cursor++
	m.persist(i18n.Copied.String())
}

// insertIndex is where a new line goes: above the selected row, or at the
// top of the file when the "new variable" row is selected.
func (m Model) insertIndex() int { return max(0, m.cursor-1) }

func (m *Model) insertBlank() {
	m.doc.Insert(m.insertIndex(), envfile.NewBlank())
	if m.cursor > 0 {
		m.cursor++
	}
	m.persist(i18n.BlankInserted.String())
}

func (m *Model) insertComment() {
	idx := m.insertIndex()
	m.doc.Insert(idx, envfile.NewComment("# "))
	m.prevCursor = m.cursor
	m.cursor = idx + 1
	m.inserting = true
	m.mode = modeEditComment
	m.in = newInput("# ")
}

func (m *Model) undoLast() {
	if len(m.undo) == 0 {
		m.status = i18n.NothingToUndo.String()
		return
	}
	s := m.undo[len(m.undo)-1]
	cur := m.doc.Clone()
	m.doc.Restore(s.doc)
	if err := m.doc.Save(m.path); err != nil {
		m.doc.Restore(cur)
		m.fail(err)
		return
	}
	m.undo = m.undo[:len(m.undo)-1]
	m.cursor = min(s.cursor, len(m.doc.Items))
	m.status = i18n.Undone.String()
}

func (m *Model) reload() {
	doc, err := envfile.Load(m.path)
	if err != nil {
		m.status, m.isErr = i18n.ReloadFailed.Fmt(err.Error()), true
		return
	}
	m.doc, m.undo = doc, nil
	m.cursor = min(m.cursor, len(doc.Items))
	m.status = i18n.Reloaded.String()
}
