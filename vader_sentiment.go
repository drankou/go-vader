package main

import (
	"fmt"
	"github.com/gonum/floats"
	"io/ioutil"
	"log"
	"math"
	"regexp"
	"strconv"
	"strings"
)

const (
	//(empirically derived mean sentiment intensity rating increase for booster words)
	B_INCR = 0.293
	B_DECR = -0.293

	//(empirically derived mean sentiment intensity rating increase for using ALLCAPs to emphasize a word)
	C_INCR   = 0.733
	N_SCALAR = -0.74

	lexicon_file       = "vader_lexicon.txt"
	emoji_lexicon_file = "emoji_utf8_lexicon.txt"

	alpha = 15			//constant for normalize
	include_nt = true 	//flag to check "n't" in negated
)

//for removing punctuation
var REGEX_REMOVE_PUNCTUATION = regexp.MustCompile(fmt.Sprintf("[%s]", regexp.QuoteMeta(`!"//$%&'()*+,-./:;<=>?@[\]^_{|}~`+"`")))

var PUNC_LIST = []string{".", "!", "?", ",", ";", ":", "-", "'", "\"", "!!", "!!!", "??", "???", "?!?", "!?!", "?!?!", "!?!?"}

var NEGATE = []string{"aint", "arent", "cannot", "cant", "couldnt", "darent", "didnt", "doesnt",
	"ain't", "aren't", "can't", "couldn't", "daren't", "didn't", "doesn't",
	"dont", "hadnt", "hasnt", "havent", "isnt", "mightnt", "mustnt", "neither",
	"don't", "hadn't", "hasn't", "haven't", "isn't", "mightn't", "mustn't",
	"neednt", "needn't", "never", "none", "nope", "nor", "not", "nothing", "nowhere",
	"oughtnt", "shant", "shouldnt", "uhuh", "wasnt", "werent",
	"oughtn't", "shan't", "shouldn't", "uh-uh", "wasn't", "weren't",
	"without", "wont", "wouldnt", "won't", "wouldn't", "rarely", "seldom", "despite"}

// booster/dampener 'intensifiers' or 'degree adverbs'
// http://en.wiktionary.org/wiki/Category:English_degree_adverbs
var BOOSTER_DICT = map[string]float64{"absolutely": B_INCR, "amazingly": B_INCR, "awfully": B_INCR, "completely": B_INCR, "considerably": B_INCR,
	"decidedly": B_INCR, "deeply": B_INCR, "effing": B_INCR, "enormously": B_INCR,
	"entirely": B_INCR, "especially": B_INCR, "exceptionally": B_INCR, "extremely": B_INCR,
	"fabulously": B_INCR, "flipping": B_INCR, "flippin": B_INCR,
	"fricking": B_INCR, "frickin": B_INCR, "frigging": B_INCR, "friggin": B_INCR, "fully": B_INCR, "fucking": B_INCR,
	"greatly": B_INCR, "hella": B_INCR, "highly": B_INCR, "hugely": B_INCR, "incredibly": B_INCR,
	"intensely": B_INCR, "majorly": B_INCR, "more": B_INCR, "most": B_INCR, "particularly": B_INCR,
	"purely": B_INCR, "quite": B_INCR, "really": B_INCR, "remarkably": B_INCR,
	"so": B_INCR, "substantially": B_INCR,
	"thoroughly": B_INCR, "totally": B_INCR, "tremendously": B_INCR,
	"uber": B_INCR, "unbelievably": B_INCR, "unusually": B_INCR, "utterly": B_INCR,
	"very":   B_INCR,
	"almost": B_DECR, "barely": B_DECR, "hardly": B_DECR, "just enough": B_DECR,
	"kind of": B_DECR, "kinda": B_DECR, "kindof": B_DECR, "kind-of": B_DECR,
	"less": B_DECR, "little": B_DECR, "marginally": B_DECR, "occasionally": B_DECR, "partly": B_DECR,
	"scarcely": B_DECR, "slightly": B_DECR, "somewhat": B_DECR,
	"sort of": B_DECR, "sorta": B_DECR, "sortof": B_DECR, "sort-of": B_DECR}

// check for sentiment laden idioms that do not contain lexicon words (future work, not yet implemented)
var SENTIMENT_LADEN_IDIOMS = map[string]int{"cut the mustard": 2, "hand to mouth": -2,
	"back handed": -2, "blow smoke": -2, "blowing smoke": -2,
	"upper hand": 1, "break a leg": 2,
	"cooking with gas": 2, "in the black": 2, "in the red": -2,
	"on the ball": 2, "under the weather": -2}

// check for special case idioms containing lexicon words
var SPECIAL_CASE_IDIOMS = map[string]float64{"the shit": 3, "the bomb": 3, "bad ass": 1.5, "yeah right": -2,
	"kiss of death": -1.5}


// Determine if input contains negation words
func negated(inputWords []string) bool {
	var inputWordsLowercased []string

	for _, inputWord := range inputWords {
		inputWordsLowercased = append(inputWordsLowercased, strings.ToLower(inputWord))
	}

	for i, word := range inputWordsLowercased {
		for _, negWord := range NEGATE {
			if negWord == word {
				return true
			}
		}

		if word == "least" {
			if i > 0 && inputWordsLowercased[i-1] != "at" {
				return true
			}
		}

		if include_nt {
			if strings.Contains(word, "n't") {
				return true
			}
		}
	}

	return false
}

// Normalize the score to be between -1 and 1 using an alpha that
// approximates the max expected value
func normalize(score float64) float64 {
	normalizedScore := score / math.Sqrt((score*score) + float64(alpha))

	if normalizedScore < -1.0 {
		return -1.0
	} else if normalizedScore > 1.0 {
		return 1.0
	} else {
		return normalizedScore
	}
}

//Check whether just some words in the input are ALL CAPS
//:param list words: The words to inspect
//:returns: `True` if some but not all items in `words` are ALL CAPS
func allcapDifferential(words []string) bool {
	var isDifferent bool
	var allcapWords int

	for _, word := range words {
		if word == strings.ToUpper(word) {
			allcapWords++
		}
	}

	capDifferential := len(words) - allcapWords
	if capDifferential > 0 && capDifferential < len(words) {
		isDifferent = true
	}

	return isDifferent
}

// Check if the preceding words increase, decrease, or negate/nullify the
// valence
func scalarIncDec(word string, valence float64, isCapDiff bool) float64 {
	var scalar float64

	wordLower := strings.ToLower(word)

	if value, ok := BOOSTER_DICT[wordLower]; ok {
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

type SentiText struct {
	Text              string
	WordsAndEmoticons []string
	IsCapDiff         bool
}

func (s *SentiText) _init(text string) {
	//TODO check if text is utf-8 encoded, encode if needed (implicit in Golang ?)
	s.Text = text
	s.WordsAndEmoticons = s._wordsAndEmoticons()
	// doesn't separate words from
	// adjacent punctuation (keeps emoticons & contractions)
	s.IsCapDiff = allcapDifferential(s.WordsAndEmoticons)
}

//Returns mapping of form:
//{
//'cat,': 'cat',
//',cat': 'cat',
//}
func (s *SentiText) _wordsPlusPunc() map[string]string {
	noPuncText := REGEX_REMOVE_PUNCTUATION.ReplaceAllString(string(s.Text), "")

	//removes punctuation (but loses emoticons & contractions)
	wordsOnly := strings.Fields(noPuncText)

	// remove singletons
	var wordsOnlyWithoutSingletons []string
	for _, word := range wordsOnly {
		if len(word) > 1 {
			wordsOnlyWithoutSingletons = append(wordsOnlyWithoutSingletons, word)
		}
	}

	// the product gives ('cat', ',') and (',', 'cat')

	puncBefore := make(map[string]string)
	puncAfter := make(map[string]string)

	for _, p := range cartesianProduct(PUNC_LIST, wordsOnly) {
		puncBefore[strings.Join(p, "")] = p[1]
	}

	for _, p := range cartesianProduct(wordsOnly, PUNC_LIST) {
		puncAfter[strings.Join(p, "")] = p[0]
	}

	wordsPuncDict := puncBefore

	//TODO check if it correct
	for key, value := range puncAfter {
		wordsPuncDict[key] = value
	}

	return wordsPuncDict
}

// Cartesian product of input iterables.
func cartesianProduct(arr1 []string, arr2 []string) [][]string {
	var result [][]string

	for _, item1 := range arr1 {
		for _, item2 := range arr2 {
			result = append(result, []string{item1, item2})
		}
	}

	return result
}

//Removes leading and trailing puncutation
//Leaves contractions and most emoticons
//Does not preserve punc-plus-letter emoticons (e.g. :D)
func (s *SentiText) _wordsAndEmoticons() []string {
	wes := strings.Fields(s.Text)
	wordsPuncDict := s._wordsPlusPunc()

	var wesCleaned []string
	for _, we := range wes {
		if len(we) > 1 {
			wesCleaned = append(wesCleaned, we)
		}
	}

	for i, we := range wesCleaned {
		if value, ok := wordsPuncDict[we]; ok {
			wesCleaned[i] = value
		}
	}

	return wesCleaned
}

//Give a sentiment intensity score to sentences.
type SentimentIntensityAnalyzer struct {
	LexiconMap      map[string]float64
	EmojiLexiconMap map[string]string
}

func (sia *SentimentIntensityAnalyzer) Init() error {
	//TODO path abs paths

	// load lexicon file
	lexicon, err := ioutil.ReadFile(lexicon_file)
	if err != nil {
		return err
	}

	sia.LexiconMap = sia.makeLexiconMap(string(lexicon))

	// load emoji lexicon file
	emojiLexicon, err := ioutil.ReadFile(emoji_lexicon_file)
	if err != nil {
		return err
	}

	sia.EmojiLexiconMap = sia.makeEmojiLexiconMap(string(emojiLexicon))

	return nil
}

//Convert lexicon file to a dictionary
func (sia *SentimentIntensityAnalyzer) makeLexiconMap(lexicon string) map[string]float64 {
	lexiconDict := make(map[string]float64)

	for _, line := range strings.Split(lexicon, "\n") {
		line = strings.Replace(line, " ", "", -1)
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

// Convert emoji lexicon file to a dictionary
func (sia *SentimentIntensityAnalyzer) makeEmojiLexiconMap(emojiLexicon string) map[string]string {
	emojiLexiconDict := make(map[string]string)

	for _, line := range strings.Split(emojiLexicon, "\n") {
		line = strings.Replace(line, " ", "", -1)
		values := strings.Split(line, "\t")

		word := values[0]
		description := values[1]

		emojiLexiconDict[word] = description
	}

	return emojiLexiconDict
}

// Return a float for sentiment strength based on the input text.
// Positive values are positive valence, negative value are negative valence.
func (sia *SentimentIntensityAnalyzer) polarityScores(text string) map[string]float64 {
	var textNoEmojiList []string

	textTokensList := strings.Fields(text)
	for _, token := range textTokensList {
		if description, ok := sia.EmojiLexiconMap[token]; ok {
			textNoEmojiList = append(textNoEmojiList, description)
		} else {
			textNoEmojiList = append(textNoEmojiList, token)
		}
	}
	text = strings.Join(textNoEmojiList, " ")

	sentiText := SentiText{}
	sentiText._init(text)

	var sentiments []float64

	wordsAndEmoticons := sentiText.WordsAndEmoticons
	for i, item := range sentiText.WordsAndEmoticons {
		valence := 0.0

		// check for vader_lexicon words that may be used as modifiers or negations
		if _, ok := BOOSTER_DICT[strings.ToLower(item)]; ok {
			sentiments = append(sentiments, valence)
			continue
		}

		if i < len(wordsAndEmoticons)-1 && strings.ToLower(item) == "kind" && strings.ToLower(wordsAndEmoticons[i+1]) == "of" {
			sentiments = append(sentiments, valence)
			continue
		}

		sentiments = sia.sentimentValence(valence, &sentiText, item, i, sentiments)
	}

	sentiments = sia.butCheck(wordsAndEmoticons, sentiments)
	valenceDict := sia.scoreValence(sentiments, text)

	return valenceDict
}

func (sia *SentimentIntensityAnalyzer) sentimentValence(valence float64, sentiText *SentiText, item string, i int, sentiments []float64) []float64 {
	isCapDiff := sentiText.IsCapDiff
	wordsAndEmoticons := sentiText.WordsAndEmoticons
	itemLowercase := strings.ToLower(item)

	//get the sentiment valence
	if value, ok := sia.LexiconMap[itemLowercase]; ok {
		valence = value

		//check if sentiment laden word is in ALL CAPS (while others aren't)
		if item == strings.ToUpper(item) && isCapDiff {
			if valence > 0 {
				valence += C_INCR
			} else {
				valence -= C_INCR
			}
		}

		for start_i := 0; start_i <= 2; start_i++ {
			//// dampen the scalar modifier of preceding words and emoticons
			//// (excluding the ones that immediately preceed the item) based
			//// on their distance from the current item.

			//TODO check if condition is written properly
			//TODO ERROR:out of range
			if i <= start_i {
				continue
			}

			if _, ok := sia.LexiconMap[strings.ToLower(wordsAndEmoticons[i-(start_i+1)])]; !ok {
				s := scalarIncDec(wordsAndEmoticons[i-(start_i+1)], valence, isCapDiff)

				if start_i == 1 && s != 0 {
					s *= 0.95
				}

				if start_i == 2 && s != 0 {
					s *= 0.9
				}

				valence += s
				valence = sia.negationCheck(valence, wordsAndEmoticons, start_i, i)

				if start_i == 2 {
					valence = sia.specialIdiomsCheck(valence, wordsAndEmoticons, i)
				}
			}
		}
		valence = sia.leastCheck(valence, wordsAndEmoticons, i)
	}

	sentiments = append(sentiments, valence)
	return sentiments
}

func (sia *SentimentIntensityAnalyzer) leastCheck(valence float64, wordsAndEmoticons []string, i int) float64 {
	// check for negation case using "least"

	if _, ok := sia.LexiconMap[strings.ToLower(wordsAndEmoticons[i-1])]; !ok && i > 1 && strings.ToLower(wordsAndEmoticons[i-1]) == "least" {
		if strings.ToLower(wordsAndEmoticons[i-2]) != "at" && strings.ToLower(wordsAndEmoticons[i-2]) != "very" {
			valence = valence * N_SCALAR
		}
	} else if _, ok := sia.LexiconMap[strings.ToLower(wordsAndEmoticons[i-1])]; !ok && i > 0 && strings.ToLower(wordsAndEmoticons[i-1]) == "least" {
		valence = valence * N_SCALAR
	}

	return valence
}

func (sia *SentimentIntensityAnalyzer) butCheck(wordsAndEmoticons []string, sentiments []float64) []float64 {
	// check for modification in sentiment due to contrastive conjunction 'but'
	var wordsAndEmoticonsLower []string

	for _, w := range wordsAndEmoticons {
		wordsAndEmoticonsLower = append(wordsAndEmoticonsLower, strings.ToLower(w))
	}

	for bi, wl := range wordsAndEmoticonsLower {
		if wl == "but" {
			for si, sentiment := range sentiments {
				if si < bi {
					//TODO check pop method on python list
					sentiments[si] = sentiment * 0.5
				} else if si > bi {
					sentiments[si] = sentiment * 1.5
				}
			}
		}
	}

	return sentiments
}

func (sia *SentimentIntensityAnalyzer) specialIdiomsCheck(valence float64, wordsAndEmoticons []string, i int) float64 {
	var wordsAndEmoticonsLower []string

	for _, w := range wordsAndEmoticons {
		wordsAndEmoticonsLower = append(wordsAndEmoticonsLower, strings.ToLower(w))
	}

	onezero := fmt.Sprintf("%s %s", wordsAndEmoticonsLower[i-1], wordsAndEmoticonsLower[i])

	twoonezero := fmt.Sprintf("%s %s %s", wordsAndEmoticonsLower[i-2], wordsAndEmoticonsLower[i-1], wordsAndEmoticonsLower[i])

	twoone := fmt.Sprintf("%s %s", wordsAndEmoticonsLower[i-2], wordsAndEmoticonsLower[i-1])

	threetwoone := fmt.Sprintf("%s %s %s", wordsAndEmoticonsLower[i-3], wordsAndEmoticonsLower[i-2], wordsAndEmoticonsLower[i-1])

	threetwo := fmt.Sprintf("%s %s", wordsAndEmoticonsLower[i-3], wordsAndEmoticonsLower[i-2])

	sequences := []string{onezero, twoonezero, twoone, threetwoone, threetwo}

	for _, seq := range sequences {
		if value, ok := SPECIAL_CASE_IDIOMS[seq]; ok {
			valence = value
			break
		}
	}

	if len(wordsAndEmoticonsLower)-1 > i {
		zeroone := fmt.Sprintf("%s %s", wordsAndEmoticonsLower[i], wordsAndEmoticonsLower[i+1])
		if value, ok := SPECIAL_CASE_IDIOMS[zeroone]; ok {
			valence = value
		}
	}

	if len(wordsAndEmoticonsLower)-1 > i+1 {
		zeroonetwo := fmt.Sprintf("%s %s %s", wordsAndEmoticonsLower[i], wordsAndEmoticonsLower[i+1], wordsAndEmoticonsLower[i+2])
		if value, ok := SPECIAL_CASE_IDIOMS[zeroonetwo]; ok {
			valence = value
		}
	}

	// check for booster/dampener bi-grams such as 'sort of' or 'kind of'

	ngrams := []string{threetwoone, threetwo, twoone}
	for _, ngram := range ngrams {
		if value, ok := BOOSTER_DICT[ngram]; ok {
			valence = valence + value
		}
	}

	return valence
}

// Future Work
// check for sentiment laden idioms that don't contain a lexicon word
func (sia *SentimentIntensityAnalyzer) sentimentLadenIdiomsCheck(valence float64, text string) float64 {
	// TODO in future

	return 0.0
}

func (sia *SentimentIntensityAnalyzer) negationCheck(valence float64, wordsAndEmoticons []string, start_i int, i int) float64 {
	var wordsAndEmoticonsLower []string

	for _, w := range wordsAndEmoticons {
		wordsAndEmoticonsLower = append(wordsAndEmoticonsLower, strings.ToLower(w))
	}

	if start_i == 0 {
		if negated([]string{wordsAndEmoticonsLower[i-(start_i+1)]}) { // 1 word preceding lexicon word (w/o stopwords)
			valence = valence * N_SCALAR
		}
	}

	if start_i == 1 {
		if wordsAndEmoticonsLower[i-2] == "never" && (wordsAndEmoticonsLower[i-1] == "so" || wordsAndEmoticonsLower[i-1] == "this") {
			valence = valence * 1.25
		} else if wordsAndEmoticonsLower[i-2] == "without" && wordsAndEmoticonsLower[i-1] == "doubt" {
			valence = valence
		} else if negated([]string{wordsAndEmoticonsLower[i-(start_i+1)]}) { // 2 words preceding the lexicon word position
			valence = valence * N_SCALAR
		}
	}

	if start_i == 2 {
		if wordsAndEmoticonsLower[i-3] == "never" && (wordsAndEmoticonsLower[i-2] == "so" || wordsAndEmoticonsLower[i-2] == "this") || (wordsAndEmoticonsLower[i-1] == "so" || wordsAndEmoticonsLower[i-1] == "this") {
			valence = valence * 1.25
		} else if wordsAndEmoticonsLower[i-3] == "without" && (wordsAndEmoticonsLower[i-2] == "doubt" || wordsAndEmoticonsLower[i-1] == "doubt") {
			valence = valence
		} else if negated([]string{wordsAndEmoticonsLower[i-(start_i+1)]}) { //3 words preceding the lexicon word position
			valence = valence * N_SCALAR
		}
	}

	return valence
}

// add emphasis from exclamation points and question marks
func (sia *SentimentIntensityAnalyzer) punctuationEmphasis(text string) float64 {
	epAmplifier := sia.amplifyEP(text)
	qmAmplifier := sia.amplifyQM(text)

	punctEmphAmplifier := epAmplifier + qmAmplifier
	return punctEmphAmplifier
}

// check for added emphasis resulting from exclamation points (up to 4 of them)
func (sia *SentimentIntensityAnalyzer) amplifyEP(text string) float64 {
	ep := regexp.MustCompile(`!`)
	matches := ep.FindAllStringIndex(text, -1)

	epCount := len(matches)
	if epCount > 4 {
		epCount = 4
	}

	// (empirically derived mean sentiment intensity rating increase for exclamation points)
	epAmplifier := float64(epCount) * 0.292

	return epAmplifier
}

// check for added emphasis resulting from question marks (2 or 3+)
func (sia *SentimentIntensityAnalyzer) amplifyQM(text string) float64 {
	qm := regexp.MustCompile(`\?`)
	matches := qm.FindAllStringIndex(text, -1)

	qmCount := len(matches)
	qmAmplifier := 0.0
	if qmCount > 1 {
		if qmCount <= 3 {
			// (empirically derived mean sentiment intensity rating increase for question marks)
			qmAmplifier = float64(qmCount) * 0.18
		} else {
			qmAmplifier = 0.96
		}
	}

	return qmAmplifier
}

// want separate positive versus negative sentiment scores
func (sia *SentimentIntensityAnalyzer) siftSentimentScores(sentiments []float64) (float64, float64, int) {
	posSum := 0.0
	negSum := 0.0
	neuCount := 0

	for _, sentiment := range sentiments {
		if sentiment > 0 {
			posSum += (sentiment + 1) //compensates for neutral words that are counted as 1
		} else if sentiment < 0 {
			negSum += (sentiment - 1) //when used with math.fabs(), compensates for neutrals
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

		compound = normalize(sumS)
		// discriminate between positive, negative and neutral sentiment scores
		posSum, negSum, neuCount := sia.siftSentimentScores(sentiments)

		if posSum > math.Abs(negSum) {
			posSum += punctEmphAmplifier
		} else if posSum < math.Abs(negSum) {
			negSum -= punctEmphAmplifier
		}

		total := posSum + math.Abs(negSum) + float64(neuCount)

		pos = math.Abs(posSum / total)
		neg = math.Abs(negSum / total)
		neu = math.Abs(float64(neuCount) / total)
	}

	sentimentDict := map[string]float64{
		"pos":      floats.Round(pos, 3),
		"neg":      floats.Round(neg, 3),
		"neu":      floats.Round(neu, 3),
		"compound": floats.Round(compound, 4),
	}

	return sentimentDict
}

func main() {
	// --- Examples -------
	sia := SentimentIntensityAnalyzer{}
	sia.Init()

	sentences := []string{"VADER is smart, handsome, and funny.", // positive sentence example
		"VADER is smart, handsome, and funny!", // punctuation emphasis handled correctly (sentiment intensity adjusted)
		"VADER is very smart, handsome, and funny.", // booster words handled correctly (sentiment intensity adjusted)
		"VADER is VERY SMART, handsome, and FUNNY.", // emphasis for ALLCAPS handled
		"VADER is VERY SMART, handsome, and FUNNY!!!", // combination of signals - VADER appropriately adjusts intensity
		"VADER is VERY SMART, uber handsome, and FRIGGIN FUNNY!!!", // booster words & punctuation make this close to ceiling for score
		"VADER is not smart, handsome, nor funny.", // negation sentence example
		"The book was good.",                       // positive sentence
		"At least it isn't a horrible book.",       // negated negative sentence with contraction
		"The book was only kind of good.", // qualified positive sentence is handled correctly (intensity adjusted)
		"The plot was good, but the characters are uncompelling and the dialog is not great.", // mixed negation sentence
		"Today SUX!", // negative slang with capitalization emphasis
		"Today only kinda sux! But I'll get by, lol", // mixed sentiment example with slang and constrastive conjunction "but"
		"Make sure you :) or :D today!",              // emoticons handled
		"Catch utf-8 emoji such as 💘 and 💋 and 😁", // emojis handled
		"Not bad at all",                              // Capitalized negation
	}

	fmt.Println("----------------------------------------------------")
	fmt.Println(" - Analyze typical example cases, including handling of:")
	fmt.Println("  -- negations")
	fmt.Println("  -- punctuation emphasis & punctuation flooding")
	fmt.Println("  -- word-shape as emphasis (capitalization difference)")
	fmt.Println("  -- degree modifiers (intensifiers such as 'very' and dampeners such as 'kind of')")
	fmt.Println("  -- slang words as modifiers such as 'uber' or 'friggin' or 'kinda'")
	fmt.Println("  -- contrastive conjunction 'but' indicating a shift in sentiment; sentiment of later text is dominant")
	fmt.Println("  -- use of contractions as negations")
	fmt.Println("  -- sentiment laden emoticons such as :) and :D")
	fmt.Println("  -- utf-8 encoded emojis such as 💘 and 💋 and 😁")
	fmt.Println("  -- sentiment laden slang words (e.g., 'sux'")
	fmt.Printf("  -- sentiment laden initialisms and acronyms (for example: 'lol')\n\n")

	for _, sentence := range sentences{
		score := sia.polarityScores(sentence)
		fmt.Printf("%s : %+v\n", sentence, score)
	}

	fmt.Println("----------------------------------------------------")
	fmt.Println(" - About the scoring: ")
	fmt.Println(` -- The 'compound' score is computed by summing the valence scores of each word in the lexicon, adjusted
	according to the rules, and then normalized to be between -1 (most extreme negative) and +1 (most extreme positive).
	This is the most useful metric if you want a single unidimensional measure of sentiment for a given sentence.
	Calling it a 'normalized, weighted composite score' is accurate. `)

	fmt.Println(`  -- The 'pos', 'neu', and 'neg' scores are ratios for proportions of text that fall in each category (so these
	should all add up to be 1... or close to it with float operation).  These are the most useful metrics if
	you want multidimensional measures of sentiment for a given sentence.`)

	fmt.Println("----------------------------------------------------")

	tricky_sentences := []string{"Sentiment analysis has never been good.",
		"Sentiment analysis has never been this good!",
		"Most automated sentiment analysis tools are shit.",
		"With VADER, sentiment analysis is the shit!",
		"Other sentiment analysis tools can be quite bad.",
		"On the other hand, VADER is quite bad ass",
		"VADER is such a badass!", // slang with punctuation emphasis
		"Without a doubt, excellent idea.",
		"Roger Dodger is one of the most compelling variations on this theme.",
		"Roger Dodger is at least compelling as a variation on the theme.",
		"Roger Dodger is one of the least compelling variations on this theme.",
		"Not such a badass after all.", // Capitalized negation with slang
		"Without a doubt, an excellent idea.",  // "without {any} doubt" as negation
	}

	fmt.Println(" - Analyze examples of tricky sentences that cause trouble to other sentiment analysis tools.")
	fmt.Println("  -- special case idioms - e.g., 'never good' vs 'never this good', or 'bad' vs 'bad ass'.")
	fmt.Printf("  -- special uses of 'least' as negation versus comparison\n\n")

	for _, sentence := range tricky_sentences{
		score := sia.polarityScores(sentence)
		fmt.Printf("%s : %+v\n", sentence, score)
	}

}
