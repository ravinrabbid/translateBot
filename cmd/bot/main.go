package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"github.com/BurntSushi/toml"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
)

type twitterAuth struct {
	ConsumerKey    string `toml:"consumerKey"`
	ConsumerSecret string `toml:"consumerSecret"`
	AccessKey      string `toml:"accessKey"`
	AccessSecret   string `toml:"accessSecret"`
}

type config struct {
	Twitter twitterAuth `toml:"twitter"`
}

type translation struct {
	Response struct {
		Text string `json:"translatedText"`
	} `json:"responseData"`
}

var configFile = flag.String("config", "", "Path to configuration file.")
var conf config

var translateURL = "http://api.mymemory.translated.net/get?langpair=ja|de&q="

func getTranslation(s string) string {
	resp, err := http.Get(translateURL + url.QueryEscape(s))
	if err != nil {
		log.Fatalf("Could not reach translation service: %v", err.Error())
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Could not reach translation service: %v", err.Error())
	}

	var trans translation
	err = json.Unmarshal(body, &trans)
	if err != nil {
		log.Fatalf("Could not parse translation: %v", err.Error())
	}

	return trans.Response.Text
}

func main() {
	flag.Parse()
	if *configFile == "" {
		log.Fatal("missing -config")
	}

	_, err := toml.DecodeFile(*configFile, &conf)
	if err != nil {
		log.Fatalf("Could not read config: %v", err.Error())
	}

	oauthConfig := oauth1.NewConfig(conf.Twitter.ConsumerKey, conf.Twitter.ConsumerSecret)
	oauthToken := oauth1.NewToken(conf.Twitter.AccessKey, conf.Twitter.AccessSecret)

	httpClient := oauthConfig.Client(oauth1.NoContext, oauthToken)
	client := twitter.NewClient(httpClient)

	tweets, _, err := client.Timelines.HomeTimeline(&twitter.HomeTimelineParams{})
	if err != nil {
		log.Fatalf("Could not retrieve timeline: %v", err.Error())
	}

	for _, tweet := range tweets {
		fmt.Printf("%v\n=====================\n", tweet.Text)
	}
}
