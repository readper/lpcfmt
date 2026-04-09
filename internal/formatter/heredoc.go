package formatter

import (
	"fmt"
	"strings"
)

const heredocPlaceholderPrefix = "__LPCFMT_HEREDOC_"

// extractHeredocs replaces LPC @MARKER...MARKER heredoc literals with
// placeholder string literals so the ANTLR formatter can parse the file.
// The original heredoc texts are returned in order so they can be restored
// after formatting.
//
// Syntax handled:
//
//	write(@HELP
//	some text
//	HELP
//	);
//
// The entire span from @MARKER through the closing MARKER\n is replaced with
// the string literal "__LPCFMT_HEREDOC_N__".
func extractHeredocs(src string) (processed string, originals []string) {
	var b strings.Builder
	b.Grow(len(src))
	i := 0
	n := len(src)

	for i < n {
		// Skip line comments — don't interpret @ inside them.
		if i+1 < n && src[i] == '/' && src[i+1] == '/' {
			for i < n && src[i] != '\n' {
				b.WriteByte(src[i])
				i++
			}
			continue
		}

		// Skip block comments.
		if i+1 < n && src[i] == '/' && src[i+1] == '*' {
			b.WriteByte(src[i])
			b.WriteByte(src[i+1])
			i += 2
			for i < n {
				if i+1 < n && src[i] == '*' && src[i+1] == '/' {
					b.WriteByte(src[i])
					b.WriteByte(src[i+1])
					i += 2
					break
				}
				b.WriteByte(src[i])
				i++
			}
			continue
		}

		// Skip string literals — @ inside them is not a heredoc.
		if src[i] == '"' {
			b.WriteByte(src[i])
			i++
			for i < n {
				if src[i] == '\\' {
					b.WriteByte(src[i])
					i++
					if i < n {
						b.WriteByte(src[i])
						i++
					}
					continue
				}
				if src[i] == '"' {
					break
				}
				b.WriteByte(src[i])
				i++
			}
			if i < n {
				b.WriteByte(src[i]) // closing "
				i++
			}
			continue
		}

		// Potential heredoc: @IDENTIFIER followed immediately by \n
		if src[i] == '@' && i+1 < n && isIdentStart(src[i+1]) {
			j := i + 1
			for j < n && isIdentChar(src[j]) {
				j++
			}
			marker := src[i+1 : j]
			if j < n && src[j] == '\n' {
				if closeEnd := findHeredocClose(src, j+1, marker); closeEnd != -1 {
					original := src[i:closeEnd]
					idx := len(originals)
					originals = append(originals, original)
					fmt.Fprintf(&b, `"%s%d__"`, heredocPlaceholderPrefix, idx)
					i = closeEnd
					continue
				}
			}
		}

		b.WriteByte(src[i])
		i++
	}

	return b.String(), originals
}

// findHeredocClose searches for a line that is exactly `marker` starting from
// position `start`. Returns the position just after the closing marker line
// (including its trailing \n), or -1 if not found.
func findHeredocClose(src string, start int, marker string) int {
	i := start
	n := len(src)
	for i < n {
		lineStart := i
		// Find end of line
		for i < n && src[i] != '\n' {
			i++
		}
		if src[lineStart:i] == marker {
			if i < n { // consume the \n
				i++
			}
			return i
		}
		if i < n {
			i++ // skip \n
		}
	}
	return -1
}

// restoreHeredocs replaces the placeholder strings inserted by extractHeredocs
// back with the original heredoc text.
func restoreHeredocs(src string, originals []string) string {
	for i, original := range originals {
		placeholder := fmt.Sprintf(`"%s%d__"`, heredocPlaceholderPrefix, i)
		src = strings.Replace(src, placeholder, original, 1)
	}
	return src
}

func isIdentStart(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || c == '_'
}

func isIdentChar(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}
