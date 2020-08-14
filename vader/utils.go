package vader

import (
	"log"
	"math"
	"strconv"
	"strings"
)

// Normalize the score to be between -1 and 1 using an alpha that
// approximates the max expected value
func Normalize(score float64) float64 {
	normalizedScore := score / math.Sqrt((score*score)+float64(Alpha))

	if normalizedScore < -1.0 {
		return -1.0
	} else if normalizedScore > 1.0 {
		return 1.0
	} else {
		return normalizedScore
	}
}

//Removes leading and trailing punctuation
//Leaves contractions and most emoticons
//Does not preserve punc-plus-letter emoticons (e.g. :D)
//Returns list of clean words from text
func CleanWordsAndEmoticons(text string) []string {
	words := strings.Fields(text)

	cleanWords := make([]string, 0, len(words))
	for _, word := range words {
		cleanWord := strings.TrimFunc(word, func(r rune) bool {
			return PunctuationRegexp.Match([]byte{byte(r)})
		})

		if len(cleanWord) <= 2 {
			cleanWords = append(cleanWords, word)
		} else {
			cleanWords = append(cleanWords, cleanWord)
		}
	}

	return cleanWords
}

//Check whether just some words in the input are ALL CAPS
func IsAllCapDiff(words []string) bool {
	for _, word := range words {
		if word != strings.ToUpper(word) {
			return true
		}
	}

	return false
}

//additional function for emoji check
//for case if they are not separated by whitespace
func ReplaceEmojis(text string) string {
	// find all emojis in text
	emojis := EmojisRegexp.FindAllString(text, -1)
	emojisText := strings.Join(emojis, " ")

	//concatenate emojis separated by whitespace with cleaned text
	cleanText := EmojisRegexp.ReplaceAllString(text, "")
	text = cleanText + " " + emojisText

	return text
}

// find percent difference occurences (+2%,-2% etc.)
// and replace it with placeholder from lexicon
func ReplacePercentages(text string) string {
	text = PositivePercentageRegexp.ReplaceAllString(text, " xpositivepercentx ")
	text = NegativePercentageRegexp.ReplaceAllString(text, " xnegativepercentx ")

	return text
}

//Convert lexicon file data to map
func MakeLexiconMap(lexicon string) map[string]float64 {
	lexiconDict := make(map[string]float64)

	for _, line := range strings.Split(strings.TrimSuffix(lexicon, "\n"), "\n") {
		line = strings.TrimSpace(line)
		values := strings.Split(line, "\t")

		word := values[0]
		measure, err := strconv.ParseFloat(values[1], 64)
		if err != nil {
			log.Fatal(err)
		}

		lexiconDict[word] = measure
	}

	return lexiconDict
}

// Convert emoji lexicon file data to map
func MakeEmojiLexiconMap(emojiLexicon string) map[string]string {
	emojiLexiconDict := make(map[string]string)

	for _, line := range strings.Split(emojiLexicon, "\n") {
		line = strings.TrimSpace(line)
		values := strings.Split(line, "\t")

		word := values[0]
		description := values[1]

		emojiLexiconDict[word] = description
	}

	return emojiLexiconDict
}

// Determine if input contains negation words
func ContainsNegation(inputWords []string) bool {
	var inputWordsLowercased []string

	for _, inputWord := range inputWords {
		inputWordsLowercased = append(inputWordsLowercased, strings.ToLower(inputWord))
	}

	for i, word := range inputWordsLowercased {
		for _, negWord := range Negations {
			if negWord == word {
				return true
			}
		}

		if word == "least" {
			if i > 0 && inputWordsLowercased[i-1] != "at" {
				return true
			}
		}

		if IncludeNt {
			if strings.Contains(word, "n't") {
				return true
			}
		}
	}

	return false
}