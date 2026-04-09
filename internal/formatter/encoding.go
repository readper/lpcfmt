package formatter

import (
	"unicode/utf8"

	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
)

// decodeIfBig5 converts Big5-encoded bytes to a UTF-8 string.
// If the input is already valid UTF-8, it is returned unchanged and
// wasEncoded is false.
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

// encodeToBig5 converts a UTF-8 string back to Big5 bytes.
func encodeToBig5(src string) ([]byte, error) {
	encoded, _, err := transform.String(traditionalchinese.Big5.NewEncoder(), src)
	return []byte(encoded), err
}
