package quote

import (
	"bytes"
	"fmt"
	"unicode/utf8"

	"al.essio.dev/pkg/shellescape"
)

// LuaQuote quotes a string for use as a Lua string literal.
// Note, lua only supports bytes so we transcode any utf8 into component bytes
func LuaQuote(s string) string {
	var buf bytes.Buffer
	buf.WriteByte('"')
	for i := 0; i < len(s); {
		r, width := utf8.DecodeRuneInString(s[i:])
		switch r {
		case '\\':
			buf.WriteString(`\\`)
		case '"':
			buf.WriteString(`\"`)
		case '\a':
			buf.WriteString(`\a`)
		case '\b':
			buf.WriteString(`\b`)
		case '\f':
			buf.WriteString(`\f`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		case '\v':
			buf.WriteString(`\v`)
		case 0: // null character
			buf.WriteString(`\000`) // Lua uses \000 for null
			// ... inside your for loop in LuaQuote ...
		default:
			switch {
			case r < 0x20 || r == 0x7f: // Control characters (excluding DEL)
				// Handle specific control characters like you already are
				// or use a generic escape for others.
				// For Lua, \ddd is a decimal escape, not octal.
				buf.WriteString(fmt.Sprintf(`\%d`, r))
			case r > 0x7f: // Multibyte UTF-8 characters
				// This is a multibyte character. We need to escape
				// each byte of its UTF-8 representation.
				var runeBytes [4]byte
				// Encode the rune back into a byte slice
				n := utf8.EncodeRune(runeBytes[:], r)
				// Write each byte as a decimal escape
				for i := 0; i < n; i++ {
					buf.WriteString(fmt.Sprintf(`\%d`, runeBytes[i]))
				}
			default:
				// It's a printable ASCII character
				buf.WriteRune(r)
			}
		}
		i += width
	}
	buf.WriteByte('"')
	return buf.String()
}

// ShellQuote quotes a string for use as a shell string literal.
func ShellQuote(s string) string {
	return shellescape.Quote(s)
}
