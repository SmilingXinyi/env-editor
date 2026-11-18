package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"env-editor/internal/envfile"
	"env-editor/internal/i18n"
)

func press(m Model, keys ...tea.KeyMsg) Model {
	for _, k := range keys {
		next, _ := m.Update(k)
		m = next.(Model)
	}
	return m
}

func runes(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

var (
	down  = tea.KeyMsg{Type: tea.KeyDown}
	enter = tea.KeyMsg{Type: tea.KeyEnter}
	esc   = tea.KeyMsg{Type: tea.KeyEsc}
)

func newTestModel(t *testing.T, content string) (Model, string) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	doc, err := envfile.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return New(path, doc, false), path
}

func read(t *testing.T, path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestInsertBlankAboveSelected(t *testing.T) {
	m, path := newTestModel(t, "A=1\nB=2\n")
	m = press(m, down, down, runes("n"))
	if got := read(t, path); got != "A=1\n\nB=2\n" {
		t.Fatalf("got %q", got)
	}
	if m.cursor != 3 {
		t.Fatalf("cursor %d should stay on B", m.cursor)
	}
}

func TestInsertCommentAboveSelected(t *testing.T) {
	m, path := newTestModel(t, "A=1\nB=2\n")
	m = press(m, down, down, runes("c"), runes("db host"), enter)
	if got := read(t, path); got != "A=1\n# db host\nB=2\n" {
		t.Fatalf("got %q", got)
	}
	if m.mode != modeList || m.cursor != 3 {
		t.Fatalf("mode %v cursor %d", m.mode, m.cursor)
	}
}

func TestInsertCommentCancel(t *testing.T) {
	m, path := newTestModel(t, "A=1\n")
	m = press(m, runes("c"), runes("x"), esc)
	if got := read(t, path); got != "A=1\n" {
		t.Fatalf("file changed: %q", got)
	}
	if len(m.doc.Items) != 1 || m.cursor != 0 || m.mode != modeList {
		t.Fatalf("items %d cursor %d mode %v", len(m.doc.Items), m.cursor, m.mode)
	}
}

func TestEditExistingComment(t *testing.T) {
	m, path := newTestModel(t, "# old\nA=1\n")
	m = press(m, down, enter, tea.KeyMsg{Type: tea.KeyCtrlU}, runes("new"), enter)
	if got := read(t, path); got != "# new\nA=1\n" {
		t.Fatalf("got %q", got)
	}
}

func TestToggleComment(t *testing.T) {
	m, path := newTestModel(t, "A=1 # n\nB=2\n")
	m = press(m, down, runes("#"))
	if got := read(t, path); got != "# A=1 # n\nB=2\n" {
		t.Fatalf("got %q", got)
	}
	m = press(m, runes("#"))
	if got := read(t, path); got != "A=1 # n\nB=2\n" {
		t.Fatalf("got %q", got)
	}
	_ = m
}

func TestDeleteNeedsConfirmForEntries(t *testing.T) {
	m, path := newTestModel(t, "A=1\n\nB=2\n")
	m = press(m, down, runes("d"))
	if got := read(t, path); got != "A=1\n\nB=2\n" {
		t.Fatalf("deleted without confirmation: %q", got)
	}
	m = press(m, runes("d"))
	if got := read(t, path); got != "\nB=2\n" {
		t.Fatalf("got %q", got)
	}
	m = press(m, runes("d"))
	if got := read(t, path); got != "B=2\n" {
		t.Fatalf("blank line should delete at once: %q", got)
	}
	m = press(m, down, runes("d"), down, runes("d"))
	if got := read(t, path); got != "B=2\n" {
		t.Fatalf("other key must cancel confirmation: %q", got)
	}
}

func TestToggleCommentRefusesDuplicate(t *testing.T) {
	m, path := newTestModel(t, "# A=1\nA=2\n")
	m = press(m, down, runes("#"))
	if !m.isErr {
		t.Fatal("expected duplicate key error")
	}
	if got := read(t, path); got != "# A=1\nA=2\n" {
		t.Fatalf("got %q", got)
	}
}

func TestCopyRowCommentsOriginal(t *testing.T) {
	m, path := newTestModel(t, "A=1 # n\nB=2\n")
	m = press(m, down, runes("y"))
	if got := read(t, path); got != "# A=1 # n\nA=1 # n\nB=2\n" {
		t.Fatalf("got %q", got)
	}
	if m.cursor != 2 {
		t.Fatalf("cursor %d should stay on the variable", m.cursor)
	}
}

func TestCopyCommentKeepsOriginal(t *testing.T) {
	m, path := newTestModel(t, "# hi\n")
	m = press(m, down, runes("y"))
	if got := read(t, path); got != "# hi\n# hi\n" {
		t.Fatalf("got %q", got)
	}
}

func TestJKNavigation(t *testing.T) {
	m, _ := newTestModel(t, "A=1\nB=2\n")
	m = press(m, runes("j"), runes("j"), runes("k"))
	if m.cursor != 1 {
		t.Fatalf("cursor %d", m.cursor)
	}
}

func TestAddAboveSelected(t *testing.T) {
	m, path := newTestModel(t, "A=1\nB=2\n")
	m = press(m, down, down, runes("a"))
	if m.cursor != 0 || m.mode != modeNewKey {
		t.Fatalf("cursor %d mode %v", m.cursor, m.mode)
	}
	m = press(m, runes("C"), enter, runes("3"), enter)
	if got := read(t, path); got != "A=1\nC=3\nB=2\n" {
		t.Fatalf("got %q", got)
	}
	if m.cursor != 2 {
		t.Fatalf("cursor %d should be on the new variable", m.cursor)
	}
}

func TestNewFromTopGoesAfterLastEntry(t *testing.T) {
	m, path := newTestModel(t, "A=1\n\n# trailing\n")
	m = press(m, enter, runes("B=2"), enter)
	if got := read(t, path); got != "A=1\nB=2\n\n# trailing\n" {
		t.Fatalf("got %q", got)
	}
}

func TestCancelNewRestoresCursor(t *testing.T) {
	m, _ := newTestModel(t, "A=1\nB=2\n")
	m = press(m, down, down, runes("a"), esc)
	if m.cursor != 2 || m.mode != modeList {
		t.Fatalf("cursor %d mode %v", m.cursor, m.mode)
	}
}

func TestCollapseBlankLines(t *testing.T) {
	m, path := newTestModel(t, "A=1\n\n\n\nB=2\n\n")
	m = press(m, down, down, down, down, down, runes("f"))
	if got := read(t, path); got != "A=1\n\nB=2\n\n" {
		t.Fatalf("got %q", got)
	}
	if m.cursor != 3 {
		t.Fatalf("cursor %d should stay on B", m.cursor)
	}
	m = press(m, runes("f"))
	if m.status != "没有连续的空行" {
		t.Fatalf("status %q", m.status)
	}
}

func TestNewKeyIsUppercased(t *testing.T) {
	m, path := newTestModel(t, "")
	m = press(m, enter, runes("db_host"), enter, runes("x"), enter)
	if got := read(t, path); got != "DB_HOST=x\n" {
		t.Fatalf("got %q", got)
	}
}

func TestEqualsSwitchesToValue(t *testing.T) {
	m, path := newTestModel(t, "")
	m = press(m, enter, runes("abc"), runes("="))
	if m.mode != modeNewValue || m.newKey != "ABC" {
		t.Fatalf("mode %v key %q", m.mode, m.newKey)
	}
	m = press(m, runes("1=2"), enter)
	if got := read(t, path); got != "ABC=1=2\n" {
		t.Fatalf("got %q", got)
	}
}

func TestPasteKeyValue(t *testing.T) {
	m, path := newTestModel(t, "")
	m = press(m, enter, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("db=x y"), Paste: true}, enter)
	if got := read(t, path); got != "DB='x y'\n" {
		t.Fatalf("got %q", got)
	}
}

func TestEqualsWithInvalidKeyStays(t *testing.T) {
	m, _ := newTestModel(t, "")
	m = press(m, enter, runes("="))
	if m.mode != modeNewKey || !m.isErr {
		t.Fatalf("mode %v isErr %v", m.mode, m.isErr)
	}
}

func TestUndo(t *testing.T) {
	m, path := newTestModel(t, "A=1\nB=2\n")
	m = press(m, down, runes("d"), runes("d"), runes("f"), runes("n"))
	if got := read(t, path); got != "\nB=2\n" {
		t.Fatalf("got %q", got)
	}
	m = press(m, runes("u"))
	if got := read(t, path); got != "B=2\n" {
		t.Fatalf("after first undo %q", got)
	}
	m = press(m, runes("u"))
	if got := read(t, path); got != "A=1\nB=2\n" {
		t.Fatalf("after second undo %q", got)
	}
	if m.cursor != 1 {
		t.Fatalf("cursor %d", m.cursor)
	}
	m = press(m, runes("u"))
	if m.status != "没有可撤销的操作" {
		t.Fatalf("status %q", m.status)
	}
}

func TestUndoEditSession(t *testing.T) {
	m, path := newTestModel(t, "A=1\n")
	m = press(m, runes("c"), runes("note"), enter, runes("u"))
	if got := read(t, path); got != "A=1\n" {
		t.Fatalf("got %q", got)
	}
	if len(m.doc.Items) != 1 {
		t.Fatalf("items %d", len(m.doc.Items))
	}
}

func TestExternalChangeIsNotOverwritten(t *testing.T) {
	m, path := newTestModel(t, "A=1\n")
	if err := os.WriteFile(path, []byte("A=1\nX=9\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m = press(m, down, runes("d"), runes("d"))
	if !m.isErr || !strings.Contains(m.status, "其他程序修改") {
		t.Fatalf("status %q", m.status)
	}
	if got := read(t, path); got != "A=1\nX=9\n" {
		t.Fatalf("file overwritten: %q", got)
	}
	if len(m.doc.Items) != 1 || m.doc.Find("A") == nil {
		t.Fatal("failed change was not rolled back")
	}
	m = press(m, runes("R"))
	if m.doc.Find("X") == nil {
		t.Fatal("reload did not pick up the change")
	}
	m = press(m, runes("d"), runes("d"))
	if got := read(t, path); got != "X=9\n" {
		t.Fatalf("got %q", got)
	}
}

func TestQuitKeys(t *testing.T) {
	m, _ := newTestModel(t, "A=1\n")
	if _, cmd := m.Update(esc); cmd != nil {
		t.Fatal("esc in the list must not quit")
	}
	if _, cmd := m.Update(runes("q")); cmd == nil {
		t.Fatal("q should quit")
	}
}

func TestCtrlHIsBackspace(t *testing.T) {
	m, _ := newTestModel(t, "A=abc\n")
	m = press(m, down, enter, tea.KeyMsg{Type: tea.KeyCtrlH})
	if got := m.in.String(); got != "ab" {
		t.Fatalf("got %q", got)
	}
}

func TestEditValueKeepsMeaning(t *testing.T) {
	m, path := newTestModel(t, `P='C:\temp'`+"\n")
	m = press(m, down, enter, tea.KeyMsg{Type: tea.KeyCtrlU}, runes(`it's C:\temp`), enter)
	if !m.isErr || m.mode != modeEditValue {
		t.Fatalf("mode %v status %q", m.mode, m.status)
	}
	if got := read(t, path); got != `P='C:\temp'`+"\n" {
		t.Fatalf("got %q", got)
	}
}

func TestMultilineEntryIsReadOnly(t *testing.T) {
	m, path := newTestModel(t, "K=\"a\nb==\"\nB=1\n")
	if n := len(m.doc.Entries()); n != 2 {
		t.Fatalf("entries %d", n)
	}
	m = press(m, down, enter)
	if m.mode != modeList || !m.isErr {
		t.Fatalf("mode %v isErr %v", m.mode, m.isErr)
	}
	m = press(m, runes("#"))
	if !m.isErr {
		t.Fatal("toggling a multi-line value should fail")
	}
	m = press(m, runes("r"), tea.KeyMsg{Type: tea.KeyCtrlU}, runes("KEY"), enter)
	if got := read(t, path); got != "KEY=\"a\nb==\"\nB=1\n" {
		t.Fatalf("got %q", got)
	}
}

func TestRename(t *testing.T) {
	m, path := newTestModel(t, "export A=1 # x\nB=2\n")
	m = press(m, down, runes("r"), tea.KeyMsg{Type: tea.KeyCtrlU}, runes("B"), enter)
	if !m.isErr || m.mode != modeRename {
		t.Fatal("rename to an existing key should fail and stay in rename")
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyCtrlU}, runes("C"), enter)
	if got := read(t, path); got != "export C=1 # x\nB=2\n" {
		t.Fatalf("got %q", got)
	}
}

func TestPasteManyLinesImports(t *testing.T) {
	m, path := newTestModel(t, "A=1\n")
	paste := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("A=9\nB=2\n# x\nC='3'\n"), Paste: true}
	m = press(m, enter, paste)
	if got := read(t, path); got != "A=1\nB=2\nC='3'\n" {
		t.Fatalf("got %q", got)
	}
	if m.mode != modeList || !strings.Contains(m.status, "1 个同名变量已跳过") {
		t.Fatalf("mode %v status %q", m.mode, m.status)
	}
}

func TestPasteManyLinesIntoValueIsRefused(t *testing.T) {
	m, _ := newTestModel(t, "A=1\n")
	m = press(m, down, enter, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x\ny"), Paste: true})
	if !m.isErr || m.in.String() != "1" {
		t.Fatalf("value %q status %q", m.in.String(), m.status)
	}
}

func TestKeepCase(t *testing.T) {
	m, path := newTestModel(t, "")
	m = press(m, enter, tea.KeyMsg{Type: tea.KeyCtrlT}, runes("db_host=x"), enter)
	if got := read(t, path); got != "db_host=x\n" {
		t.Fatalf("got %q", got)
	}
}

func TestSearch(t *testing.T) {
	m, _ := newTestModel(t, "A=1\n# db settings\nDB_HOST=x\nDB_PORT=5432\n")
	m = press(m, runes("/"), runes("db_"))
	if m.cursor != 3 {
		t.Fatalf("cursor %d", m.cursor)
	}
	m = press(m, down)
	if m.cursor != 4 {
		t.Fatalf("next: cursor %d", m.cursor)
	}
	m = press(m, down)
	if m.cursor != 3 {
		t.Fatalf("wrap: cursor %d", m.cursor)
	}
	m = press(m, enter)
	if m.mode != modeList || m.cursor != 3 {
		t.Fatalf("mode %v cursor %d", m.mode, m.cursor)
	}
	m = press(m, runes("/"), runes("5432"), esc)
	if m.cursor != 3 {
		t.Fatalf("esc should restore cursor, got %d", m.cursor)
	}
}

func TestViewSanitizesAndFits(t *testing.T) {
	long := strings.Repeat("长", 100)
	m, _ := newTestModel(t, "A=\"\x1b[31mred\"\nB="+long+"\n# "+long+"\n")
	m.width, m.height = 40, 10
	v := m.View()
	if strings.Contains(v, "\x1b[31mred") {
		t.Fatal("escape sequence from the file reached the terminal")
	}
	if n := strings.Count(v, "\n") + 1; n > m.height {
		t.Fatalf("%d lines > height %d", n, m.height)
	}
	for _, line := range strings.Split(v, "\n") {
		if w := lipgloss.Width(line); w > 40 {
			t.Fatalf("line width %d > 40: %q", w, line)
		}
	}
}

func TestMask(t *testing.T) {
	m, _ := newTestModel(t, "SECRET=hunter2\n")
	m.masked = true
	if strings.Contains(m.View(), "hunter2") {
		t.Fatal("masked value shown")
	}
	m = press(m, runes("v"))
	if !strings.Contains(m.View(), "hunter2") {
		t.Fatal("value should show after v")
	}
}

func TestHelpAlwaysVisible(t *testing.T) {
	m, _ := newTestModel(t, "A=1\n")
	m.width, m.height = 100, 24
	list := m.View()
	for _, want := range []string{"重命名", "搜索", "撤销", "合并空行", "显示/隐藏值", "退出"} {
		if !strings.Contains(list, want) {
			t.Errorf("list help lacks %q", want)
		}
	}
	m = press(m, down, enter)
	edit := m.View()
	if !strings.Contains(edit, "ctrl+u 清空") || strings.Contains(edit, "Ctrl+U") {
		t.Errorf("edit help: %q", edit)
	}
	if a, b := strings.Count(list, "\n"), strings.Count(edit, "\n"); a != b || a+1 != m.height {
		t.Errorf("layout changed: list %d lines, edit %d lines, height %d", a+1, b+1, m.height)
	}
}

func TestHelpFitsSmallTerminals(t *testing.T) {
	m, _ := newTestModel(t, "A=1\nB=2\nC=3\nD=4\nE=5\nF=6\n")
	for _, size := range [][2]int{{20, 8}, {30, 12}, {60, 24}, {120, 40}} {
		m.width, m.height = size[0], size[1]
		for _, md := range []mode{modeList, modeSearch, modeNewKey, modeEditValue} {
			m.mode = md
			lines := strings.Split(m.View(), "\n")
			if len(lines) > m.height {
				t.Errorf("%v mode %v: %d lines > height", size, md, len(lines))
			}
			for _, l := range lines {
				if w := lipgloss.Width(l); w > max(10, m.width) {
					t.Errorf("%v mode %v: width %d: %q", size, md, w, l)
				}
			}
		}
	}
}

func TestEnglish(t *testing.T) {
	i18n.Set(i18n.EN)
	defer i18n.Set(i18n.ZH)

	m, _ := newTestModel(t, "A=1\n")
	m.width, m.height = 100, 24
	m = press(m, runes("u"))
	if m.status != "Nothing to undo" {
		t.Fatalf("status %q", m.status)
	}
	for _, want := range []string{"rename", "search", "undo", "merge blank lines", "quit"} {
		if !strings.Contains(m.View(), want) {
			t.Errorf("English view lacks %q", want)
		}
	}
	m = press(m, down, runes("d"))
	if !strings.Contains(m.status, "Press d again") {
		t.Errorf("status %q", m.status)
	}
}

func TestHelpFitsBothLanguages(t *testing.T) {
	defer i18n.Set(i18n.ZH)
	for _, lang := range []i18n.Lang{i18n.EN, i18n.ZH} {
		i18n.Set(lang)
		m, _ := newTestModel(t, "A=1\nB=2\nC=3\nD=4\nE=5\nF=6\n")
		for _, size := range [][2]int{{20, 8}, {30, 12}, {60, 24}, {120, 40}} {
			m.width, m.height = size[0], size[1]
			rows := -1
			for _, md := range []mode{modeList, modeSearch, modeNewKey, modeNewValue, modeEditValue} {
				m.mode = md
				lines := strings.Split(m.View(), "\n")
				if len(lines) > m.height {
					t.Errorf("%s %v mode %v: %d lines > height", lang, size, md, len(lines))
				}
				if rows >= 0 && len(lines) != rows {
					t.Errorf("%s %v mode %v: layout changed", lang, size, md)
				}
				rows = len(lines)
				for _, l := range lines {
					if w := lipgloss.Width(l); w > max(10, m.width) {
						t.Errorf("%s %v mode %v: width %d: %q", lang, size, md, w, l)
					}
				}
			}
		}
	}
}
