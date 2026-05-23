package formatter

import (
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
)

// big5BackslashPlaceholder is a Unicode Private Use Area codepoint used to
// represent the extra 0x5C byte that fix_big5_escape.py inserts after Big5
// characters whose second byte equals 0x5C (e.g. 功 U+529F = [0xa5 0x5c],
// 蓋 = [0xbb 0x5c], 許 = [0xb3 0x5c]).
//
// During Big5→UTF-8 decoding the sequence [b1, 0x5C, 0x5C] is converted to
// [rune(功), big5BackslashPlaceholder].  The placeholder is a non-backslash
// non-ASCII codepoint, so the ANTLR grammar's SChar rule accepts it without
// error and the formatter passes it through unchanged.
//
// During UTF-8→Big5 re-encoding the placeholder is written back as a single
// 0x5C byte, reconstructing the original [b1, 0x5C, 0x5C] sequence.
//
// Choosing U+E001 (first Private Use Area slot after U+E000):
//   - Not encoded by any Big5 code-point, so it cannot appear in source files.
//   - Not a special character in the ANTLR grammar or in Go strings.
const big5BackslashPlaceholder = ''

// decodeIfBig5 converts Big5-encoded bytes to a UTF-8 string.
// If the input is already valid UTF-8 it is returned unchanged and
// wasEncoded is false.
//
// Big5 "許/功/蓋" fix: the sequence [b1, 0x5C, 0x5C] (a Big5 char whose
// second byte is 0x5C followed by the extra 0x5C from fix_big5_escape.py) is
// decoded as [rune + big5BackslashPlaceholder] rather than [rune + '\'].
// This prevents ANTLR from seeing a bare '\' inside a string literal, which
// would cause a spurious token-recognition error and potentially corrupt the
// string on re-encode.
func decodeIfBig5(data []byte) (utf8str string, wasEncoded bool, err error) {
	if utf8.Valid(data) {
		return string(data), false, nil
	}

	// Fast path: if no [b1>=0x81, 0x5C, 0x5C] pattern exists, use the
	// standard stream decoder (avoids per-pair allocations).
	hasFix := false
	for i := 0; i+2 < len(data); i++ {
		if data[i] >= 0x81 && data[i] <= 0xFE && data[i+1] == 0x5C && data[i+2] == 0x5C {
			hasFix = true
			break
		}
	}
	if !hasFix {
		decoded, _, err := transform.String(traditionalchinese.Big5.NewDecoder(), string(data))
		if err != nil {
			return string(data), false, err
		}
		return decoded, true, nil
	}

	// Slow path: byte-by-byte scan so we can intercept [b1, 0x5C, 0x5C].
	var buf strings.Builder
	buf.Grow(len(data))
	for i := 0; i < len(data); {
		b := data[i]
		if b >= 0x81 && b <= 0xFE && i+1 < len(data) {
			pair := data[i : i+2]
			decoded, _, e := transform.String(traditionalchinese.Big5.NewDecoder(), string(pair))
			if e == nil && len(decoded) > 0 {
				buf.WriteString(decoded)
				secondByte := data[i+1]
				i += 2
				if secondByte == 0x5C && i < len(data) && data[i] == 0x5C {
					// Extra 0x5C from fix_big5_escape.py: substitute placeholder
					// so ANTLR never sees a bare '\' in a string literal.
					buf.WriteRune(big5BackslashPlaceholder)
					i++
				}
				continue
			}
		}
		buf.WriteByte(b)
		i++
	}
	return buf.String(), true, nil
}

// buildRoundTripMap scans Big5 bytes and records, for each decoded rune, the
// first Big5 byte pair that produced it.  This lets encodeToBig5Canonical
// reproduce the exact same bytes as the original file.
func buildRoundTripMap(data []byte) map[rune][]byte {
	m := make(map[rune][]byte)
	for i := 0; i < len(data); {
		b := data[i]
		if b < 0x80 {
			i++
			continue
		}
		// Big5 double-byte: consume 2 bytes
		if i+1 >= len(data) {
			break
		}
		pair := data[i : i+2]
		decoded, _, err := transform.String(traditionalchinese.Big5.NewDecoder(), string(pair))
		if err == nil && len(decoded) > 0 {
			r, _ := utf8.DecodeRuneInString(decoded)
			if r != utf8.RuneError {
				if _, exists := m[r]; !exists {
					cp := make([]byte, 2)
					copy(cp, pair)
					m[r] = cp
				}
			}
		}
		i += 2
	}
	return m
}

// encodeToBig5 converts a UTF-8 string back to Big5 bytes, using rtMap to
// preserve the original code-point positions for any rune that was seen in
// the source file.  Pass a nil rtMap to use the Go encoder's default mapping.
func encodeToBig5(src string) ([]byte, error) {
	encoded, _, err := transform.String(traditionalchinese.Big5.NewEncoder(), src)
	return []byte(encoded), err
}

func encodeToBig5Canonical(src string, rtMap map[rune][]byte) ([]byte, error) {
	if len(rtMap) == 0 {
		return encodeToBig5(src)
	}
	out := make([]byte, 0, len(src))
	for _, r := range src {
		// Placeholder inserted by decodeIfBig5 for the extra 0x5C from
		// fix_big5_escape.py: write back as a literal backslash byte.
		if r == big5BackslashPlaceholder {
			out = append(out, 0x5C)
			continue
		}
		if r < 0x80 {
			out = append(out, byte(r))
			continue
		}
		if orig, ok := rtMap[r]; ok {
			out = append(out, orig...)
			continue
		}
		// Fall back to standard encoder for runes not seen in original.
		s, _, err := transform.String(traditionalchinese.Big5.NewEncoder(), string(r))
		if err != nil {
			return nil, err
		}
		out = append(out, []byte(s)...)
	}
	return out, nil
}
