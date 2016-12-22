package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"flag"
	"fmt"
	"log"
	"net/url"
	"time"
)

var (
	baseURL string

	campaign string
	content  string
	medium   string
	source   string

	lang    string
	os      string
	product string

	hmacKey string
)

func init() {
	flag.StringVar(&baseURL, "baseurl", "https://stubattribution-default.stage.mozaws.net", "base stub attribution service url")

	flag.StringVar(&campaign, "campaign", "testcampaign", "campaign")
	flag.StringVar(&content, "content", "testcontent", "content")
	flag.StringVar(&medium, "medium", "testmedium", "medium")
	flag.StringVar(&source, "source", "mozilla.com", "source")

	flag.StringVar(&lang, "lang", "en-US", "")
	flag.StringVar(&os, "os", "win", "")
	flag.StringVar(&product, "product", "test-stub", "")

	flag.StringVar(&hmacKey, "hmackey", "testkey", "test hmac key")
}

func genCode() string {
	query := url.Values{}
	query.Set("campaign", campaign)
	query.Set("content", content)
	query.Set("medium", medium)
	query.Set("source", source)
	query.Set("timestamp", fmt.Sprintf("%d", time.Now().UTC().Unix()))
	return query.Encode()
}

func hmacSig(code string) string {
	mac := hmac.New(sha256.New, []byte(hmacKey))
	mac.Write([]byte(code))
	return fmt.Sprintf("%x", mac.Sum(nil))
}

func genURL(code, sig string) string {
	query := url.Values{}
	query.Set("attribution_code", code)
	query.Set("attribution_sig", sig)

	query.Set("lang", lang)
	query.Set("os", os)
	query.Set("product", product)

	u, err := url.Parse(baseURL)
	if err != nil {
		log.Fatal("Could not parse url:", err)
	}
	u.RawQuery = query.Encode()
	return u.String()
}

func main() {
	flag.Parse()
	code := genCode()
	sig := hmacSig(url.QueryEscape(code))
	fmt.Println(genURL(code, sig))
}
