package main

import (
	"fmt"
	"log"

	"github.com/drankou/go-vader/vader"
)

func main() {
	// --- Examples -------
	sia := vader.SentimentIntensityAnalyzer{}
	err := sia.Init()
	if err != nil {
		log.Fatal(err)
	}

	sentences := []string{"VADER is smart, handsome, and funny.",                              // positive sentence example
		"VADER is smart, handsome, and funny!",                                                // punctuation emphasis handled correctly (sentiment intensity adjusted)
		"VADER is very smart, handsome, and funny.",                                           // booster words handled correctly (sentiment intensity adjusted)
		"VADER is VERY SMART, handsome, and FUNNY.",                                           // emphasis for ALLCAPS handled
		"VADER is VERY SMART, handsome, and FUNNY!!!",                                         // combination of signals - VADER appropriately adjusts intensity
		"VADER is VERY SMART, uber handsome, and FRIGGIN FUNNY!!!",                            // booster words & punctuation make this close to ceiling for score
		"VADER is not smart, handsome, nor funny.",                                            // negation sentence example
		"The book was good.",                                                                  // positive sentence
		"At least it isn't a horrible book.",                                                  // negated negative sentence with contraction
		"The book was only kind of good.",                                                     // qualified positive sentence is handled correctly (intensity adjusted)
		"The plot was good, but the characters are uncompelling and the dialog is not great.", // mixed negation sentence
		"Today SUX!", // negative slang with capitalization emphasis
		"Today only kinda sux! But I'll get by, lol", // mixed sentiment example with slang and constrastive conjunction "but"
		"Make sure you :) or :D today!",              // emoticons handled
		"Catch utf-8 emoji such as 💘 and 💋 and 😁", // emojis handled
		"Not bad at all",                             // Capitalized negation
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

	for _, sentence := range sentences {
		score := sia.PolarityScores(sentence)
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
		"Not such a badass after all.",        // Capitalized negation with slang
		"Without a doubt, an excellent idea.", // "without {any} doubt" as negation
	}

	fmt.Println(" - Analyze examples of tricky sentences that cause trouble to other sentiment analysis tools.")
	fmt.Println("  -- special case idioms - e.g., 'never good' vs 'never this good', or 'bad' vs 'bad ass'.")
	fmt.Printf("  -- special uses of 'least' as negation versus comparison\n\n")

	for _, sentence := range tricky_sentences {
		score := sia.PolarityScores(sentence)
		fmt.Printf("%s : %+v\n", sentence, score)
	}

}
