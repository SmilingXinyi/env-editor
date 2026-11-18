package tui

import "testing"

func TestWordJump(t *testing.T) {
	in := newInput("foo bar,baz.qux")
	var stops []int
	for in.pos > 0 {
		in.wordLeft()
		stops = append(stops, in.pos)
	}
	want := []int{12, 8, 4, 0}
	if len(stops) != len(want) {
		t.Fatalf("left stops %v, want %v", stops, want)
	}
	for i := range want {
		if stops[i] != want[i] {
			t.Fatalf("left stops %v, want %v", stops, want)
		}
	}

	stops = nil
	for in.pos < len(in.text) {
		in.wordRight()
		stops = append(stops, in.pos)
	}
	want = []int{3, 7, 11, 15}
	for i := range want {
		if stops[i] != want[i] {
			t.Fatalf("right stops %v, want %v", stops, want)
		}
	}
}

func TestInsertStripsNewlines(t *testing.T) {
	in := newInput("ab")
	in.left()
	in.insert("x\r\ny\n")
	if in.String() != "axyb" || in.pos != 3 {
		t.Fatalf("got %q pos %d", in.String(), in.pos)
	}
}

func TestDeleteAndClear(t *testing.T) {
	in := newInput("héllo wörld")
	in.deleteWordLeft()
	if in.String() != "héllo " {
		t.Fatalf("got %q", in.String())
	}
	in.backspace()
	in.home()
	in.deleteForward()
	if in.String() != "éllo" {
		t.Fatalf("got %q", in.String())
	}
	in.clear()
	if in.String() != "" || in.pos != 0 {
		t.Fatal("clear failed")
	}
}
