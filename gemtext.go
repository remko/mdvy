package main

import (
	"bufio"
	"fmt"
	"html"
	"io"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

type Gemtext []Node

type Node interface {
	Line() int
	Equal(o Node) bool
}

type node struct {
	line int
}

func (n node) Line() int {
	return n.line
}

type Paragraph struct {
	node
	Text string
}

func (n *Paragraph) Equal(o Node) bool {
	if o, ok := o.(*Paragraph); ok {
		return n.Text == o.Text
	}
	return false
}

type Link struct {
	node
	URL   string
	Label string
}

func (n *Link) Equal(o Node) bool {
	if o, ok := o.(*Link); ok {
		return n.URL == o.URL && n.Label == o.Label
	}
	return false
}

type Heading struct {
	node
	Level int
	Text  string
}

func (n *Heading) Equal(o Node) bool {
	if o, ok := o.(*Heading); ok {
		return n.Level == o.Level && n.Text == o.Text
	}
	return false
}

type List struct {
	node
	Items []*Paragraph
}

func (n *List) Equal(o Node) bool {
	if o, ok := o.(*List); ok {
		return slices.EqualFunc(n.Items, o.Items, func(a *Paragraph, b *Paragraph) bool {
			return a.Equal(b)
		})
	}
	return false
}

type Quote struct {
	node
	Paragraphs []*Paragraph
}

func (n *Quote) Equal(o Node) bool {
	if o, ok := o.(*Quote); ok {
		return slices.EqualFunc(n.Paragraphs, o.Paragraphs, func(a *Paragraph, b *Paragraph) bool {
			return a.Equal(b)
		})
	}
	return false
}

type Pre struct {
	node
	Alt        string
	Paragraphs []*Paragraph
}

func (n *Pre) Equal(o Node) bool {
	if o, ok := o.(*Pre); ok {
		return slices.EqualFunc(n.Paragraphs, o.Paragraphs, func(a *Paragraph, b *Paragraph) bool {
			return a.Equal(b)
		})
	}
	return false
}

func ParseGemtext(r io.Reader) (Gemtext, error) {
	var result = []Node{}
	scn := bufio.NewScanner(r)
	scn.Split(bufio.ScanLines)
	pre := false
	var prev Node
	line := 0
	for scn.Scan() {
		line += 1
		text := scn.Text()
		node := node{line: line}
		if pre {
			if strings.HasPrefix(text, "```") {
				pre = false
			} else {
				prev.(*Pre).Paragraphs = append(prev.(*Pre).Paragraphs, &Paragraph{node: node, Text: text})
			}
		} else {
			var ok bool
			if strings.HasPrefix(text, ">") {
				var q *Quote
				if q, ok = prev.(*Quote); !ok {
					q = &Quote{node: node, Paragraphs: []*Paragraph{}}
					result = append(result, q)
					prev = q
				}
				q.Paragraphs = append(q.Paragraphs, &Paragraph{node: node, Text: text})
			} else if strings.HasPrefix(text, "* ") {
				var q *List
				if q, ok = prev.(*List); !ok {
					q = &List{node: node, Items: []*Paragraph{}}
					result = append(result, q)
					prev = q
				}
				q.Items = append(q.Items, &Paragraph{node: node, Text: strings.TrimLeftFunc(text[2:], unicode.IsSpace)})
			} else if strings.HasPrefix(text, "# ") {
				prev = &Heading{node: node, Level: 1, Text: strings.TrimSpace(text[1:])}
				result = append(result, prev)
			} else if strings.HasPrefix(text, "## ") {
				prev = &Heading{node: node, Level: 2, Text: strings.TrimSpace(text[2:])}
				result = append(result, prev)
			} else if strings.HasPrefix(text, "### ") {
				prev = &Heading{node: node, Level: 3, Text: strings.TrimSpace(text[3:])}
				result = append(result, prev)
			} else if strings.HasPrefix(text, "=> ") {
				url := strings.TrimSpace(text[3:])
				var label string
				i := strings.IndexFunc(url, unicode.IsSpace)
				if i > 0 {
					label = strings.TrimLeftFunc(url[i+1:], unicode.IsSpace)
					url = url[:i]
				}
				prev = &Link{node: node, URL: url, Label: label}
				result = append(result, prev)
			} else if strings.HasPrefix(text, "```") {
				pre = true
				prev = &Pre{node: node, Alt: text[3:], Paragraphs: []*Paragraph{}}
				result = append(result, prev)
			} else {
				prev = &Paragraph{node: node, Text: text}
				result = append(result, prev)
			}
		}
	}
	return result, scn.Err()
}

var linkIcon = `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="icon" viewBox="0 0 16 16"><path d="M6.354 5.5H4a3 3 0 0 0 0 6h3a3 3 0 0 0 2.83-4H9c-.086 0-.17.01-.25.031A2 2 0 0 1 7 10.5H4a2 2 0 1 1 0-4h1.535c.218-.376.495-.714.82-1z"/><path d="M9 5.5a3 3 0 0 0-2.83 4h1.098A2 2 0 0 1 9 6.5h3a2 2 0 1 1 0 4h-1.535a4.02 4.02 0 0 1-.82 1H12a3 3 0 1 0 0-6z"/></svg>`

func writeEl(w io.Writer, tag string, attrs map[string]string) {
	io.WriteString(w, "<")
	io.WriteString(w, tag)
	for k, v := range attrs {
		io.WriteString(w, " ")
		io.WriteString(w, k)
		io.WriteString(w, "=\"")
		io.WriteString(w, html.EscapeString(v))
		io.WriteString(w, "\"")
	}
	io.WriteString(w, ">")
}

func GemtextToHTML(gt Gemtext, pgt Gemtext, w io.Writer) error {
	i := 0
	for _, n := range gt {
		// Search for a node
		changed := false
		if pgt != nil {
			if p, ok := n.(*Paragraph); ok && strings.TrimSpace(p.Text) == "" {
				// Ignore empty paragraphs
			} else {
				found := false
				for j := i; j < len(pgt); j++ {
					if n.Equal(pgt[j]) {
						found = true
						i = j
						break
					}
				}
				if !found {
					changed = true
				}
			}
		}

		attrs := map[string]string{"data-line": strconv.Itoa(n.Line())}
		if changed {
			attrs["class"] = "changed"
		}
		switch node := n.(type) {
		case *Paragraph:
			writeEl(w, "p", attrs)
			io.WriteString(w, html.EscapeString(node.Text))
			io.WriteString(w, "</p>")
		case *Link:
			writeEl(w, "div", attrs)
			io.WriteString(w, linkIcon)
			io.WriteString(w, " ")
			io.WriteString(w, fmt.Sprintf("<a href=\"%s\">", html.EscapeString(node.URL)))
			if node.Label != "" {
				io.WriteString(w, html.EscapeString(node.Label))
			} else {
				io.WriteString(w, html.EscapeString(node.URL))
			}
			io.WriteString(w, "</a>")
			io.WriteString(w, "</div>")
		case *Heading:
			writeEl(w, fmt.Sprintf("h%d", node.Level), attrs)
			io.WriteString(w, html.EscapeString(node.Text))
			io.WriteString(w, fmt.Sprintf("</h%d>", node.Level))
		case *List:
			writeEl(w, "ul", attrs)
			for _, p := range node.Items {
				attrs := map[string]string{"data-line": strconv.Itoa(p.line)}
				writeEl(w, "li", attrs)
				io.WriteString(w, "<li>")
				io.WriteString(w, p.Text)
				io.WriteString(w, "</li>")
			}
			io.WriteString(w, "</ul>")
		case *Quote:
			io.WriteString(w, "<blockquote>")
			for _, p := range node.Paragraphs {
				attrs := map[string]string{"data-line": strconv.Itoa(p.line)}
				writeEl(w, "p", attrs)
				io.WriteString(w, p.Text)
				io.WriteString(w, "</p>")
			}
			io.WriteString(w, "</blockquote>")
		case *Pre:
			writeEl(w, "pre", attrs)
			for _, p := range node.Paragraphs {
				io.WriteString(w, p.Text)
				io.WriteString(w, "\n")
			}
			io.WriteString(w, "</pre>")
		}
		io.WriteString(w, "\n")
	}
	return nil
}
