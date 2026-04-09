package formatter

// Format formats LPC source code.
//
// It handles two MudOS-specific concerns before delegating to the ANTLR
// formatter:
//
//  1. Big5 encoding: if the source is not valid UTF-8 it is assumed to be
//     Big5 and converted to UTF-8 for parsing.  The result is converted back
//     to Big5 so the file encoding is preserved.
//
//  2. @MARKER heredoc literals: the ANTLR grammar does not recognise the
//     LPC heredoc syntax.  Heredocs are extracted and replaced with string
//     literal placeholders before formatting and restored afterwards.
func Format(src string) (string, error) {
	// 1. Detect and decode Big5 if needed.
	utf8src, wasBig5, err := decodeIfBig5([]byte(src))
	if err != nil {
		// Cannot decode — return original unchanged.
		return src, nil
	}

	// 2. Extract heredocs.
	processed, heredocs := extractHeredocs(utf8src)

	// 3. Format with ANTLR.
	formatted, err := FormatWithANTLR(processed)
	if err != nil {
		return src, nil
	}

	// 4. Restore heredocs.
	result := restoreHeredocs(formatted, heredocs)

	// 5. Re-encode to Big5 if the original file was Big5.
	if wasBig5 {
		big5bytes, err := encodeToBig5(result)
		if err != nil {
			return src, nil
		}
		return string(big5bytes), nil
	}

	return result, nil
}
