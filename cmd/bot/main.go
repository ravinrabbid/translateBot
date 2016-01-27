package main

import (
	"encoding/json"
	// "errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
)

type botConfig struct {
	TranslatedUser string `toml:"translatedUser"`
	SourceLanguage string `toml:"sourceLanguage"`
	TargetLanguage string `toml:"targetLanguage"`
}

type twitterAuth struct {
	ConsumerKey    string `toml:"consumerKey"`
	ConsumerSecret string `toml:"consumerSecret"`
	AccessKey      string `toml:"accessKey"`
	AccessSecret   string `toml:"accessSecret"`
}

type config struct {
	Bot     botConfig   `toml:"bot"`
	Twitter twitterAuth `toml:"twitter"`
}

type translation struct {
	Response struct {
		Text string `json:"translatedText"`
	} `json:"responseData"`
	// Status string `json:"responseStatus"`
}

var configFile = flag.String("config", "", "Path to configuration file.")
var conf config

var translateBaseURL = "http://api.mymemory.translated.net/get?"
var translateURL = ""

//TODO strip urls/usernames
func getTranslation(s string) (string, error) {
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

	// if trans.Status != "200" {
	// 	return "", errors.New(trans.Response.Text)
	// }

	text, _ := url.QueryUnescape(trans.Response.Text) //TODO make this work

	return text, nil
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

	translateURL = translateBaseURL + "langpair=" + conf.Bot.SourceLanguage + "|" + conf.Bot.TargetLanguage + "&q="

	oauthConfig := oauth1.NewConfig(conf.Twitter.ConsumerKey, conf.Twitter.ConsumerSecret)
	oauthToken := oauth1.NewToken(conf.Twitter.AccessKey, conf.Twitter.AccessSecret)

	httpClient := oauthConfig.Client(oauth1.NoContext, oauthToken)
	client := twitter.NewClient(httpClient)

	demux := twitter.NewSwitchDemux()
	demux.Tweet = func(tweet *twitter.Tweet) {
		log.Printf("Got tweet: '%v' by %v", tweet.Text, tweet.User.ScreenName)
		//TODO omit retweets without comment
		if tweet.User.ScreenName == conf.Bot.TranslatedUser && tweet.InReplyToStatusID == 0 {
			log.Print("Translating...")
			translation, err := getTranslation(tweet.Text)
			if err != nil {
				log.Printf("Translation failed: %v", err.Error())
			} else {
				client.Statuses.Update("@"+tweet.User.Name+" "+translation, &twitter.StatusUpdateParams{InReplyToStatusID: tweet.ID})
			}
		}
	}

	userParams := &twitter.StreamUserParams{
		StallWarnings: twitter.Bool(true),
		With:          "followings",
	}
	stream, err := client.Streams.User(userParams)
	if err != nil {
		log.Fatal(err)
	}

	go demux.HandleChan(stream.Messages)

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	log.Println(<-ch)

	fmt.Println("Stopping Stream...")
	stream.Stop()
}
