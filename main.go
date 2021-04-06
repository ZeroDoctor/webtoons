package main

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/alexflint/go-arg"
	"github.com/geziyor/geziyor"
	"github.com/geziyor/geziyor/client"
	"github.com/jung-kurt/gofpdf"
	ppt "github.com/zerodoctor/goprettyprinter"
)

var episodeMap = make(map[string]string)
var start int
var end int

var args struct {
	Title   string `arg:"required,-t,--title" help:"desire title number to download"`
	Start   int    `arg:"required,-s,--start" help:"episode number to start from"`
	End     int    `arg:"-e,--end" help:"episode number to end on" default:"50000"` // TODO: change 50000 to something else
	Verbose bool   `arg:"-v,--verbose" help:"some extra logging"`
}

func parseComic(g *geziyor.Geziyor, r *client.Response) {
	r.HTMLDoc.Find("#topEpisodeList").Find(".episode_cont").Find("li").Each(
		func(_ int, s *goquery.Selection) {
			episodeNumber, _ := s.Attr("data-episode-no")
			num, err := strconv.Atoi(episodeNumber)
			if err != nil || num > end || num < start {
				ppt.Verboseln("Skipped episode:", num)
				return
			}
			next, _ := s.Find("a").Attr("href")
			title, _ := s.Find("img").Attr("alt")
			episodeMap[next] = title
		},
	)
}

func handleResp(resp *http.Response) ([]byte, error) {
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		errStr := ppt.Errorln("failed to read body:", err.Error())
		return nil, errors.New(errStr)
	}

	return data, nil
}

func createPDF(title string, pages [][]byte, imgType []string) {
	pdf := gofpdf.NewCustom(&gofpdf.InitType{
		OrientationStr: "P",
		UnitStr:        "in",
		Size:           gofpdf.SizeType{Wd: 8.33, Ht: 13.33},
	})
	pdf.SetTopMargin(0.0)
	pdf.SetHeaderFuncMode(func() {}, false)
	pwidth, pheight := pdf.GetPageSize()
	cheight := 0.0

	for i, p := range pages {
		pdf.RegisterImageReader(title+strconv.Itoa(i), imgType[i], bytes.NewBuffer(p))
		if pdf.Ok() {
			options := gofpdf.ImageOptions{
				ReadDpi:   false,
				ImageType: imgType[i],
			}
			pdf.ImageOptions(title+strconv.Itoa(i), 0, pdf.GetY(), pwidth, pheight, true, options, 0, "")

			cheight += pheight
			if cheight > pheight {
				cheight = 0.0
			}
		}

	}

	err := pdf.OutputFileAndClose("./" + title + ".pdf")
	if err != nil {
		ppt.Errorln("failed to create pdf:", err.Error())
		os.Exit(1)
	}
}

func parseEpisode(g *geziyor.Geziyor, r *client.Response) {
	defer func() {
		if r := recover(); r != nil {
			ppt.Errorln(r)
			os.Exit(1)
		}
	}()

	title := episodeMap[g.Opt.StartURLs[0]]
	var imgType []string
	var pages [][]byte
	r.HTMLDoc.Find("#_imageList").Find("img").Each(
		func(counter int, s *goquery.Selection) {
			href, ok := s.Attr("data-url")
			if ok {
				url, err := url.Parse(r.JoinURL(href))
				if err != nil {
					ppt.Errorln("failed to parse url:", err.Error())
					return
				}

				req := &http.Request{
					Method: "GET",
					Header: http.Header(map[string][]string{
						"Referer": {"http://www.webtoons.com"},
					}),
					URL: url,
				}

				resp, err := g.Client.Do(req)
				if err != nil {
					ppt.Errorln("failed request:", err.Error())
				}

				data, err := handleResp(resp)
				if err != nil {
					return
				}

				imgType = append(imgType, "png")
				if strings.Contains(href, "jpg") {
					imgType[len(imgType)-1] = "jpg"
				}

				pages = append(pages, data)
			}
		},
	)

	createPDF(title, pages, imgType)
}

func main() {
	ppt.DisplayWarning = false
	ppt.SetInfoColor(ppt.Cyan)
	ppt.Decorator("", "|", "")
	ppt.LoggerFlags = ppt.FILE | ppt.LINE
	ppt.LoggerPrefix = func() string {
		return " [" + time.Now().Format("15:04:05") + "|" + ppt.WhereAmI() + "]: "
	}

	ppt.Infoln("Start program")

	arg.MustParse(&args)
	title := args.Title
	start = args.Start
	end = args.End
	if args.Verbose {
		ppt.Infoln("set to verbose")
		ppt.SetCurrentLevel(ppt.VerboseLevel)
	}

	url := "http://www.webtoons.com/en/fantasy/gosu/chapter-1/viewer?title_no=" + title + "&episode_no=1"

	geziyor.NewGeziyor(&geziyor.Options{
		StartURLs: []string{url},
		ParseFunc: parseComic,
	}).Start()

	for url := range episodeMap {
		geziyor.NewGeziyor(&geziyor.Options{
			StartURLs: []string{url},
			ParseFunc: parseEpisode,
		}).Start()
	}
}
