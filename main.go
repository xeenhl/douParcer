package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

type Comment struct {
	User struct {
		Id   int
		Name string
	}
	Text string
}

type matcher func(val1, val2 string) bool

func main() {
	resp, err := http.Get("https://dou.ua/forums/topic/27292/")

	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		panic(err)
	}

	n, err := html.Parse(strings.NewReader(string(body)))

	if err != nil {
		panic(err)
	}

	comList := findByAttribute(n, "id", "commentsList")
	comms := findAllByAttributeMathcer(comList, "class", "b-comment", contains)

	fmt.Println(len(comms))
}

func findByAttribute(root *html.Node, attribute, value string) *html.Node {

	for _, attr := range root.Attr {
		if attr.Key == attribute && attr.Val == value {
			return root
		}
	}

	for n := root.FirstChild; n != nil; n = n.NextSibling {
		node := findByAttribute(n, attribute, value)
		if node != nil {
			return node
		}
	}

	return nil
}

func findAllByAttributeMathcer(root *html.Node, attribute, value string, mtch matcher) []*html.Node {
	nodes := make([]*html.Node, 0, 0)
	for _, attr := range root.Attr {
		if attr.Key == attribute && mtch(attr.Val, value) {
			nodes = append(nodes, root)
		}
	}

	for n := root.FirstChild; n != nil; n = n.NextSibling {
		nodesApnd := findAllByAttributeMathcer(n, attribute, value, mtch)
		if len(nodes) > 0 {
			nodes = append(nodes, nodesApnd...)
		}
	}

	return nodes
}

func contains(sourse, match string) bool {
	return strings.Contains(sourse, match)
}
