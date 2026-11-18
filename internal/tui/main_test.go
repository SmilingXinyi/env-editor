package tui

import (
	"os"
	"testing"

	"env-editor/internal/i18n"
)

func TestMain(m *testing.M) {
	i18n.Set(i18n.ZH)
	os.Exit(m.Run())
}
