package main

import (
	"bufio"
	"errors"
	"flag"
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

var endings = [4]string{",", ".", "?", "!"}
var tokens = make(map[string]int)
var mode = flag.String("mode", "validate", "help message for flagname")

const (
	sp           = "&"
	validateMode = "validate"
	teachMode    = "teach"
)

func main() {
	flag.Parse()
	uploadTokens(tokens, "out.tokens")
	fmt.Println("Upladed tokens: ", len(tokens))

	//"https://dou.ua/forums/topic/27292/"
	comentaries := populateComentaries("https://dou.ua/forums/topic/28061/")
	fmt.Println("Comentaries found: ", len(comentaries))

	if *mode == validateMode {
		toxComsC := 0
		comentaries = validateCometraries(comentaries)
		for _, com := range comentaries {
			if com.ToxicRate > 0 {
				fmt.Println(com)
				toxComsC++
			}
		}
		fmt.Printf("Found %v toxic comments\n", toxComsC)
	}

	if *mode == teachMode {
		teachModel(comentaries)
	}

	saveComentsTokensToFile(comentaries, "out.tokens")

}

func validateCometraries(comentaries []Comment) []Comment {
	fmt.Println("In validation mode")
	calulated := make([]Comment, 0, 0)
	for _, com := range comentaries {
		com.ToxicRate = calculateToxic(com.Tokens, com.ReplyDepth)
		calulated = append(calulated, com)
	}
	return calulated
}

func calculateToxic(toks []string, depth int) int {
	toxrate := depth
	for _, tok := range toks {
		if toxr, ok := tokens[tok]; ok {
			toxrate += toxr
		}
	}
	return toxrate
}

func teachModel(comentaries []Comment) {
	fmt.Println("In teaching mode")

	reader := bufio.NewReader(os.Stdin)
	for _, comment := range comentaries {

		fmt.Println(comment)
		fmt.Print("Is this commentary count as toxic (true/false): ")
		eval, _ := reader.ReadString('\n')
		eval = strings.Trim(eval, " \r\n")
		toxic, err := strconv.ParseBool(eval)
		if err != nil {
			toxic = false
		}
		updateTokensToxic(comment.Tokens, toxic)
	}

}

func uploadTokens(tokens map[string]int, file string) {
	f, err := os.OpenFile(file, os.O_RDONLY, 0644)

	if err != nil {
		panic(err)
	}

	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		tok, tox, err := getTokenToxic(scanner.Text())
		if err == nil {
			tokens[tok] = tox
		}
	}
}

func getTokenToxic(rawToken string) (string, int, error) {
	data := strings.Split(rawToken, sp)
	if len(data) == 1 {
		return data[0], 0, nil
	}

	if len(data) == 2 {
		tox, err := strconv.ParseInt(data[1], 10, 64)

		if err != nil {
			return "", 0, err
		}

		return data[0], int(tox), nil
	}

	return "", 0, errors.New(fmt.Sprintf("Cant process tpoken: %v", rawToken))
}

func populateComentaries(url string) []Comment {

	resp, err := http.Get(url)

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

	return parceToComentaries(comms)
}

func updateTokensToxic(toks []string, increase bool) {
	for _, tok := range toks {
		if increase {
			tokens[tok] += 1
		}
		if !increase {
			tokens[tok] -= 1
		}
	}
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

func addUniqeTokens(tokens []string, tokensMap map[string]int) int {
	newToc := 0
	for _, val := range tokens {
		val = removeEndings(val)
		if _, ok := tokensMap[val]; !ok {
			tokensMap[val] = 0
			newToc++
		}
	}
	return newToc
}

func saveComentsTokensToFile(comentaries []Comment, file string) {
	f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)

	if err != nil {
		panic(err)
	}

	defer f.Close()

	t := make([]string, 0, 0)
	for _, c := range comentaries {
		t = append(t, c.Tokens...)
	}

	newTokens := addUniqeTokens(t, tokens)

	f.Truncate(0)
	f.Seek(0, 0)

	for tok, tox := range tokens {
		_, err := f.WriteString(tok + sp + strconv.Itoa(tox) + "\n")
		if err != nil {
			panic(err)
		}
	}

	fmt.Println("Saved new uniqe tokens: ", newTokens)
}

func removeEndings(token string) string {
	for _, ending := range endings {
		if strings.HasSuffix(token, ending) {
			return token[:len(token)-1]
		}
	}
	return token
}

func contains(sourse, match string) bool {
	return strings.Contains(sourse, match)
}
