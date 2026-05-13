package db

import (
	"regexp"
	"strings"
	"unicode"
)

var punctuationReg = regexp.MustCompile(`[^\p{L}\p{N}]+`)

// BigramTokenize 将文本进行二元分词，用于全文检索影子列
// 规则：
// 1. 移除非字母数字的标点符号，按空格/标点初步切分块。
// 2. 对每个块内的 CJK（中日韩）字符，使用滑动窗口进行 2-gram 切分。
// 3. 对于块内的非 CJK（英文、数字等）字符，按单词整体保留。
func BigramTokenize(text string) string {
	if text == "" {
		return ""
	}

	// 1. 初步切分，按非字母数字字符分割
	chunks := punctuationReg.Split(text, -1)
	var allTokens []string

	for _, chunk := range chunks {
		if chunk == "" {
			continue
		}
		runes := []rune(chunk)
		length := len(runes)

		var currentWord []rune
		for i := 0; i < length; i++ {
			r := runes[i]
			if isCJK(r) {
				// 遇到中文字符，先冲刷掉之前的英文单词
				if len(currentWord) > 0 {
					allTokens = append(allTokens, string(currentWord))
					currentWord = nil
				}
				// 1-gram
				allTokens = append(allTokens, string(r))
				// 2-gram
				if i < length-1 && isCJK(runes[i+1]) {
					allTokens = append(allTokens, string(runes[i:i+2]))
				}
			} else {
				// 累积英文/数字
				currentWord = append(currentWord, r)
			}
		}
		// 循环结束，冲刷最后一个单词
		if len(currentWord) > 0 {
			allTokens = append(allTokens, string(currentWord))
		}
	}

	// 4. 去重，减小索引体积
	tokenMap := make(map[string]bool)
	var uniqueTokens []string
	for _, t := range allTokens {
		if !tokenMap[t] {
			tokenMap[t] = true
			uniqueTokens = append(uniqueTokens, t)
		}
	}

	return strings.Join(uniqueTokens, " ")
}

func isCJK(r rune) bool {
	return unicode.Is(unicode.Han, r) ||
		unicode.In(r, unicode.Hiragana, unicode.Katakana, unicode.Hangul)
}
