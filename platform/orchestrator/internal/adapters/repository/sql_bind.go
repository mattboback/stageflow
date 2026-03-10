package db

import (
	"strconv"
	"strings"
	"sync"
)

var bindCache sync.Map

func bindPostgresParams(query string) string {
	if cached, ok := bindCache.Load(query); ok {
		value, typeOK := cached.(string)
		if typeOK {
			return value
		}
	}

	rebuilt := rebindQuestionMarks(query)
	bindCache.Store(query, rebuilt)

	return rebuilt
}

func rebindQuestionMarks(query string) string {
	var (
		builder       strings.Builder
		argIndex      = 1
		inSingleQuote bool
	)

	builder.Grow(len(query) + 8)

	for idx := 0; idx < len(query); idx++ {
		char := query[idx]

		switch char {
		case '\'':
			builder.WriteByte(char)

			if inSingleQuote {
				if idx+1 < len(query) && query[idx+1] == '\'' {
					builder.WriteByte(query[idx+1])
					idx++

					continue
				}

				inSingleQuote = false

				continue
			}

			inSingleQuote = true
		case '?':
			if inSingleQuote {
				builder.WriteByte(char)

				continue
			}

			builder.WriteByte('$')
			builder.WriteString(strconv.Itoa(argIndex))
			argIndex++
		default:
			builder.WriteByte(char)
		}
	}

	return builder.String()
}
