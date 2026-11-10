package db

import (
	"strings"
	"unicode"

	"github.com/api-go/cast"
)

// rewritePostgresQuery adapts the package's portable SQL convention to pgx.
// Question marks and backticks inside quoted SQL literals are preserved.
func rewritePostgresQuery(query string) string {
	var out strings.Builder
	out.Grow(len(query) + 8)
	placeholder := 0
	var quote byte
	for i := 0; i < len(query); i++ {
		ch := query[i]
		if quote != 0 {
			if quote == '`' && ch == '`' {
				out.WriteByte('"')
			} else {
				out.WriteByte(ch)
			}
			if ch == quote {
				if i+1 < len(query) && query[i+1] == quote {
					out.WriteByte(query[i+1])
					i++
				} else {
					quote = 0
				}
			}
			continue
		}
		switch ch {
		case '\'', '"':
			quote = ch
			out.WriteByte(ch)
		case '`':
			quote = '`'
			out.WriteByte('"')
		case '?':
			placeholder++
			out.WriteByte('$')
			out.WriteString(cast.String(placeholder))
		default:
			if isSQLWordByte(ch) {
				start := i
				for i+1 < len(query) && isSQLWordByte(query[i+1]) {
					i++
				}
				word := query[start : i+1]
				if postgresIdentifierNeedsQuote(word) {
					out.WriteByte('"')
					out.WriteString(word)
					out.WriteByte('"')
				} else {
					out.WriteString(word)
				}
			} else {
				out.WriteByte(ch)
			}
		}
	}
	return out.String()
}

func isSQLWordByte(ch byte) bool {
	return ch == '_' || ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9'
}

func postgresIdentifierNeedsQuote(word string) bool {
	hasUpper, hasLower := false, false
	for _, ch := range word {
		hasUpper = hasUpper || unicode.IsUpper(ch)
		hasLower = hasLower || unicode.IsLower(ch)
	}
	return hasUpper && hasLower
}

func (db *DB) prepareQuery(query string) string {
	if db != nil && db.Config != nil && isPostgresType(db.Config.Type) {
		return rewritePostgresQuery(query)
	}
	return query
}
