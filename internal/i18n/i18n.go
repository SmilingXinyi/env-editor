// Package i18n holds the texts shown to the user, in English and Chinese.
package i18n

import (
	"fmt"
	"os"
	"strings"
)

// Lang is a supported language.
type Lang string

const (
	EN Lang = "en"
	ZH Lang = "zh"
)

var current = EN

// Set selects the language of all messages.
func Set(l Lang) { current = l }

// Current returns the selected language.
func Current() Lang { return current }

// Detect picks the language from the locale variables, in the order the C
// library reads them: the first one that is set decides. A Chinese locale
// (zh_CN.UTF-8, zh_TW, ...) gives Chinese, anything else English.
func Detect(getenv func(string) string) Lang {
	for _, name := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := getenv(name); v != "" {
			if strings.HasPrefix(strings.ToLower(v), "zh") {
				return ZH
			}
			return EN
		}
	}
	return EN
}

// Init selects the language from the environment.
func Init() { Set(Detect(os.Getenv)) }

// Msg is one text in every supported language. Formatting verbs must be the
// same in all of them.
type Msg struct{ en, zh string }

// all lists every message, for the tests.
var all []Msg

func def(en, zh string) Msg {
	m := Msg{en, zh}
	all = append(all, m)
	return m
}

// String returns the text in the selected language.
func (m Msg) String() string {
	if current == ZH {
		return m.zh
	}
	return m.en
}

// Fmt formats the text like fmt.Sprintf.
func (m Msg) Fmt(args ...any) string { return fmt.Sprintf(m.String(), args...) }
