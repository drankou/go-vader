package vader

import (
	"strings"
)

type SentiText struct {
	WordsAndEmoticons      []string
	WordsAndEmoticonsLower []string
	IsCapDiff              bool
}

func NewSentiText(text string) *SentiText {
	wordsAndEmoticons := CleanWordsAndEmoticons(text)
	isCapDiff := IsAllCapDiff(wordsAndEmoticons)

	wordsAndEmoticonsLower := make([]string, 0, len(wordsAndEmoticons))
	for _, w := range wordsAndEmoticons {
		wordsAndEmoticonsLower = append(wordsAndEmoticonsLower, strings.ToLower(w))
	}

	return &SentiText{
		WordsAndEmoticons:      wordsAndEmoticons,
		WordsAndEmoticonsLower: wordsAndEmoticonsLower,
		IsCapDiff:              isCapDiff,
	}
}
