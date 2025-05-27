package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

const baseURL string = "https://scrape-me.dreamsofcode.io/"

type LinkState string

const (
	Dead  LinkState = "Dead"
	Alive LinkState = "Alive"
)

var (
	allLinks = make(map[string]LinkState)
	wg       sync.WaitGroup
	mu       sync.Mutex
)

// time without go routine: 13.1114
// time with go routine:     5.8767

func main() {
	start := time.Now()

	wg.Add(1)

	go getBody(baseURL)

	fmt.Println("Loading")
	wg.Wait()

	mu.Lock()
	for key, val := range allLinks {
		if val == Alive {
			fmt.Println("Alive links: ", key)
		}
	}

	for key, val := range allLinks {
		if val == Dead {
			fmt.Println("Dead links: ", key)
		}
	}
	mu.Unlock()

	fmt.Println("Time taken: ", time.Since(start))
}

func getBody(url string) {
	defer wg.Done()
	if _, found := allLinks[url]; found {
		return
	}
	res, err := http.Get(url)
	if err != nil {
		log.Fatalln(err)
	}

	defer res.Body.Close()
	mu.Lock()
	if res.StatusCode != 200 {
		allLinks[url] = Dead
	} else {
		allLinks[url] = Alive
	}
	mu.Unlock()

	doc, err := html.Parse(res.Body)
	if err != nil {
		log.Fatalln("Error while parsing html: ", err)
	}
	links, err := getAllLinks(doc)
	if err != nil {
		log.Fatalln(err)
	}
	for _, link := range links {
		wg.Add(1)
		go getBody(link)
	}
}

func getAllLinks(n *html.Node) ([]string, error) {
	var links []string
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, attr := range n.Attr {
			key, value := attr.Key, attr.Val
			if key == "href" && isRelative(value) {
				f, err := url.JoinPath(baseURL, value)
				if err != nil {
					return nil, err
				}
				links = append(links, f)
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		l, err := getAllLinks(c)
		if err != nil {
			return nil, err
		}
		links = append(links, l...)
	}
	return links, nil
}

func isRelative(link string) bool {
	host, err := url.Parse(baseURL)
	if err != nil {
		return false
	}
	if len(link) == 0 {
		return true
	}
	p, err := url.Parse(link)
	if err != nil {
		return false
	}
	if p.Host == host.Host {
		return true
	}
	if strings.HasPrefix(link, "/") {
		return true
	}
	return false
}
