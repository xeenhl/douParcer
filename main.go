package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

type Comment struct {
	User       string
	ReplyDepth int
	Tokens     []string
	ToxicRate  int
}

type matcher func(val1, val2 string) bool

func main() {
	//resp, err := http.Get("https://dou.ua/forums/topic/27292/")
	resp, err := http.Get("https://dou.ua/forums/topic/28029/")

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

	comentaries := parceToComentaries(comms)
	fmt.Println("Comentaries found: ", len(comentaries))

	saveComentsTokensToFile(comentaries)

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
		if len(nodesApnd) > 0 {
			nodes = append(nodes, nodesApnd...)
		}
	}

	return nodes
}

func parceToComentaries(nodes []*html.Node) []Comment {
	coms := make([]Comment, 0, 0)

	for _, comment := range nodes {
		c := Comment{}
		c.Tokens = getTokensFromNode(comment)
		c.User = getUserFromNode(comment)
		c.ReplyDepth = getReplyDepFromNode(comment)

		coms = append(coms, c)
	}

	return coms
}

func getTokensFromNode(node *html.Node) []string {
	coms := make([]string, 0, 0)

	tn := findAllByAttributeMathcer(node, "class", "text", contains)

	for n := tn[0].FirstChild; n != nil; n = n.NextSibling {
		if n.FirstChild != nil {
			coms = append(coms, strings.Fields(n.FirstChild.Data)...)
		}
	}
	return coms
}

func getUserFromNode(node *html.Node) string {
	tn := findAllByAttributeMathcer(node, "class", "avatar", contains)
	v := strings.Split(tn[0].Attr[1].Val, "/")
	return v[4]
}

func getReplyDepFromNode(node *html.Node) int {
	level := strings.Split(node.Attr[0].Val, "-")
	v, _ := strconv.Atoi(level[2])
	return v
}

func uniqeTokens(tokens []string, m map[string]bool) []string {
	u := make([]string, 0, 0)

	for _, val := range tokens {
		if _, ok := m[val]; !ok {
			m[val] = true
			u = append(u, val)
		}
	}

	return u
}

func saveComentsTokensToFile(comentaries []Comment) {
	f, err := os.OpenFile("out.tokens", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)

	if err != nil {
		panic(err)
	}

	defer f.Close()

	t := make([]string, 0, 0)
	for _, c := range comentaries {
		t = append(t, c.Tokens...)
	}

	m := make(map[string]bool)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		m[scanner.Text()] = true
	}

	out := uniqeTokens(t, m)

	for _, tok := range out {
		_, err := f.WriteString(tok + "\n")
		if err != nil {
			panic(err)
		}
	}

	fmt.Println("Saved new uniqe tokens: ", len(out))
}

func contains(sourse, match string) bool {
	return strings.Contains(sourse, match)
}
