package api

import (
	"fmt"
	"strings"
)

// buildQuery 根据标记构建查询语句
func buildQuery(tokens []Token) (map[string]interface{}, error) {
	if len(tokens) == 0 {
		return nil, fmt.Errorf("空的查询标记")
	}

	// 处理特殊情况：星号表示匹配所有
	if len(tokens) == 1 && tokens[0].Type == "value" && tokens[0].Value == "*" {
		return map[string]interface{}{
			"match_all": map[string]interface{}{},
		}, nil
	}

	// 处理单个 group
	if len(tokens) == 1 && tokens[0].Type == "group" {
		return buildQuery(tokens[0].SubTokens)
	}

	// 处理单个值
	if len(tokens) == 1 && tokens[0].Type == "value" {
		value := tokens[0].Value
		value = processEscapedChars(value)

		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
			(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			quotedValue := value[1 : len(value)-1]
			return map[string]interface{}{
				"multi_match": map[string]interface{}{
					"query":  quotedValue,
					"type":   "phrase",
					"fields": []string{"*"},
				},
			}, nil
		}

		return map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  value,
				"type":   "phrase",
				"fields": []string{"*"},
			},
		}, nil
	}

	// 处理字段查询
	if len(tokens) == 3 && tokens[0].Type == "field" {
		field := tokens[0].Value
		operator := tokens[1].Value
		value := tokens[2].Value
		value = processEscapedChars(value)

		if value == "*" {
			if operator == ":" || operator == "=" {
				return map[string]interface{}{
					"exists": map[string]interface{}{
						"field": field,
					},
				}, nil
			} else if operator == "!=" {
				return map[string]interface{}{
					"bool": map[string]interface{}{
						"must_not": []interface{}{
							map[string]interface{}{
								"exists": map[string]interface{}{
									"field": field,
								},
							},
						},
					},
				}, nil
			}
		}

		switch operator {
		case ":", "=":
			if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
				(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
				value = value[1 : len(value)-1]
			}
			return map[string]interface{}{
				"match_phrase": map[string]interface{}{
					field: value,
				},
			}, nil
		case "!=":
			if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
				(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
				value = value[1 : len(value)-1]
			}
			return map[string]interface{}{
				"bool": map[string]interface{}{
					"must_not": []interface{}{
						map[string]interface{}{
							"match_phrase": map[string]interface{}{
								field: value,
							},
						},
					},
				},
			}, nil
		}
	}

	// 处理复杂查询
	var boolQuery = map[string]interface{}{
		"must":     []interface{}{},
		"should":   []interface{}{},
		"must_not": []interface{}{},
	}

	// 检查是否是顶层OR查询
	isTopLevelOr := false
	for i := 0; i < len(tokens); i++ {
		if tokens[i].Type == "logic" && strings.EqualFold(tokens[i].Value, "or") {
			isTopLevelOr = true
			break
		}
	}

	if isTopLevelOr {
		return buildOrQuery(tokens)
	}

	// 处理其他查询类型
	var currentClause []interface{}
	var currentOperator string = "must"
	var isNot bool = false

	for i := 0; i < len(tokens); i++ {
		token := tokens[i]

		switch token.Type {
		case "logic":
			if len(currentClause) > 0 {
				if isNot {
					boolQuery["must_not"] = append(boolQuery["must_not"].([]interface{}), currentClause...)
					isNot = false
				} else {
					boolQuery[currentOperator] = append(boolQuery[currentOperator].([]interface{}), currentClause...)
				}
				currentClause = []interface{}{}
			}

			if strings.EqualFold(token.Value, "or") {
				currentOperator = "should"
			} else {
				currentOperator = "must"
			}

		case "operator":
			if token.Value == "not" || token.Value == "!" {
				isNot = true
			}

		case "field":
			if i+2 < len(tokens) && (tokens[i+1].Type == "operator" || tokens[i+1].Value == ":" || tokens[i+1].Value == "=" || tokens[i+1].Value == "!=") {
				field := token.Value
				operator := tokens[i+1].Value
				value := tokens[i+2].Value
				value = processEscapedChars(value)

				var query map[string]interface{}

				if value == "*" {
					if operator == ":" || operator == "=" {
						query = map[string]interface{}{
							"exists": map[string]interface{}{
								"field": field,
							},
						}
					} else if operator == "!=" {
						query = map[string]interface{}{
							"bool": map[string]interface{}{
								"must_not": []interface{}{
									map[string]interface{}{
										"exists": map[string]interface{}{
											"field": field,
										},
									},
								},
							},
						}
					}
				} else {
					switch operator {
					case ":", "=":
						if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
							(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
							value = value[1 : len(value)-1]
						}
						query = map[string]interface{}{
							"match_phrase": map[string]interface{}{
								field: value,
							},
						}
					case "!=":
						if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
							(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
							value = value[1 : len(value)-1]
						}
						query = map[string]interface{}{
							"bool": map[string]interface{}{
								"must_not": []interface{}{
									map[string]interface{}{
										"match_phrase": map[string]interface{}{
											field: value,
										},
									},
								},
							},
						}
					}
				}

				if isNot {
					boolQuery["must_not"] = append(boolQuery["must_not"].([]interface{}), query)
					isNot = false
				} else {
					currentClause = append(currentClause, query)
				}

				i += 2
			}

		case "group":
			groupQuery, err := buildQuery(token.SubTokens)
			if err != nil {
				return nil, err
			}

			if isNot {
				boolQuery["must_not"] = append(boolQuery["must_not"].([]interface{}), groupQuery)
				isNot = false
			} else {
				currentClause = append(currentClause, groupQuery)
			}

		case "value":
			value := token.Value
			value = processEscapedChars(value)

			if value == "*" {
				query := map[string]interface{}{
					"match_all": map[string]interface{}{},
				}
				if !isNot {
					currentClause = append(currentClause, query)
				}
				isNot = false
			} else {
				if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
					(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
					quotedValue := value[1 : len(value)-1]
					query := map[string]interface{}{
						"multi_match": map[string]interface{}{
							"query":  quotedValue,
							"type":   "phrase",
							"fields": []string{"*"},
						},
					}
					if isNot {
						boolQuery["must_not"] = append(boolQuery["must_not"].([]interface{}), query)
						isNot = false
					} else {
						currentClause = append(currentClause, query)
					}
				} else {
					query := map[string]interface{}{
						"multi_match": map[string]interface{}{
							"query":  value,
							"type":   "phrase",
							"fields": []string{"*"},
						},
					}
					if isNot {
						boolQuery["must_not"] = append(boolQuery["must_not"].([]interface{}), query)
						isNot = false
					} else {
						currentClause = append(currentClause, query)
					}
				}
			}
		}
	}

	// 处理最后一个子句
	if len(currentClause) > 0 {
		if isNot {
			boolQuery["must_not"] = append(boolQuery["must_not"].([]interface{}), currentClause...)
		} else {
			boolQuery[currentOperator] = append(boolQuery[currentOperator].([]interface{}), currentClause...)
		}
	}

	// 清理空的子句
	if len(boolQuery["must"].([]interface{})) == 0 {
		delete(boolQuery, "must")
	}
	if len(boolQuery["should"].([]interface{})) == 0 {
		delete(boolQuery, "should")
	}
	if len(boolQuery["must_not"].([]interface{})) == 0 {
		delete(boolQuery, "must_not")
	}

	return map[string]interface{}{
		"bool": boolQuery,
	}, nil
}

// buildOrQuery 构建OR逻辑查询
func buildOrQuery(tokens []Token) (map[string]interface{}, error) {
	var parts [][]Token
	var currentPart []Token

	for i := 0; i < len(tokens); i++ {
		if tokens[i].Type == "logic" && strings.EqualFold(tokens[i].Value, "or") {
			if len(currentPart) > 0 {
				parts = append(parts, currentPart)
				currentPart = []Token{}
			}
		} else {
			currentPart = append(currentPart, tokens[i])
		}
	}

	if len(currentPart) > 0 {
		parts = append(parts, currentPart)
	}

	var shouldClauses []interface{}

	for _, part := range parts {
		if len(part) == 0 {
			continue
		}

		var clause map[string]interface{}
		var err error

		if len(part) == 1 && part[0].Type == "group" {
			clause, err = buildQuery(part[0].SubTokens)
			if err != nil {
				return nil, err
			}
		} else if len(part) == 1 && part[0].Type == "value" {
			value := part[0].Value
			value = processEscapedChars(value)

			if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
				(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
				quotedValue := value[1 : len(value)-1]
				clause = map[string]interface{}{
					"multi_match": map[string]interface{}{
						"query":  quotedValue,
						"type":   "phrase",
						"fields": []string{"*"},
					},
				}
			} else {
				clause = map[string]interface{}{
					"multi_match": map[string]interface{}{
						"query":  value,
						"type":   "phrase",
						"fields": []string{"*"},
					},
				}
			}
		} else if len(part) == 3 && part[0].Type == "field" {
			field := part[0].Value
			operator := part[1].Value
			value := part[2].Value
			value = processEscapedChars(value)

			if value == "*" {
				if operator == ":" || operator == "=" {
					clause = map[string]interface{}{
						"exists": map[string]interface{}{
							"field": field,
						},
					}
				} else if operator == "!=" {
					clause = map[string]interface{}{
						"bool": map[string]interface{}{
							"must_not": []interface{}{
								map[string]interface{}{
									"exists": map[string]interface{}{
										"field": field,
									},
								},
							},
						},
					}
				}
			} else {
				switch operator {
				case ":", "=":
					if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
						(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
						value = value[1 : len(value)-1]
					}
					clause = map[string]interface{}{
						"match_phrase": map[string]interface{}{
							field: value,
						},
					}
				case "!=":
					if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
						(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
						value = value[1 : len(value)-1]
					}
					clause = map[string]interface{}{
						"bool": map[string]interface{}{
							"must_not": []interface{}{
								map[string]interface{}{
									"match_phrase": map[string]interface{}{
										field: value,
									},
								},
							},
						},
					}
				}
			}
		} else {
			clause, err = buildQuery(part)
			if err != nil {
				return nil, err
			}
		}

		if clause != nil {
			shouldClauses = append(shouldClauses, clause)
		}
	}

	return map[string]interface{}{
		"bool": map[string]interface{}{
			"should":               shouldClauses,
			"minimum_should_match": 1,
		},
	}, nil
}
