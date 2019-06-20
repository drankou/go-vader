##GoVader. Sentiment analysis tool written in GoLang.

"VADER (Valence Aware Dictionary and sEntiment Reasoner) is a lexicon and rule-based sentiment analysis tool that is specifically attuned to sentiments expressed in social media."

Original python implementation of VADER (https://github.com/cjhutto/vaderSentiment).

#Getting started

##Install:

`go get github.com/drankou/go-vader`


##Example of usage:
````
sia := SentimentIntensityAnalyzer{}
err := sia.Init()
if err != nil {
    log.Fatal(err)
}

score := sia.PolarityScores("VADER is smart, handsome, and funny!")
fmt.Println(score)
//output: map[pos:0.746 neg:0 neu:0.254 compound:0.8316]

````



