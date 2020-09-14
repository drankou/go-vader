package vader

import (
	"fmt"
	"io/ioutil"
	"math"
	"strings"

	"github.com/gonum/floats"
)

//Give a sentiment intensity score to sentences.
type SentimentIntensityAnalyzer struct {
	LexiconMap        map[string]float64
	EmojiLexiconMap   map[string]string
	SpecialCaseIdioms map[string]float64
}

// Initialize sentiment analyzer with lexicons
// if no filepaths passed to init, using default lexicon files
func (sia *SentimentIntensityAnalyzer) Init(filenames ...string) error {
	var lexiconFilename string
	var emojiLexiconFilename string

	if len(filenames) == 2 {
		lexiconFilename = filenames[0]
		emojiLexiconFilename = filenames[1]
	} else {
		lexiconFilename = "../data/vader_lexicon.txt"
		emojiLexiconFilename = "../data/emoji_utf8_lexicon.txt"
	}

	// load lexicon file
	lexicon, err := ioutil.ReadFile(lexiconFilename)
	if err != nil {
		return err
	}
	sia.LexiconMap = MakeLexiconMap(string(lexicon))

	// load emoji lexicon file
	emojiLexicon, err := ioutil.ReadFile(emojiLexiconFilename)
	if err != nil {
		return err
	}
	sia.EmojiLexiconMap = MakeEmojiLexiconMap(string(emojiLexicon))

	//set special case idioms for analyzer
	sia.SpecialCaseIdioms = SpecialCaseIdioms

	return nil
}

// Return a float for sentiment strength based on the input text.
// Positive values are positive valence, negative value are negative valence.
func (sia *SentimentIntensityAnalyzer) PolarityScores(text string) map[string]float64 {
	if strings.Contains(text, "%") {
		text = ReplacePercentages(text)
	}

	// create list of tokens from text
	var textTokensList []string
	for _, token := range strings.Fields(text) {
		if EmojisRegexp.Match([]byte(token)) {
			emoticons := strings.Split(token, "")
			for _, emoticon := range emoticons {
				textTokensList = append(textTokensList, emoticon)
			}
		} else {
			textTokensList = append(textTokensList, token)
		}
	}

	// replace all emojis with its description
	textNoEmojiList := make([]string, 0, len(textTokensList))
	for _, token := range textTokensList {
		if description, ok := sia.EmojiLexiconMap[token]; ok {
			textNoEmojiList = append(textNoEmojiList, description)
		} else {
			textNoEmojiList = append(textNoEmojiList, token)
		}
	}

	// prepare sentiText for further processing
	text = strings.TrimSpace(strings.Join(textNoEmojiList, " "))
	sentiText := NewSentiText(text)

	sentiments := make([]float64, 0, len(sentiText.WordsAndEmoticonsLower))
	for wordIndex, word := range sentiText.WordsAndEmoticonsLower {
		valence := 0.0

		// check for vader_lexicon words that may be used as modifiers or negations
		if _, ok := BoosterMap[word]; ok {
			sentiments = append(sentiments, valence)
		} else if wordIndex < len(sentiText.WordsAndEmoticonsLower)-1 && word == "kind" && sentiText.WordsAndEmoticonsLower[wordIndex+1] == "of" {
			sentiments = append(sentiments, valence)
		} else {
			sentiments = sia.sentimentValence(valence, sentiText, word, wordIndex, sentiments)
		}
	}

	sentiments = butCheck(sentiText.WordsAndEmoticonsLower, sentiments)
	valenceDict := sia.scoreValence(sentiments, text)

	return valenceDict
}

func (sia *SentimentIntensityAnalyzer) sentimentValence(valence float64, sentiText *SentiText, token string, tokenIndex int, sentiments []float64) []float64 {
	//get the sentiment valence
	if value, ok := sia.LexiconMap[token]; ok {
		valence = value

		//check for "no" as negation for an adjacent lexicon item vs "no" as its own stand-alone lexicon item
		if token == "no" && tokenIndex != len(sentiText.WordsAndEmoticons)-1 {
			if _, found := sia.LexiconMap[sentiText.WordsAndEmoticonsLower[tokenIndex+1]]; found {
				// don't use valence of "no" as a lexicon item. Instead set it's valence to 0.0 and negate the next item
				valence = 0.0
			}

			if (tokenIndex > 0 && sentiText.WordsAndEmoticonsLower[tokenIndex-1] == "no") ||
				(tokenIndex > 1 && sentiText.WordsAndEmoticonsLower[tokenIndex-2] == "no") ||
				(tokenIndex > 2 && sentiText.WordsAndEmoticonsLower[tokenIndex-3] == "no" &&
					(sentiText.WordsAndEmoticonsLower[tokenIndex-1] == "or" || sentiText.WordsAndEmoticonsLower[tokenIndex-1] == "nor")) {
				valence = value * N_SCALAR
			}
		}

		//check if sentiment laden word is in ALL CAPS (while others aren't)
		if token == strings.ToUpper(token) && sentiText.IsCapDiff {
			if valence > 0 {
				valence += C_INCR
			} else {
				valence -= C_INCR
			}
		}

		//check preceding words modifiers
		for startIndex := 0; startIndex < 3; startIndex++ {
			if tokenIndex > startIndex {
				if _, ok := sia.LexiconMap[sentiText.WordsAndEmoticonsLower[tokenIndex-(startIndex+1)]]; !ok {
					// add boost value to actual valence
					valence += getBoostValue(sentiText.WordsAndEmoticonsLower[tokenIndex-(startIndex+1)], startIndex, valence, sentiText.IsCapDiff)

					// check negation
					valence = sia.negationCheck(valence, sentiText.WordsAndEmoticonsLower, tokenIndex)
				}
			}
		}
	}

	//check special case idioms
	valence = sia.specialIdiomsCheck(valence, sentiText.WordsAndEmoticonsLower, tokenIndex)

	sentiments = append(sentiments, valence)
	return sentiments
}

// check boost of previous words
func getBoostValue(token string, startIndex int, valence float64, isCapDiff bool) float64 {
	boost := scalarIncDec(token, valence, isCapDiff)
	if boost != 0 {
		switch startIndex {
		case 0:
			boost *= 1
		case 1:
			boost *= 0.95
		case 2:
			boost *= 0.9
		}
	}

	return boost
}

func butCheck(wordsAndEmoticons []string, sentiments []float64) []float64 {
	// check for modification in sentiment due to contrastive conjunction 'but'
	for wi, word := range wordsAndEmoticons {
		if word == "but" {
			for si, sentiment := range sentiments {
				if si < wi {
					sentiments[si] = sentiment * 0.5
				} else if si > wi {
					sentiments[si] = sentiment * 1.5
				}
			}
		}
	}

	return sentiments
}

func (sia *SentimentIntensityAnalyzer) specialIdiomsCheck(valence float64, wordsAndEmoticons []string, tokenIndex int) float64 {
	if len(wordsAndEmoticons) == 0 {
		return valence
	}

	ngrams := make(map[string]string)
	//construct possible ngrams
	switch v := tokenIndex; {
	case v > 2:
		ngrams["threeTwoOne"] = fmt.Sprintf("%s %s %s", wordsAndEmoticons[tokenIndex-3], wordsAndEmoticons[tokenIndex-2], wordsAndEmoticons[tokenIndex-1])
		ngrams["threeTwo"] = fmt.Sprintf("%s %s", wordsAndEmoticons[tokenIndex-3], wordsAndEmoticons[tokenIndex-2])
		fallthrough
	case v > 1:
		ngrams["twoOneZero"] = fmt.Sprintf("%s %s %s", wordsAndEmoticons[tokenIndex-2], wordsAndEmoticons[tokenIndex-1], wordsAndEmoticons[tokenIndex])
		ngrams["twoOne"] = fmt.Sprintf("%s %s", wordsAndEmoticons[tokenIndex-2], wordsAndEmoticons[tokenIndex-1])
		fallthrough
	case v > 0:
		ngrams["oneZero"] = fmt.Sprintf("%s %s", wordsAndEmoticons[tokenIndex-1], wordsAndEmoticons[tokenIndex])
	}

	if len(wordsAndEmoticons)-1 > tokenIndex+1 {
		ngrams["zeroOneTwo"] = fmt.Sprintf("%s %s %s", wordsAndEmoticons[tokenIndex], wordsAndEmoticons[tokenIndex+1], wordsAndEmoticons[tokenIndex+2])
	}

	if len(wordsAndEmoticons)-1 > tokenIndex {
		ngrams["zeroOne"] = fmt.Sprintf("%s %s", wordsAndEmoticons[tokenIndex], wordsAndEmoticons[tokenIndex+1])
	}

	for _, ngram := range ngrams {
		if value, ok := sia.SpecialCaseIdioms[ngram]; ok {
			valence = value
			break
		}
	}

	possibleBoosters := []string{ngrams["threeTwoOne"], ngrams["threeTwo"], ngrams["twoOne"]}
	// check for booster/dampener bi-grams such as 'sort of' or 'kind of'
	for _, ngram := range possibleBoosters {
		if value, ok := BoosterMap[ngram]; ok {
			valence = valence + value
		}
	}

	if valence != 0 {
		valence = sia.negationCheck(valence, wordsAndEmoticons, specialCaseIdiomStartIndex)
	}

	return valence
}

// Future Work
// check for sentiment laden idioms that don't contain a lexicon word
func (sia *SentimentIntensityAnalyzer) sentimentLadenIdiomsCheck(valence float64, text string) float64 {
	// TODO in future
	return 0.0
}

//check for negations
func (sia *SentimentIntensityAnalyzer) negationCheck(valence float64, wordsAndEmoticons []string, tokenIndex int) float64 {
	if len(wordsAndEmoticons) == 0 {
		return valence
	}

	//check previous words for negations
	switch i := tokenIndex; {
	case i > 2:
		if wordsAndEmoticons[tokenIndex-3] == "never" &&
			((wordsAndEmoticons[tokenIndex-2] == "so" || wordsAndEmoticons[tokenIndex-2] == "this") ||
				(wordsAndEmoticons[tokenIndex-1] == "so" || wordsAndEmoticons[tokenIndex-1] == "this")) {
			return valence * 1.25
		} else if wordsAndEmoticons[tokenIndex-3] == "without" && (wordsAndEmoticons[tokenIndex-2] == "doubt" || wordsAndEmoticons[tokenIndex-1] == "doubt") {
			return valence
		} else if ContainsNegation(wordsAndEmoticons[tokenIndex-3 : tokenIndex]) { //3 words preceding the lexicon word position
			return valence * N_SCALAR
		}
	case i > 1:
		if wordsAndEmoticons[tokenIndex-2] == "never" && (wordsAndEmoticons[tokenIndex-1] == "so" || wordsAndEmoticons[tokenIndex-1] == "this") {
			return valence * 1.25
		} else if wordsAndEmoticons[tokenIndex-2] == "without" && wordsAndEmoticons[tokenIndex-1] == "doubt" {
			return valence
		} else if ContainsNegation(wordsAndEmoticons[tokenIndex-2 : tokenIndex]) { // 2 words preceding the lexicon word position
			return valence * N_SCALAR
		}
	case i > 0:
		if ContainsNegation(wordsAndEmoticons[tokenIndex-1 : tokenIndex]) { // 1 word preceding lexicon word (w/o stopwords)
			return valence * N_SCALAR
		}
	}

	return valence
}

// add emphasis from exclamation points and question marks
func (sia *SentimentIntensityAnalyzer) punctuationEmphasis(text string) float64 {
	epAmplifier := sia.amplifyEP(text)
	qmAmplifier := sia.amplifyQM(text)

	return epAmplifier + qmAmplifier
}

// check for added emphasis resulting from exclamation points (up to 4 of them)
func (sia *SentimentIntensityAnalyzer) amplifyEP(text string) float64 {
	epCount := strings.Count(text, "!")
	if epCount > MaxEM {
		epCount = MaxEM
	}

	// (empirically derived mean sentiment intensity rating increase for exclamation points)
	return float64(epCount) * 0.292
}

// check for added emphasis resulting from question marks (2 or 3+)
func (sia *SentimentIntensityAnalyzer) amplifyQM(text string) float64 {
	qmCount := strings.Count(text, "?")
	if qmCount > 1 {
		if qmCount <= MaxQM {
			return float64(qmCount) * 0.18
		} else {
			return 0.96
		}
	}

	return 0.0
}

// want separate positive versus negative sentiment scores
func (sia *SentimentIntensityAnalyzer) siftSentimentScores(sentiments []float64) (float64, float64, float64) {
	posSum := 0.0
	negSum := 0.0
	neuCount := 0.0

	for _, sentiment := range sentiments {
		if sentiment > 0 {
			posSum += sentiment + 1 //compensates for neutral words that are counted as 1
		} else if sentiment < 0 {
			negSum += sentiment - 1 //when used with math.fabs(), compensates for neutrals
		} else {
			neuCount++
		}
	}

	return posSum, negSum, neuCount
}

func (sia *SentimentIntensityAnalyzer) scoreValence(sentiments []float64, text string) map[string]float64 {
	var compound float64
	var pos float64
	var neg float64
	var neu float64

	if len(sentiments) > 0 {
		sumS := floats.Sum(sentiments)

		// compute and add emphasis from punctuation in text
		punctEmphAmplifier := sia.punctuationEmphasis(text)
		if sumS > 0 {
			sumS += punctEmphAmplifier
		} else if sumS < 0 {
			sumS -= punctEmphAmplifier
		}
		compound = Normalize(sumS)

		// discriminate between positive, negative and neutral sentiment scores
		posSum, negSum, neuCount := sia.siftSentimentScores(sentiments)
		if posSum > math.Abs(negSum) {
			posSum += punctEmphAmplifier
		} else if posSum < math.Abs(negSum) {
			negSum -= punctEmphAmplifier
		}

		total := posSum + math.Abs(negSum) + neuCount

		pos = math.Abs(posSum / total)
		neg = math.Abs(negSum / total)
		neu = math.Abs(neuCount / total)
	}

	sentimentMap := map[string]float64{
		"pos":      floats.Round(pos, 3),
		"neg":      floats.Round(neg, 3),
		"neu":      floats.Round(neu, 3),
		"compound": floats.Round(compound, 4),
	}

	return sentimentMap
}

// Check if the preceding words increase, decrease, or negate/nullify the
// valence
func scalarIncDec(word string, valence float64, isCapDiff bool) float64 {
	var scalar float64

	if value, ok := BoosterMap[word]; ok {
		scalar = value
		if valence < 0 {
			scalar *= -1
		}
		//check if booster/dampener word is in ALLCAPS (while others aren't)
		if word == strings.ToUpper(word) && isCapDiff {
			if valence > 0 {
				scalar += C_INCR
			} else {
				scalar -= C_INCR
			}
		}
	}

	return scalar
}
