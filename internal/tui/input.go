package tui

import (
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// input is a single-line text buffer with a cursor.
type input struct {
	text []rune
	pos  int
}

func newInput(s string) input {
	r := []rune(s)
	return input{text: r, pos: len(r)}
}

func (i *input) String() string { return string(i.text) }

// insert adds s at the cursor. Line breaks and other control characters are
// dropped so that a value always stays on one line.
func (i *input) insert(s string) {
	var in []rune
	for _, r := range s {
		if !unicode.IsControl(r) {
			in = append(in, r)
		}
	}
	if len(in) == 0 {
		return
	}
	out := make([]rune, 0, len(i.text)+len(in))
	out = append(out, i.text[:i.pos]...)
	out = append(out, in...)
	out = append(out, i.text[i.pos:]...)
	i.text = out
	i.pos += len(in)
}

func (i *input) backspace() {
	if i.pos == 0 {
		return
	}
	i.text = append(i.text[:i.pos-1], i.text[i.pos:]...)
	i.pos--
}

func (i *input) deleteForward() {
	if i.pos >= len(i.text) {
		return
	}
	i.text = append(i.text[:i.pos], i.text[i.pos+1:]...)
}

func (i *input) clear() {
	i.text = nil
	i.pos = 0
}

func (i *input) left() {
	if i.pos > 0 {
		i.pos--
	}
}

func (i *input) right() {
	if i.pos < len(i.text) {
		i.pos++
	}
}

func (i *input) home() { i.pos = 0 }
func (i *input) end()  { i.pos = len(i.text) }

func isSep(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }

func (i *input) wordStart() int {
	p := i.pos
	for p > 0 && isSep(i.text[p-1]) {
		p--
	}
	for p > 0 && !isSep(i.text[p-1]) {
		p--
	}
	return p
}

func (i *input) wordEnd() int {
	p := i.pos
	for p < len(i.text) && isSep(i.text[p]) {
		p++
	}
	for p < len(i.text) && !isSep(i.text[p]) {
		p++
	}
	return p
}

func (i *input) wordLeft()  { i.pos = i.wordStart() }
func (i *input) wordRight() { i.pos = i.wordEnd() }

func (i *input) deleteWordLeft() {
	p := i.wordStart()
	i.text = append(i.text[:p], i.text[i.pos:]...)
	i.pos = p
}

var cursorStyle = lipgloss.NewStyle().Reverse(true)

// handleKey applies an editing key and reports whether the text changed.
func (i *input) handleKey(msg tea.KeyMsg) bool {
	before := i.String()
	switch msg.String() {
	case "left", "ctrl+b":
		i.left()
	case "right", "ctrl+f":
		i.right()
	case "home", "ctrl+a":
		i.home()
	case "end", "ctrl+e":
		i.end()
	case "ctrl+left", "alt+left", "alt+b":
		i.wordLeft()
	case "ctrl+right", "alt+right", "alt+f":
		i.wordRight()
	case "backspace", "ctrl+h":
		i.backspace()
	case "delete", "ctrl+d":
		i.deleteForward()
	case "ctrl+u":
		i.clear()
	case "alt+backspace", "ctrl+w":
		i.deleteWordLeft()
	default:
		switch msg.Type {
		case tea.KeySpace:
			i.insert(" ")
		case tea.KeyRunes:
			if !msg.Alt {
				i.insert(string(msg.Runes))
			}
		}
	}
	return i.String() != before
}

// view renders the text within width cells, scrolling horizontally so the
// cursor stays visible. The cursor is drawn as a reversed cell.
func (i *input) view(width int) string {
	if width < 2 {
		width = 2
	}
	cursorW := 1
	if i.pos < len(i.text) {
		cursorW = max(1, runewidth.RuneWidth(printable(i.text[i.pos])))
	}
	// Walk back from the cursor as far as the width allows.
	start, used := i.pos, cursorW
	for start > 0 {
		w := runewidth.RuneWidth(printable(i.text[start-1]))
		if used+w > width {
			break
		}
		used += w
		start--
	}
	var b strings.Builder
	used = 0
	for idx := start; idx < len(i.text); idx++ {
		r := printable(i.text[idx])
		w := runewidth.RuneWidth(r)
		if used+w > width {
			break
		}
		if idx == i.pos {
			b.WriteString(cursorStyle.Render(string(r)))
		} else {
			b.WriteRune(r)
		}
		used += w
	}
	if i.pos >= len(i.text) {
		b.WriteString(cursorStyle.Render(" "))
	}
	return b.String()
}
