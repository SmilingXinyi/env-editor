package tui

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"env-editor/internal/envfile"
	"env-editor/internal/i18n"
)

const maxKeyWidth = 32

var (
	titleStyle = lipgloss.NewStyle().Bold(true)
	dimStyle   = lipgloss.NewStyle().Faint(true)
	keyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	selStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("39")).Bold(true)
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)

// KeyHelp lists the key bindings for --help. The editor shows a shorter
// version of it at the bottom of the screen, see helpGroups.
func KeyHelp() string { return i18n.HelpFull.String() }

// chrome is the number of rows around the list: title, blank line, blank
// line and status line.
const chrome = 4

func (m Model) visibleRows() int { return max(3, m.height-chrome-len(m.helpBlock())) }

func (m *Model) fixScroll() {
	v := m.visibleRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+v {
		m.offset = m.cursor - v + 1
	}
	m.offset = max(0, min(m.offset, len(m.doc.Items)+1-v))
}

// printable maps control characters to visible symbols, so that file
// content cannot move the cursor or inject terminal escape sequences.
func printable(r rune) rune {
	switch {
	case r < 0x20:
		return 0x2400 + r // ␀ … ␟, e.g. ␉ for a tab
	case r == 0x7f:
		return '␡'
	case unicode.IsControl(r):
		return '�'
	}
	return r
}

func sanitize(s string) string { return strings.Map(printable, s) }

func trunc(s string, w int) string { return runewidth.Truncate(sanitize(s), max(0, w), "…") }

func (m Model) View() string {
	w := max(10, m.width)
	entries := m.doc.Entries()

	kw := 0
	for _, e := range entries {
		kw = max(kw, runewidth.StringWidth(sanitize(e.Key)))
	}
	kw = min(kw, maxKeyWidth)

	var b strings.Builder
	title := i18n.Title.Fmt(m.path, len(entries))
	if m.masked {
		title += i18n.TitleMasked.String()
	}
	b.WriteString(titleStyle.Render(trunc(title, w)))
	b.WriteString("\n\n")

	v := m.visibleRows()
	end := min(len(m.doc.Items)+1, m.offset+v)
	for r := m.offset; r < end; r++ {
		b.WriteString(m.renderRow(r, m.doc.Items, kw, w))
		b.WriteByte('\n')
	}
	for i := end - m.offset; i < v; i++ {
		b.WriteByte('\n')
	}

	b.WriteByte('\n')
	switch {
	case m.mode == modeSearch:
		prefix := i18n.SearchPrompt.String()
		b.WriteString(keyStyle.Render(prefix) + m.in.view(w-runewidth.StringWidth(prefix)-1))
		if m.status != "" {
			b.WriteString("  " + errStyle.Render(m.status))
		}
	case m.status == "":
	case m.isErr:
		b.WriteString(errStyle.Render(trunc(m.status, w)))
	default:
		b.WriteString(okStyle.Render(trunc(m.status, w)))
	}
	for _, l := range m.helpBlock() {
		b.WriteByte('\n')
		b.WriteString(dimStyle.Render(l))
	}
	return b.String()
}

// helpGroup is one line of the help, or several when it does not fit.
type helpGroup struct {
	label string
	items []string
}

func (m Model) helpGroups(md mode) []helpGroup {
	switch md {
	case modeList:
		return []helpGroup{
			{i18n.GroupBrowse.String(), []string{i18n.HelpSelect.String(), i18n.HelpEdit.String(), i18n.HelpSearch.String(), i18n.HelpMask.String(), i18n.HelpReload.String(), i18n.HelpQuit.String()}},
			{i18n.GroupAdd.String(), []string{i18n.HelpAdd.String(), i18n.HelpBlank.String(), i18n.HelpComment.String(), i18n.HelpCopy.String()}},
			{i18n.GroupChange.String(), []string{i18n.HelpRename.String(), i18n.HelpToggle.String(), i18n.HelpDelete.String(), i18n.HelpMerge.String(), i18n.HelpUndo.String()}},
		}
	case modeSearch:
		return []helpGroup{{i18n.GroupSearch.String(), []string{i18n.HelpSearchHint.String(), i18n.HelpMatches.String(), i18n.HelpPick.String(), i18n.HelpCancel.String()}}}
	case modeNewKey:
		caseHint := i18n.HelpCaseUpper.String()
		if m.keepCase {
			caseHint = i18n.HelpCaseKeep.String()
		}
		return []helpGroup{
			{i18n.GroupNew.String(), []string{i18n.HelpConfirm.String(), i18n.HelpCancel.String(), i18n.HelpStartValue.String(), caseHint, i18n.HelpClear.String()}},
			{i18n.GroupImport.String(), []string{i18n.HelpImport.String()}},
		}
	case modeNewValue:
		return []helpGroup{{i18n.GroupNew.String(), []string{i18n.HelpAddValue.String(), i18n.HelpBackToKey.String(), i18n.HelpClear.String(), i18n.HelpMove.String()}}}
	}
	return []helpGroup{{i18n.GroupEdit.String(), []string{i18n.HelpConfirm.String(), i18n.HelpCancel.String(), i18n.HelpMove.String(), i18n.HelpWord.String(), i18n.HelpClear.String(), i18n.HelpDeleteWord.String()}}}
}

var helpModes = []mode{modeList, modeSearch, modeNewKey, modeNewValue, modeEditValue}

// helpLabelWidth is the width of the label column: the widest label of any
// mode plus two spaces, so the column does not move between modes.
func (m Model) helpLabelWidth() int {
	w := 0
	for _, md := range helpModes {
		for _, g := range m.helpGroups(md) {
			w = max(w, runewidth.StringWidth(g.label))
		}
	}
	return w + 2
}

// helpLines lays the help of mode md out in rows of at most w cells. Items
// that do not fit on the row move to the next, indented below the label.
func (m Model) helpLines(md mode, w int) []string {
	var out []string
	lw := m.helpLabelWidth()
	for _, g := range m.helpGroups(md) {
		cur, n := runewidth.FillRight(g.label, lw), 0
		for _, it := range g.items {
			sep := ""
			if n > 0 {
				sep = " · "
			}
			if n > 0 && runewidth.StringWidth(cur+sep+it) > w {
				out = append(out, trunc(cur, w))
				cur, n, sep = strings.Repeat(" ", lw), 0, ""
			}
			cur += sep + it
			n++
		}
		out = append(out, trunc(cur, w))
	}
	return out
}

// helpBlock returns the help rows for the current mode. Every mode takes
// the same number of rows so the list does not jump when editing starts. On
// a short terminal the help is cut so the list keeps at least 3 rows.
func (m Model) helpBlock() []string {
	w := max(10, m.width)
	n := 0
	for _, md := range helpModes {
		n = max(n, len(m.helpLines(md, w)))
	}
	lines := m.helpLines(m.mode, w)
	for len(lines) < n {
		lines = append(lines, "")
	}
	return lines[:min(len(lines), max(1, m.height-chrome-3))]
}

func (m Model) renderRow(r int, items []*envfile.Item, kw, w int) string {
	selected := r == m.cursor
	editing := selected && m.mode != modeList && m.mode != modeSearch

	if r == 0 {
		switch {
		case editing && m.mode == modeNewKey:
			prefix := "+ KEY: "
			return keyStyle.Render(prefix) + m.in.view(w-runewidth.StringWidth(prefix)-1)
		case editing && m.mode == modeNewValue:
			prefix := "+ " + trunc(m.newKey, w/2) + " = "
			return keyStyle.Render(prefix) + m.in.view(w-runewidth.StringWidth(prefix)-1)
		}
		return styleRow(i18n.NewVariableRow.String(), selected, w)
	}

	it := items[r-1]
	marker := "  "
	if selected {
		marker = "> "
	}
	if !it.Entry {
		text := marker + it.Raw
		if it.IsBlank() {
			text = marker + i18n.BlankRow.String()
		}
		switch {
		case editing:
			return marker + m.in.view(w-runewidth.StringWidth(marker)-1)
		case selected:
			return styleRow(text, true, w)
		}
		return dimStyle.Render(trunc(text, w))
	}
	key := runewidth.FillRight(trunc(it.Key, kw), kw)
	if editing && m.mode == modeRename {
		return marker + m.in.view(w-runewidth.StringWidth(marker)-1)
	}
	prefix := marker + key + " = "
	if editing {
		return marker + keyStyle.Render(key) + " = " + m.in.view(w-runewidth.StringWidth(prefix)-1)
	}
	value := it.Value
	switch {
	case m.masked && value != "":
		value = "••••••"
	case it.Multiline():
		value, _, _ = strings.Cut(value, "\n")
		value = strings.TrimSuffix(value, "\r") + i18n.MultilineTail.String()
	}
	val := trunc(value, w-runewidth.StringWidth(prefix))
	if selected {
		return styleRow(prefix+val, true, w)
	}
	return marker + keyStyle.Render(key) + " = " + val
}

func styleRow(text string, selected bool, w int) string {
	if !selected {
		return text
	}
	return selStyle.Render(runewidth.FillRight(trunc(text, w), w))
}
