package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"path"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/browser"
	webview "github.com/webview/webview_go"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

//go:embed style.css
var style string

//go:embed script.js
var script string

var tmpl = template.Must(template.New("index").Parse(`
<style>{{.Style}}</style>
<body>
	<div id="content"></div>
	<script>{{.Script}}</script>
</body>
`))

type View struct {
	source string
	md     goldmark.Markdown
	wv     webview.WebView
	fsw    *fsnotify.Watcher
	gt     Gemtext
}

func NewView(source string) (*View, error) {
	var md goldmark.Markdown
	if !strings.HasSuffix(source, ".gmi") {
		md = goldmark.New(
			goldmark.WithExtensions(extension.GFM, extension.Typographer),
			// goldmark.WithParserOptions(
			// 	parser.WithAutoHeadingID(),
			// ),
			goldmark.WithRendererOptions(
				html.WithUnsafe()),
		)
	}

	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	err = fsw.Add(path.Dir(source))
	if err != nil {
		return nil, err
	}

	wv := webview.New(true)
	wv.SetTitle(source)
	wv.SetSize(600, 800, webview.HintNone)

	var html bytes.Buffer
	err = tmpl.Execute(&html, struct {
		Style  template.CSS
		Script template.JS
	}{Style: template.CSS(style), Script: template.JS(script)})
	if err != nil {
		return nil, err
	}
	wv.SetHtml(string(html.Bytes()))

	view := &View{
		source: source,
		md:     md,
		fsw:    fsw,
		wv:     wv,
	}

	err = wv.Bind("onReady", func() {
		err = view.render()
		if err != nil {
			log.Printf("render error: %v", err)
		}
	})
	if err != nil {
		return nil, err
	}
	err = wv.Bind("openURL", func(url string) error {
		return browser.OpenURL(url)
	})
	if err != nil {
		return nil, err
	}
	err = wv.Bind("quit", func() {
		wv.Terminate()
	})
	if err != nil {
		return nil, err
	}

	return view, nil
}

func (v *View) Run() {
	go v.watch()
	v.wv.Run()
	v.fsw.Close()
	v.wv.Destroy()
}

func (v *View) render() error {
	inputf, err := os.Open(v.source)
	if err != nil {
		return err
	}

	var content bytes.Buffer
	if v.md != nil {
		input, err := io.ReadAll(inputf)
		if err != nil {
			return err
		}
		if err := v.md.Convert(input, &content); err != nil {
			return err
		}
	} else {
		gt, err := ParseGemtext(inputf)
		if err != nil {
			return err
		}
		if err := GemtextToHTML(gt, v.gt, &content); err != nil {
			return err
		}
		v.gt = gt
		// log.Printf("%s", gt, content.String())
	}

	// log.Printf("html: %s", content)
	contentjson, err := json.Marshal(string(content.Bytes()))
	if err != nil {
		return err
	}
	eval := fmt.Sprintf(`setContent(%s)`, contentjson)
	v.wv.Dispatch(func() {
		v.wv.Eval(eval)
	})
	return nil

}

func (v *View) watch() {
	for {
		select {
		case event, ok := <-v.fsw.Events:
			if !ok {
				return
			}
			log.Println("event:", event)
			if event.Name == v.source && (event.Has(fsnotify.Write) || event.Has(fsnotify.Create)) {
				err := v.render()
				if err != nil {
					log.Printf("render error: %v", err)
				}
			}

		case err, ok := <-v.fsw.Errors:
			if !ok {
				return
			}
			log.Println("watcher error:", err)
		}
	}
}

func main_() error {
	flag.Parse()
	if len(flag.Args()) == 0 {
		return errors.New("missing file")
	}
	inputp := flag.Args()[0]
	view, err := NewView(inputp)
	if err != nil {
		return err
	}
	view.Run()
	return nil
}

func main() {
	if err := main_(); err != nil {
		fmt.Printf("error: %s", err.Error())
		os.Exit(-1)
	}
}
