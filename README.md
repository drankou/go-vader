# GoVader. Sentiment analysis tool written in GO (GoLang).

"VADER (Valence Aware Dictionary and sEntiment Reasoner) is a lexicon and rule-based sentiment analysis tool that is specifically attuned to sentiments expressed in social media."

Original python implementation of VADER (https://github.com/cjhutto/vaderSentiment).

# Getting started

## Install:

`go get github.com/drankou/go-vader`


## Example of usage:
````
sia := vader.SentimentIntensityAnalyzer{}
err := sia.Init()
if err != nil {
    log.Fatal(err)
}

score := sia.PolarityScores("VADER is smart, handsome, and funny!")
fmt.Println(score)
//output: map[pos:0.746 neg:0 neu:0.254 compound:0.8316]

````

## Accuracy on twitter dataset (200k tweets: 100k positive and 100k negative):
```
Total number of analyzed tweets:  200000
The accuracy of tested sentiment is: 51.0%
2019/06/20 17:41:17 False positive: 15.8%
2019/06/20 17:41:17 False negative: 5.07%
````
