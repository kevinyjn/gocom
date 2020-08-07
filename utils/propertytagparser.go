package utils

import "strings"

// ParsePropertyTagValue parse go tag value as property name and attributes
func ParsePropertyTagValue(text string) (string, map[string]string) {
	var name = ""
	var attributes = map[string]string{}
	var l = len(text)
	var c byte
	var expressionStart = 0
	var lastChar = byte(0)
	var backslashCount = 0
	var attrName string
	var backslashIndexes = []int{}
	for i := 0; i < l; i++ {
		c = text[i]
		if -1 < expressionStart && ('\'' == c || '"' == c) && ('\\' != lastChar || (backslashCount%2) == 0) {
			endQuoteIndex := findEndQuoteChar(text, i, l)
			if 0 > endQuoteIndex {
				break
			}
			i = endQuoteIndex
			continue
		}

		switch c {
		case ':':
			if '\\' != lastChar {
				attrName = escapeBackslashText(text, expressionStart, i, backslashIndexes)
				expressionStart = i + 1
				backslashIndexes = []int{}
			}
			break
		case ',':
			if '\\' != lastChar {
				val := escapeBackslashText(text, expressionStart, i, backslashIndexes)
				expressionStart = i + 1
				backslashIndexes = []int{}
				if "" == attrName && "" == name {
					name = val
				} else {
					attributes[attrName] = val
					attrName = ""
				}
			}
			break
		case '\\':
			if '\\' == lastChar {
				lastChar = byte(0)
			} else {
				backslashIndexes = append(backslashIndexes, i)
			}
			backslashCount++
			break
		default:
			break
		}
		lastChar = c
	}

	if expressionStart < l {
		val := escapeBackslashText(text, expressionStart, l, backslashIndexes)
		if "" == attrName && "" == name {
			name = val
		} else {
			attributes[attrName] = val
		}
	}

	return name, attributes
}

func findEndQuoteChar(value string, quoteStart int, valueLen int) int {
	quoteChar := value[quoteStart]
	backslashCount := 0
	lastChar := byte(0)
	var c byte
	for i := quoteStart + 1; i < valueLen; i++ {
		c = value[i]
		if '\\' == c {
			backslashCount++
		} else if c == quoteChar && (0 == (backslashCount%2) || '\\' != lastChar) {
			return i
		}
		lastChar = c
	}
	return -1
}

func escapeBackslashText(value string, start int, end int, backslashIndexes []int) string {
	if len(backslashIndexes) > 0 {
		var texts = []string{}
		for _, i := range backslashIndexes {
			if i < start {
				continue
			} else if i >= end-1 {
				texts = append(texts, value[start:end])
				start = i + 1
				break
			}
			texts = append(texts, value[start:i])
			start = i + 1
		}
		if start < end {
			texts = append(texts, value[start:end])
		}
		return strings.Join(texts, "")
	}
	return value[start:end]
}
