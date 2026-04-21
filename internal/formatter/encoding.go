package formatter

import (
	"unicode/utf8"

	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
)

// decodeIfBig5 converts Big5-encoded bytes to a UTF-8 string.
// If the input is already valid UTF-8, it is returned unchanged and
// wasEncoded is false.
// It also returns a round-trip map: rune -> original Big5 bytes, so that
// callers can re-encode back to the SAME Big5 code points rather than
// whatever the Go encoder prefers (which may differ for characters that have
// multiple valid Big5 positions, e.g. 包 is a55d in traditional Big5 but
// fabd in Big5-HKSCS — both decode to the same rune, but the Go encoder
// always emits the HKSCS position).
func decodeIfBig5(data []byte) (utf8str string, wasEncoded bool, err error) {
	if utf8.Valid(data) {
		return string(data), false, nil
	}
	decoded, _, err := transform.String(traditionalchinese.Big5.NewDecoder(), string(data))
	if err != nil {
		return string(data), false, err
	}
	return decoded, true, nil
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
