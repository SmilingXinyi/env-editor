package i18n

import (
	"regexp"
	"testing"
)

func env(vars map[string]string) func(string) string {
	return func(k string) string { return vars[k] }
}

func TestDetect(t *testing.T) {
	cases := []struct {
		name string
		vars map[string]string
		want Lang
	}{
		{"empty", nil, EN},
		{"zh LANG", map[string]string{"LANG": "zh_CN.UTF-8"}, ZH},
		{"zh_TW", map[string]string{"LANG": "zh_TW.UTF-8"}, ZH},
		{"en LANG", map[string]string{"LANG": "en_US.UTF-8"}, EN},
		{"other", map[string]string{"LANG": "ja_JP.UTF-8"}, EN},
		{"C locale", map[string]string{"LANG": "C"}, EN},
		{"LC_ALL wins", map[string]string{"LC_ALL": "en_US.UTF-8", "LANG": "zh_CN.UTF-8"}, EN},
		{"LC_ALL zh wins", map[string]string{"LC_ALL": "zh_CN.UTF-8", "LANG": "en_US.UTF-8"}, ZH},
		{"LC_MESSAGES before LANG", map[string]string{"LC_MESSAGES": "zh_CN", "LANG": "en_US"}, ZH},
		{"empty LC_ALL is skipped", map[string]string{"LC_ALL": "", "LANG": "zh_CN"}, ZH},
		{"upper case", map[string]string{"LANG": "ZH_CN"}, ZH},
	}
	for _, c := range cases {
		if got := Detect(env(c.vars)); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestSelectsByLanguage(t *testing.T) {
	defer Set(current)
	m := def("hello %s", "你好 %s")
	Set(EN)
	if got := m.Fmt("x"); got != "hello x" {
		t.Errorf("en: %q", got)
	}
	Set(ZH)
	if got := m.Fmt("x"); got != "你好 x" {
		t.Errorf("zh: %q", got)
	}
}

var verbs = regexp.MustCompile(`%[-+# 0-9.]*[a-zA-Z%]`)

func TestMessagesAreComplete(t *testing.T) {
	if len(all) == 0 {
		t.Fatal("no messages registered")
	}
	for _, m := range all {
		if m.en == "" || m.zh == "" {
			t.Errorf("missing translation: en %q zh %q", m.en, m.zh)
		}
		en, zh := verbs.FindAllString(m.en, -1), verbs.FindAllString(m.zh, -1)
		if len(en) != len(zh) {
			t.Errorf("format verbs differ: %q vs %q", m.en, m.zh)
			continue
		}
		for i := range en {
			if en[i] != zh[i] {
				t.Errorf("format verbs differ: %q vs %q", m.en, m.zh)
			}
		}
	}
}
