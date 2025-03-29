package utils

import (
	"regexp"
	"strings"
)

func EscapeMarkdown(s string) (o string) {
	o = s
	o = escapeEscape(o)
	o = EscapeCodeBlock(o)
	o = escapeItalicOrBoldOrUnderline(o)
	o = escapeStrikethrough(o)
	o = escapeSpoiler(o)
	o = escapeMaskedLink(o)

	return
}

func EscapeCodeBlock(s string) string {
	return strings.ReplaceAll(s, "`", "\u02cb")
}

func escapeItalicOrBoldOrUnderline(s string) string {
	pass1 := strings.ReplaceAll(s, `_`, `\_`)
	return strings.ReplaceAll(pass1, `*`, `\*`)
}

func escapeStrikethrough(s string) string {
	return strings.ReplaceAll(s, "~~", `\~\~`)
}

func escapeSpoiler(s string) string {
	return strings.ReplaceAll(s, "||", `\|\|`)
}

func escapeEscape(s string) string {
	return strings.ReplaceAll(s, `\`, `\\`)
}

var (
	maskedLinkRegex = regexp.MustCompile(`\[.+]\(.+\)`)
)

func escapeMaskedLink(s string) string {
	return maskedLinkRegex.ReplaceAllString(s, `\$0`)
}
