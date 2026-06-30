package sqltext

import (
	"errors"
	"strings"
	"unicode"

	"github.com/xwb1989/sqlparser"
)

var errEmptyStatement = errors.New("SQL statement is empty")

var keywords = map[string]struct{}{
	"ADD": {}, "ALL": {}, "ALTER": {}, "AND": {}, "AS": {}, "ASC": {}, "BETWEEN": {}, "BY": {},
	"CASE": {}, "CREATE": {}, "DELETE": {}, "DESC": {}, "DISTINCT": {}, "DROP": {}, "ELSE": {},
	"EXISTS": {}, "FROM": {}, "GROUP": {}, "HAVING": {}, "IN": {}, "INNER": {}, "INSERT": {},
	"INTO": {}, "IS": {}, "JOIN": {}, "LEFT": {}, "LIKE": {}, "LIMIT": {}, "NOT": {}, "NULL": {},
	"ON": {}, "OR": {}, "ORDER": {}, "OUTER": {}, "RIGHT": {}, "SELECT": {}, "SET": {}, "TABLE": {},
	"THEN": {}, "TRUNCATE": {}, "UNION": {}, "UPDATE": {}, "VALUES": {}, "WHEN": {}, "WHERE": {},
	"WITH": {},
}

func Prepare(statement string) (string, error) {
	normalised := UppercaseKeywords(statement)
	if err := Validate(normalised); err != nil {
		return normalised, err
	}

	return normalised, nil
}

func Validate(statement string) error {
	trimmed := strings.TrimSpace(statement)
	if trimmed == "" {
		return errEmptyStatement
	}

	_, err := sqlparser.Parse(strings.TrimSuffix(trimmed, ";"))
	return err
}

func UppercaseKeywords(statement string) string {
	var builder strings.Builder
	for index := 0; index < len(statement); {
		ch := statement[index]
		switch ch {
		case '\'', '"', '`':
			next := copyQuoted(&builder, statement, index, ch)
			index = next
		case '[':
			next := copyBracketedIdentifier(&builder, statement, index)
			index = next
		case '-':
			if index+1 < len(statement) && statement[index+1] == '-' {
				next := copyLineComment(&builder, statement, index)
				index = next
				continue
			}
			builder.WriteByte(ch)
			index++
		case '/':
			if index+1 < len(statement) && statement[index+1] == '*' {
				next := copyBlockComment(&builder, statement, index)
				index = next
				continue
			}
			builder.WriteByte(ch)
			index++
		default:
			if isIdentStart(rune(ch)) {
				next := readIdentifier(statement, index)
				word := statement[index:next]
				upper := strings.ToUpper(word)
				if _, ok := keywords[upper]; ok {
					builder.WriteString(upper)
				} else {
					builder.WriteString(word)
				}
				index = next
			} else {
				builder.WriteByte(ch)
				index++
			}
		}
	}

	return builder.String()
}

func copyQuoted(builder *strings.Builder, input string, start int, quote byte) int {
	builder.WriteByte(quote)
	for index := start + 1; index < len(input); index++ {
		builder.WriteByte(input[index])
		if input[index] != quote {
			continue
		}
		if index+1 < len(input) && input[index+1] == quote {
			index++
			builder.WriteByte(input[index])
			continue
		}
		return index + 1
	}

	return len(input)
}

func copyBracketedIdentifier(builder *strings.Builder, input string, start int) int {
	builder.WriteByte(input[start])
	for index := start + 1; index < len(input); index++ {
		builder.WriteByte(input[index])
		if input[index] == ']' {
			return index + 1
		}
	}

	return len(input)
}

func copyLineComment(builder *strings.Builder, input string, start int) int {
	for index := start; index < len(input); index++ {
		builder.WriteByte(input[index])
		if input[index] == '\n' {
			return index + 1
		}
	}

	return len(input)
}

func copyBlockComment(builder *strings.Builder, input string, start int) int {
	builder.WriteString("/*")
	for index := start + 2; index < len(input); index++ {
		builder.WriteByte(input[index])
		if input[index] == '/' && index > start+2 && input[index-1] == '*' {
			return index + 1
		}
	}

	return len(input)
}

func readIdentifier(input string, start int) int {
	for index := start; index < len(input); index++ {
		if !isIdentPart(rune(input[index])) {
			return index
		}
	}

	return len(input)
}

func isIdentStart(ch rune) bool {
	return ch == '_' || unicode.IsLetter(ch)
}

func isIdentPart(ch rune) bool {
	return ch == '_' || unicode.IsLetter(ch) || unicode.IsDigit(ch)
}
