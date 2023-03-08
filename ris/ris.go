// Package ris (ris.go) :
// These methods are for retrieving images from url and file.
package ris

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	baseurl = "https://www.google.com"
)

// requestParams : Parameters for fetchURL
type requestParams struct {
	Method      string
	URL         string
	Contenttype string
	Data        io.Reader
	Client      *http.Client
}

// Imgdata : Image URL
type Imgdata struct {
	OU      string `json:"ou"`
	WebPage bool
}

// DefImg : Initialize imagdata.
func DefImg(webpages bool) *Imgdata {
	return &Imgdata{
		WebPage: webpages,
	}
}

// fetchURL : Fetch method
func (r *requestParams) fetchURL() (*http.Response, error) {
	req, err := http.NewRequest(
		r.Method,
		r.URL,
		r.Data,
	)
	if err != nil {
		return nil, err
	}
	if len(r.Contenttype) > 0 {
		req.Header.Set("Content-Type", r.Contenttype)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.102 Safari/537.36")
	res, _ := r.Client.Do(req)
	return res, nil
}

// getURLs : Retrieve URLs.
func (r *requestParams) getURLs(res *http.Response, imWebPage bool) ([]string, error) {
	var url string
	var chk bool
	var ar []string
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}
	doc.Find("g-section-with-header").Each(func(_ int, s *goquery.Selection) {
		url, chk = s.Find("div").Find("title-with-lhs-icon").Find("a").Attr("href")
		if !chk {
			fmt.Fprint(os.Stderr, "Error: Base URL cannot be retrieved. The specification of Google side might be changed.\n")
			os.Exit(1)
		}
	})
	r.URL = baseurl + url
	r.Client = &http.Client{Timeout: time.Duration(10) * time.Second}
	res, err = r.fetchURL()
	if err != nil {
		return nil, err
	}
	doc, err = goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}
	reg1 := regexp.MustCompile("key: 'ds:1'")
	reg2 := regexp.MustCompile(`"(https?:\/\/.+?)",\d+,\d+`)
	reg3 := regexp.MustCompile(`https:\/\/encrypted\-tbn0`)
	reg4 := regexp.MustCompile(`"(https?:\/\/.+?)"`)
	reg5 := regexp.MustCompile(`https?:\/\/.+?\.jpg|https?:\/\/.+?\.png|https?:\/\/.+?\.gif`)
	doc.Find("script").Each(func(_ int, s *goquery.Selection) {
		if reg1.MatchString(s.Text()) {
			var urls [][]string
			if imWebPage {
				urls = reg4.FindAllStringSubmatch(s.Text(), -1)
				for _, u := range urls {
					if !reg3.MatchString(u[1]) && !reg5.MatchString(u[0]) {
						ss, err := strconv.Unquote(`"` + u[1] + `"`)
						if err == nil {
							ar = append(ar, ss)
						}
					}
				}
			} else {
				urls = reg2.FindAllStringSubmatch(s.Text(), -1)
				for _, u := range urls {
					if !reg3.MatchString(u[1]) {
						ss, err := strconv.Unquote(`"` + u[1] + `"`)
						if err == nil {
							ar = append(ar, ss)
						}
					}
				}
			}
		}
	})
	if len(ar) == 0 {
		return nil, errors.New("data couldn't be retrieved")
	}
	return ar, nil
}

func downloadFile(filepath string, url string) (err error) {

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (iPad; CPU OS 12_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	// Get the data
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

// ImgFromURL : Search images from an image URL
func (im *Imgdata) ImgFromURL(searchimage string) ([]string, error) {
	fileName := "/tmp/" + RandStringRunes(10)
	fullName := fileName + "." + "jpg"
	downloadFile(fullName, searchimage)
	return im.ImgFromFile(fullName)
}

// ImgFromFile : Search images from an image file
func (im *Imgdata) ImgFromFile(file string) ([]string, error) {
	var err error
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fs, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer fs.Close()
	data, err := w.CreateFormFile("encoded_image", file)
	if err != nil {
		return nil, err
	}
	if _, err = io.Copy(data, fs); err != nil {
		return nil, err
	}
	w.Close()
	r := &requestParams{
		Method: "POST",
		URL:    baseurl + "/searchbyimage/upload",
		Data:   &b,
		Client: &http.Client{
			Timeout:       time.Duration(10) * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error { return errors.New("Redirect") },
		},
		Contenttype: w.FormDataContentType(),
	}
	var res *http.Response
	for {
		res, err = r.fetchURL()
		if err != nil {
			return nil, err
		}
		if res.StatusCode == 200 {
			break
		}
		reurl, _ := res.Location()
		r.URL = reurl.String()
		r.Method = "GET"
		r.Data = nil
		r.Contenttype = ""
	}
	ar, err := r.getURLs(res, im.WebPage)
	if err != nil {
		return nil, err
	}
	return ar, nil
}

// Download : Download image files from searched image URLs
func Download(r []string, c int) error {
	var wg sync.WaitGroup
	dlch := make(chan string, len(r))
	workers := 2
	reg := regexp.MustCompile(`\?.+`)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(wg *sync.WaitGroup, dlch chan string) {
			defer wg.Done()
			var res *http.Response
			var err error
			for {
				dlurl, fin := <-dlch
				if !fin {
					return
				}
				fname := reg.ReplaceAllString(dlurl, "")
				filename := filepath.Base(fname)
				conv := strings.Replace(strings.TrimSpace(fname), filename, "", -1)
				conv = strings.Replace(strings.TrimSpace(conv), "http://", "", -1)
				conv = strings.Replace(strings.TrimSpace(conv), "https://", "", -1)
				conv = strings.Replace(strings.TrimSpace(conv), "/", "_", -1)
				conv = strings.Replace(strings.TrimSpace(conv), ".", "-", -1)
				conv += filename
				r := &requestParams{
					Method: "GET",
					URL:    dlurl,
					Data:   nil,
					Client: &http.Client{Timeout: time.Duration(100) * time.Second},
				}
				res, err = r.fetchURL()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v. ", err)
					os.Exit(1)
				}
				body, err := ioutil.ReadAll(res.Body)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v. ", err)
					os.Exit(1)
				}
				ioutil.WriteFile(conv, body, 0777)
				defer res.Body.Close()
			}
		}(&wg, dlch)
	}
	for i := 0; i < c; i++ {
		dlch <- r[i]
	}
	close(dlch)
	wg.Wait()
	return nil
}
