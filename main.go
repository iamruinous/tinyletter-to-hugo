// A quick script for converting tinyletter HTML files to Markdown, suitable for use in a static file generator such as Hugo or Jekyll
//A fork of https://gist.github.com/clipperhouse/010d4666892807afee16ba7711b41401
package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/lunny/html2md"

	"github.com/PuerkitoBio/goquery"
  "github.com/araddon/dateparse"
)

type post struct {
	Title, Author, Body   string
	Date, Lastmod         string
	Subtitle, Description string
	Canonical, FullURL    string
	FeaturedImage         string
	Images                []string
	Tags                  []string
	HddFolder             string
	Draft                 bool
	IsComment             bool
}

func main() {

	if len(os.Args) != 4 {
		fmt.Println("usage: path/to/tinyletter-export-folder/posts/ path/to/hugo/content/ content-type")
		fmt.Println("example: ./tinylettertohugo ~/Downloads/tinyletter/posts/ /srv/go/myblog/ posts")
		os.Exit(1)
	}
	// Location of exported, unzipped tinyletter HTML files
	var postsHTMLFolder = os.Args[1]

	// Destination for Markdown files, perhaps the content folder for Hugo or Jekyll
	var hugoContentFolder = os.Args[2] + "/"

	var hugoContentType = os.Args[3]

	files, err := ioutil.ReadDir(postsHTMLFolder)
	if err != nil {
		panic(err)
	}

	err = os.MkdirAll(hugoContentFolder, os.ModePerm)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Found %d articles.\n", len(files))

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".html") || f.IsDir() {
			fmt.Printf("Ignoring (ext) %s\n", f.Name())
			continue
		}

		inpath := filepath.Join(postsHTMLFolder, f.Name())
		doc, err := read(inpath)
		if err != nil {
			fmt.Println("Error read html: ", err)
			continue
		}

		cleanupDoc(doc)
		post, err := process(doc, f, hugoContentFolder, hugoContentType)
		if err != nil {
			fmt.Println("Error process: ", err)
			os.RemoveAll(post.HddFolder)
			continue
		}

		if post.Draft == false && post.IsComment {
			fmt.Printf("Ignoring (comment) %s\n", f.Name())
			os.RemoveAll(post.HddFolder)
			continue
		}

		fmt.Printf("Processing %s => %s\n", f.Name(), post.HddFolder)

		outpath := post.HddFolder
		post.Body = docToMarkdown(doc)
		if len(post.Title) == 0 || len(post.Body) == 0 {
			fmt.Printf("Ignoring (empty) %s\n", f.Name())
			os.RemoveAll(post.HddFolder)
			continue
		}

		write(post, outpath)
	}
}

func nbsp(r rune) rune {
	if r == '\u00A0' {
		return ' '
	}
	return r
}

func process(doc *goquery.Document, f os.FileInfo, contentFolder, contentType string) (p post, err error) {
	defer func() {
		if mypanic := recover(); mypanic != nil {
			err = mypanic.(error)
		}
	}()

	p = post{}
	p.Lastmod = time.Now().Format(time.RFC3339)
	p.Title = strings.TrimSpace(doc.Find("title").Text())
  p.Title = strings.ReplaceAll(p.Title, "\"", "")
	p.Date = strings.TrimSpace(doc.Find("div.date").Text())
  t, _ := dateparse.ParseAny(p.Date)
  p.Date = t.Format("2006-01-02T15:04:05Z")
  p.Author = strings.TrimSpace(doc.Find("div.by-line").Text())
  p.Author = strings.ReplaceAll(p.Author, "by ", "")

	p.Subtitle = ""
	p.Description = ""
	p.IsComment = false
	p.Canonical = ""
  p.Draft = false
  p.Tags = []string{}

	// hugo/content/article_title/*
	slug := slug(p.Title)
	if len(slug) == 0 {
		slug = "noname_" + p.Date
	}
	pageBundle := slug
	p.HddFolder = fmt.Sprintf("%s%s/%s.md", contentFolder, contentType, pageBundle)
	os.RemoveAll(p.HddFolder) //make sure does not exists
	err = os.Mkdir(p.HddFolder, os.ModePerm)
	if err != nil {
		err = fmt.Errorf("error post folder: %s", err)
		return
	}
	p.Images, p.FeaturedImage, err = fetchAndReplaceImages(doc, p.HddFolder, contentType, pageBundle)

	if err != nil {
		err = fmt.Errorf("error images folder: %s", err)
	}

	//fallback, the featured image is the first one
	if len(p.FeaturedImage) == 0 && len(p.Images) > 0 {
		p.FeaturedImage = p.Images[0]
	}

	return
}

func docToMarkdown(doc *goquery.Document) string {
	body := ""
	doc.Find("div.message-body").Each(func(i int, s *goquery.Selection) {
		h, _ := s.Html()
		body += html2md.Convert(strings.TrimSpace(h))
	})
	body = strings.Map(nbsp, body)

	return strings.TrimSpace(body)
}

func cleanupDoc(doc *goquery.Document) {
	doc.Find("h1").Each(func(i int, selection *goquery.Selection) {
		selection.Remove()
	})
}

func read(path string) (*goquery.Document, error) {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// Load the HTML document
	return goquery.NewDocumentFromReader(f)
}

func write(post post, path string) {
	os.Remove(path)
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	err = tmpl.Execute(f, post)
	if err != nil {
		panic(err)
	}
}

var spaces = regexp.MustCompile(`[\s]+`)
var notallowed = regexp.MustCompile(`[^\p{L}\p{N}.\s]`)
var athe = regexp.MustCompile(`^(a\-|the\-)`)

func slug(s string) string {
	result := s
	result = strings.Replace(result, "%", " percent", -1)
	result = strings.Replace(result, "#", " sharp", -1)
	result = notallowed.ReplaceAllString(result, "")
	result = spaces.ReplaceAllString(result, "-")
	result = strings.ToLower(result)
	result = athe.ReplaceAllString(result, "")

	return result
}

var tmpl = template.Must(template.New("").Parse(`---
title: "{{ .Title }}"
author: "{{ .Author }}"
date: {{ .Date }}
lastmod: {{ .Lastmod }}
draft: false

---

{{ .Body }}
`))

func getTagsFor(url string) ([]string, error) {
	//TODO make a custom client with a small timeout!
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, err
	}
	var result []string
	//fmt.Printf("%s", doc.Text())
	doc.Find("ul.tags>li>a").Each(func(i int, selection *goquery.Selection) {
		result = append(result, selection.Text())
	})
	return result, nil
}

func fetchAndReplaceImages(doc *goquery.Document, folder, contentType, pageBundle string) ([]string, string, error) {
	images := doc.Find("img")
	if images.Length() == 0 {
		return nil, "", nil
	}

	diskImagesFolder := folder + "images/"
	err := os.Mkdir(diskImagesFolder, os.ModePerm)
	if err != nil {
		return nil, "", fmt.Errorf("error images folder: %s\n", err)
	}

	var index int
	var featuredImage string
	var result []string

	images.Each(func(i int, imgDomElement *goquery.Selection) {
		index++
		original, has := imgDomElement.Attr("src")
		if has == false {
			fmt.Print("warning img no src\n")
			return
		}

		pieces := strings.Split(original, ".")
		ext := pieces[len(pieces)-1]
		filename := fmt.Sprintf("%d.%s", index, ext)
		url := fmt.Sprintf("/%s/%s/images/%s", contentType, pageBundle, filename)
		diskPath := fmt.Sprintf("%s%s", diskImagesFolder, filename)

		err := DownloadFile(original, diskPath)
		if err != nil {
			fmt.Printf("error image: %s\n", err)
			return
		}
		//we presume that folder is the hugo/static/img folder
		imgDomElement.SetAttr("src", url)
		fmt.Printf("saved image %s => %s\n", original, diskPath)

		result = append(result, url)
		if _, isFeatured := imgDomElement.Attr("data-is-featured"); isFeatured {
			featuredImage = url
		}
	})
	return result, featuredImage, nil
}

// DownloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func DownloadFile(url, filepath string) error {
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
