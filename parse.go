package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/geziyor/geziyor"
	"github.com/geziyor/geziyor/client"
	"github.com/jung-kurt/gofpdf"
	"github.com/vbauerster/mpb/v6"
	ppt "github.com/zerodoctor/goprettyprinter"
)

// create folder and about file from comic info
func createInit(comic Info) {
	if _, err := os.Stat("./" + comic.Title); os.IsNotExist(err) {
		err := os.Mkdir("./"+comic.Title, 0755)
		if err != nil {
			ppt.Errorln("failed to create folder:", err.Error())
			os.Exit(1)
		}
	}

	if _, err := os.Stat("./" + comic.Title + "/about.json"); os.IsNotExist(err) {
		data, err := json.MarshalIndent(comic, "", "  ")
		if err != nil {
			ppt.Errorln("failed to marshal comic info:", err.Error())
			os.Exit(1)
		}

		err = ioutil.WriteFile("./"+comic.Title+"/about.json", data, 0755)
		if err != nil {
			ppt.Errorln("failed to write to file:", err.Error())
			os.Exit(1)
		}
	}
}

func parse() {
	// * note: if webtoons update there url schema, we would have to figure out this all over again
	list := "https://www.webtoons.com/en/GENRE/TITLE/list?title_no=" + args.Title
	url := "http://www.webtoons.com/en/GENRE/TITLE/CHAPTER/viewer?title_no=" + args.Title + "&episode_no=1"

	// parse comic infomation like title, author, genre, etc.
	ppt.Infoln("fetching comic from book store:", list)
	addProgress("comic info:", "parsing...", func(bar *mpb.Bar) {
		parseInfo(list, bar)
	})

	comic = <-info
	createInit(comic)

	// parse comic episode list
	addProgress("episode list:", "fetching...", func(bar *mpb.Bar) {
		parseComic(url, bar)
	})

	// ensures that the code above executes first before continuing
	<-wait

	// parse episode to get a list of panels to create final pdf
	var procs []Process
	for urlStr := range episodeMap {
		p := Process{
			url:  urlStr,
			name: episodeMap[urlStr],
			task: "downloading...",
			f: func(p Process, bar *mpb.Bar) {
				parseEpisode(p.url, bar)
			},
		}

		procs = append(procs, p)
	}

	addMultiProgress(procs)
}

func parseInfo(list string, bar *mpb.Bar) {
	if bar != nil {
		bar.SetTotal(4, false)
	}
	geziyor.NewGeziyor(&geziyor.Options{
		StartURLs: []string{list},
		// the most volatile piece of code. if they make same changes to the front-end, everything breaks
		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
			if args.Verbose {
				ppt.Infoln("parsing comic info...")
			}
			var comic Info
			var err error
			comic.Title = r.HTMLDoc.Find(".info").Find(".subj").Text()
			comic.Subscriber = r.HTMLDoc.Find(".grade_area").Find("span.ico_subscribe + em").Text()
			comic.Rating = r.HTMLDoc.Find("#_starScoreAverage").Text()
			comic.Summary = r.HTMLDoc.Find("#_asideDetail > p.summary").Text()
			end, _ := r.HTMLDoc.Find("#_listUl").Find("li").Attr("data-episode-no")
			comic.End, err = strconv.Atoi(end)
			if err != nil {
				ppt.Errorln("failed to parse latest episode number:", err.Error())
			}

			inc(bar, 1)
			var prefixes []string
			var creators []string
			r.HTMLDoc.Find("div._authorInfoLayer div._authorInnerContent").Find("p.by").Each(
				func(_ int, s *goquery.Selection) {
					prefixes = append(prefixes, s.Text())
				},
			)

			inc(bar, 1)
			r.HTMLDoc.Find("div._authorInfoLayer div._authorInnerContent").Find("h3.title").Each(
				func(_ int, s *goquery.Selection) {
					creators = append(creators, s.Text())
				},
			)

			inc(bar, 1)
			for i := range prefixes {
				comic.Creators = append(comic.Creators, prefixes[i]+": "+creators[i])
			}

			inc(bar, 1)
			info <- comic
			if bar != nil {
				bar.SetTotal(4, true)
			}
		},
		LogDisabled: !args.Verbose,
	}).Start()
}

// finds out what episode to queue for downloading
func parseComic(urlStr string, bar *mpb.Bar) {
	if args.End <= -1 {
		totalEp = float64(comic.End)
		args.End = comic.End
	}

	if bar != nil {
		bar.SetTotal(int64(totalEp), false)
	}
	geziyor.NewGeziyor(&geziyor.Options{
		StartURLs: []string{urlStr},
		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
			if args.Verbose {
				ppt.Infoln("parsing episode list...")
			}
			r.HTMLDoc.Find("#topEpisodeList").Find("div.episode_cont").Find("li").Each(
				func(i int, s *goquery.Selection) {
					episodeNumber, _ := s.Attr("data-episode-no")
					num, err := strconv.Atoi(episodeNumber)
					if err != nil {
						ppt.Errorln("failed to parse episode number:", err.Error())
						os.Exit(1)
					}
					if args.End != -1 && (num > args.End || num < args.Start) {
						ppt.Verboseln("Skipped episode:", num)
						return
					}
					next, _ := s.Find("a").Attr("href")
					title, _ := s.Find("img").Attr("alt")
					episodeMap[next] = fmt.Sprintf("[%d]", num) + title

					inc(bar, 1)
					if bar != nil && args.End == -1 && bar.Current() > int64(totalEp)-10 {
						totalEp += 20
						bar.SetTotal(int64(totalEp), false)
					}
				},
			)

			wait <- false
			if bar != nil {
				bar.SetTotal(int64(totalEp), true)
			}
		},
		LogDisabled: !args.Verbose,
	}).Start()
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

func createPDF(title string, pages [][]byte, imgType []string, bar *mpb.Bar) {
	if args.Verbose {
		ppt.Infoln("creating " + title + ".pdf...")
	}
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
		inc(bar, 1)
	}

	// replace windows
	replacer := strings.NewReplacer(
		":", "_",
		"<", "_",
		">", "_",
		" ", "_",
		"*", "_",
		"|", "_",
	)
	title = replacer.Replace(title)
	err := pdf.OutputFileAndClose("./" + comic.Title + "/" + title + ".pdf")
	if err != nil {
		ppt.Errorln("failed to create pdf:", err.Error())
		os.Exit(1)
	}

	inc(bar, 1)
}

func parseEpisode(urlStr string, bar *mpb.Bar) {
	start := time.Now()
	geziyor.NewGeziyor(&geziyor.Options{
		StartURLs: []string{urlStr},
		ParseFunc: func(g *geziyor.Geziyor, r *client.Response) {
			defer func() {
				if r := recover(); r != nil {
					ppt.Errorln("recovered from painc:", r)
					os.Exit(1)
				}
			}()
			if args.Verbose {
				ppt.Infoln("parsing episode panels...")
			}

			var imgType []string
			var panels [][]byte
			totalPanels := len(r.HTMLDoc.Find("#_imageList").Find("img").Nodes)
			if bar != nil {
				bar.SetTotal(int64(float64(totalPanels)*1.5), false)
			}

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

						imageType := resp.Header["Content-Type"][0][len("image/"):]
						imgType = append(imgType, imageType)

						panels = append(panels, panel)

						inc(bar, 1)
					}
				},
			)

			if bar != nil {
				total := len(panels) + totalPanels + 1
				bar.SetTotal(int64(total), false)
			}

			// create episode pdf
			title := episodeMap[g.Opt.StartURLs[0]]
			createPDF(title, panels, imgType, bar)
		},
		LogDisabled: !args.Verbose,
	}).Start()

	if bar != nil {
		bar.SetTotal(int64(total), true)
		bar.DecoratorEwmaUpdate(time.Since(start))
	}
}
