package api

import (
	"fmt"
	"strings"
)

// tokenize 将查询字符串分割成标记
func tokenize(input string) ([]Token, error) {
	var tokens []Token

	// 处理特殊情况：星号表示匹配所有
	if input == "*" {
		return []Token{{Type: "value", Value: "*"}}, nil
	}

	// 预处理：处理引号包裹的短语
	var processedParts []string
	inQuote := false
	var currentPhrase strings.Builder
	var quoteChar rune
	isEscaped := false

	// 用于处理字段:引号值模式
	inFieldQuoteValue := false
	var fieldName string
	var fieldDelimiter string
	skipNext := false

	for i, char := range input {
		if skipNext {
			skipNext = false
			continue
		}
		if isEscaped {
			currentPhrase.WriteRune(char)
			isEscaped = false
			continue
		}

		if char == '\\' {
			currentPhrase.WriteRune(char)
			isEscaped = true
			continue
		}

		// 检测字段:引号值的模式
		if !inQuote && !inFieldQuoteValue && (char == ':' || char == '=' || char == '!') {
			// 检查是否是 !=
			if char == '!' && i+1 < len(input) && input[i+1] == '=' {
				fieldDelimiter = "!="
				fieldName = currentPhrase.String()
				currentPhrase.Reset()

				if fieldName == "" && len(processedParts) > 0 {
					fieldName = processedParts[len(processedParts)-1]
					processedParts = processedParts[:len(processedParts)-1]
				}

				for j := i + 2; j < len(input); j++ {
					if input[j] == ' ' || input[j] == '\t' {
						continue
					}
					if input[j] == '"' || input[j] == '\'' {
						inFieldQuoteValue = true
					}
					break
				}

				if !inFieldQuoteValue {
					if fieldName != "" {
						processedParts = append(processedParts, fieldName)
					}
					processedParts = append(processedParts, fieldDelimiter)
					fieldName = ""
					fieldDelimiter = ""
				}

				skipNext = true
				continue
			} else {
				fieldDelimiter = string(char)
				fieldName = currentPhrase.String()
				currentPhrase.Reset()

				if fieldName == "" && len(processedParts) > 0 {
					fieldName = processedParts[len(processedParts)-1]
					processedParts = processedParts[:len(processedParts)-1]
				}

				for j := i + 1; j < len(input); j++ {
					if input[j] == ' ' || input[j] == '\t' {
						continue
					}
					if input[j] == '"' || input[j] == '\'' {
						inFieldQuoteValue = true
					}
					break
				}

				if !inFieldQuoteValue {
					if fieldName != "" {
						processedParts = append(processedParts, fieldName)
					}
					processedParts = append(processedParts, fieldDelimiter)
					fieldName = ""
					fieldDelimiter = ""
				}
				continue
			}
		}

		if char == '"' || char == '\'' {
			if inQuote && char == quoteChar {
				currentPhrase.WriteRune(char)

				if inFieldQuoteValue {
					tokens = append(tokens, Token{Type: "field", Value: fieldName})
					tokens = append(tokens, Token{Type: "operator", Value: fieldDelimiter})
					tokens = append(tokens, Token{Type: "value", Value: currentPhrase.String()})

					inFieldQuoteValue = false
					fieldName = ""
					fieldDelimiter = ""
				} else {
					processedParts = append(processedParts, currentPhrase.String())
				}

				currentPhrase.Reset()
				inQuote = false
			} else if !inQuote {
				if currentPhrase.Len() > 0 {
					if !inFieldQuoteValue {
						processedParts = append(processedParts, currentPhrase.String())
					}
					currentPhrase.Reset()
				}
				currentPhrase.WriteRune(char)
				inQuote = true
				quoteChar = char
			} else {
				currentPhrase.WriteRune(char)
			}
		} else if inQuote {
			currentPhrase.WriteRune(char)
		} else if char == ' ' || char == '\t' {
			if currentPhrase.Len() > 0 {
				if !inFieldQuoteValue {
					processedParts = append(processedParts, currentPhrase.String())
				}
				currentPhrase.Reset()
			}
		} else {
			currentPhrase.WriteRune(char)
		}
	}

	if currentPhrase.Len() > 0 {
		processedParts = append(processedParts, currentPhrase.String())
	}

	// 处理分割后的部分
	i := 0
	for i < len(processedParts) {
		part := processedParts[i]

		// 处理括号
		if strings.HasPrefix(part, "(") {
			depth := 0
			groupParts := []string{}

			for j := i; j < len(processedParts); j++ {
				currentPart := processedParts[j]

				for _, ch := range currentPart {
					if ch == '(' {
						depth++
					} else if ch == ')' {
						depth--
					}
				}

				groupParts = append(groupParts, currentPart)

				if depth == 0 {
					groupStr := strings.Join(groupParts, " ")
					groupStr = strings.TrimPrefix(groupStr, "(")
					groupStr = strings.TrimSuffix(groupStr, ")")
					groupStr = strings.TrimSpace(groupStr)

					subTokens, err := tokenize(groupStr)
					if err != nil {
						return nil, err
					}

					tokens = append(tokens, Token{
						Type:      "group",
						Value:     groupStr,
						SubTokens: subTokens,
					})

					i = j + 1
					goto continueLoop
				}
			}

			return nil, fmt.Errorf("括号不匹配: %s", part)
		}

		// 处理引号包裹的内容
		if (strings.HasPrefix(part, "\"") && strings.HasSuffix(part, "\"")) ||
			(strings.HasPrefix(part, "'") && strings.HasSuffix(part, "'")) {
			tokens = append(tokens, Token{Type: "value", Value: part})
			i++
			continue
		}

		// 处理逻辑运算符
		if strings.EqualFold(part, "and") {
			tokens = append(tokens, Token{Type: "logic", Value: "and"})
			i++
			continue
		}
		if strings.EqualFold(part, "or") {
			tokens = append(tokens, Token{Type: "logic", Value: "or"})
			i++
			continue
		}
		if strings.EqualFold(part, "not") {
			tokens = append(tokens, Token{Type: "operator", Value: "not"})
			i++
			continue
		}

		// 检查字段:值格式
		if i+2 < len(processedParts) {
			if processedParts[i+1] == ":" || processedParts[i+1] == "=" || processedParts[i+1] == "!=" {
				tokens = append(tokens, Token{Type: "field", Value: part})
				tokens = append(tokens, Token{Type: "operator", Value: processedParts[i+1]})
				tokens = append(tokens, Token{Type: "value", Value: processedParts[i+2]})
				i += 3
				continue
			}
		}

		// 处理单个部分中含有操作符的情况
		for _, op := range []string{"!=", ":", "="} {
			if strings.Contains(part, op) {
				if !((strings.HasPrefix(part, "\"") && strings.HasSuffix(part, "\"")) ||
					(strings.HasPrefix(part, "'") && strings.HasSuffix(part, "'"))) {
					parts := strings.SplitN(part, op, 2)
					if len(parts) == 2 && parts[0] != "" {
						tokens = append(tokens, Token{Type: "field", Value: parts[0]})
						tokens = append(tokens, Token{Type: "operator", Value: op})
						tokens = append(tokens, Token{Type: "value", Value: parts[1]})
						goto nextPart
					}
				}
			}
		}

		// 普通关键词
		tokens = append(tokens, Token{Type: "value", Value: part})
		i++
		continue

	nextPart:
		i++

	continueLoop:
	}

	return tokens, nil
}
