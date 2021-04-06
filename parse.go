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

	"github.com/PuerkitoBio/goquery"
	"github.com/geziyor/geziyor"
	"github.com/geziyor/geziyor/client"
	"github.com/jung-kurt/gofpdf"
	ppt "github.com/zerodoctor/goprettyprinter"
)

// the most volatile piece of code. if they make same changes to the front-end, everything breaks
func parseInfo(g *geziyor.Geziyor, r *client.Response) {
	var comic Info
	comic.Title = r.HTMLDoc.Find(".info").Find(".subj").Text()
	comic.Subscriber = r.HTMLDoc.Find(".grade_area").Find("span.ico_subscribe + em").Text()
	comic.Rating = r.HTMLDoc.Find("#_starScoreAverage").Text()
	comic.Summary = r.HTMLDoc.Find("#_asideDetail > p.summary").Text()

	var prefixes []string
	var creators []string
	r.HTMLDoc.Find("div._authorInfoLayer div._authorInnerContent").Find("p.by").Each(
		func(_ int, s *goquery.Selection) {
			prefixes = append(prefixes, s.Text())
		},
	)
	r.HTMLDoc.Find("div._authorInfoLayer div._authorInnerContent").Find("h3.title").Each(
		func(_ int, s *goquery.Selection) {
			creators = append(creators, s.Text())
		},
	)
	for i := range prefixes {
		comic.Creators = append(comic.Creators, prefixes[i]+": "+creators[i])
	}
	info <- comic
}

// finds out what episode to queue for downloading
func parseComic(g *geziyor.Geziyor, r *client.Response) {
	r.HTMLDoc.Find("#topEpisodeList").Find("div.episode_cont").Find("li").Each(
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

	wait <- false
}

// converts the response body to image bytes
func readImage(resp *http.Response) ([]byte, error) {
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
		// desired comic size is 800x1280 pixels which convert to "inches" is 8.33x13.33
		Size: gofpdf.SizeType{Wd: 8.33, Ht: 13.33},
	})
	// remove pdf header
	pdf.SetTopMargin(0.0)
	pdf.SetHeaderFuncMode(func() {}, false)
	pwidth, pheight := pdf.GetPageSize()

	for i, p := range pages {
		// get image
		pdf.RegisterImageReader(title+strconv.Itoa(i), imgType[i], bytes.NewBuffer(p))
		if pdf.Ok() {
			options := gofpdf.ImageOptions{
				ReadDpi:   false,
				ImageType: imgType[i],
			}
			// add image to page
			pdf.ImageOptions(title+strconv.Itoa(i), 0, pdf.GetY(), pwidth, pheight, true, options, 0, "")
		}

	}

	err := pdf.OutputFileAndClose("./" + comic.Title + "/" + title + ".pdf")
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

	var imgType []string
	var panels [][]byte
	r.HTMLDoc.Find("#_imageList").Find("img").Each(
		func(counter int, s *goquery.Selection) {
			// find panel image url
			href, ok := s.Attr("data-url")
			if ok {
				url, err := url.Parse(r.JoinURL(href))
				if err != nil {
					ppt.Errorln("failed to parse url:", err.Error())
					return
				}

				// create get request with important header
				req := &http.Request{
					Method: "GET",
					Header: http.Header(map[string][]string{
						// * note: super important header. if changed, thing will become a lot harder
						"Referer": {"http://www.webtoons.com"},
					}),
					URL: url,
				}

				// send request
				resp, err := g.Client.Do(req)
				if err != nil {
					ppt.Errorln("failed request:", err.Error())
				}

				// handle response
				panel, err := readImage(resp)
				if err != nil {
					return
				}

				imgType = append(imgType, "png")
				if strings.Contains(href, "jpg") {
					imgType[len(imgType)-1] = "jpg"
				}

				panels = append(panels, panel)
			}
		},
	)

	// create episode pdf
	title := episodeMap[g.Opt.StartURLs[0]]
	createPDF(title, panels, imgType)
}
