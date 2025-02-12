// Package main (goris.go) :
// This file is included all commands and options.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Sterrenhemel/goris/ris"

	"github.com/urfave/cli"
)

const (
	appname = "goris"
)

// dispres : Display results
func dispres(r []string, c int, outputFile string) {
	results := []map[string]interface{}{}
	for _, url := range r {
		results = append(results, map[string]interface{}{
			"url": url,
		})
	}
	results = results[:c]
	jsonString, _ := json.Marshal(results)
	os.WriteFile(outputFile, jsonString, os.ModePerm)
}

// handler : Handler of goris
func handler(c *cli.Context) error {
	n := c.Int("number")
	outputFile := c.String("output")
	// offset := c.Int("offset")
	if n > 100 {
		n = 100
	}
	if len(c.String("fromurl")) == 0 && len(c.String("fromfile")) == 0 {
		return fmt.Errorf("no parameters. You can see help by '$ %s -h'", appname)
	}
	var results []string
	var err error
	if len(c.String("fromurl")) > 0 && len(c.String("fromfile")) == 0 {
		results, err = ris.DefImg(c.Bool("webpages")).ImgFromURL(c.String("fromurl"))
		if err != nil {
			return err
		}
	}
	if len(c.String("fromurl")) == 0 && len(c.String("fromfile")) > 0 {
		results, err = ris.DefImg(c.Bool("webpages")).ImgFromFile(c.String("fromfile"))
		if err != nil {
			return err
		}
	}
	if c.Bool("download") && (len(c.String("fromurl")) > 0 || len(c.String("fromfile")) > 0) && !c.Bool("webpages") {
		err := ris.Download(results, n)
		if err != nil {
			return err
		}
	}
	dispres(results, n, outputFile)
	return nil
}

// createHelp : Create help document.
func createHelp() *cli.App {
	a := cli.NewApp()
	a.Name = appname
	a.Authors = []cli.Author{
		{Name: "tanaike [ https://github.com/tanaikech/goris ] ", Email: "tanaike@hotmail.com"},
	}
	a.Usage = "Search for images with Google Reverse Image Search."
	a.Version = "3.0.2"

	a.Commands = []cli.Command{
		{
			Name:        "search",
			Aliases:     []string{"s"},
			Usage:       "[ " + appname + " s -u URL ] or [ " + appname + " s -f file ]",
			Description: "Do search images.",
			Action:      handler,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "fromurl, u",
					Usage: "Reverse Image Search from an URL.",
				},
				&cli.StringFlag{
					Name:  "fromfile, f",
					Usage: "Reverse Image Search from an image file.",
				},
				&cli.IntFlag{
					Name:  "number, n",
					Usage: "Number of retrieved image URLs. ( 1 - 100 )",
					Value: 50,
				},
				// &cli.IntFlag{
				// 	Name:  "offset, offset",
				// 	Usage: "Number of retrieved image URLs. ( 1 - 100 )",
				// 	Value: 0,
				// },
				&cli.StringFlag{
					Name:  "output, o",
					Usage: "output file",
				},
				&cli.BoolFlag{
					Name:  "download, d",
					Usage: "Download images from retrieved URLs.",
				},
				&cli.BoolFlag{
					Name:  "webpages, w",
					Usage: "This is boolean. Retrieve web pages with matching images on Google top page. When this is not used, images are retrieved.",
				},
			},
		},
	}
	return a
}

// main : Main of goris
func main() {
	a := createHelp()
	err := a.Run(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
