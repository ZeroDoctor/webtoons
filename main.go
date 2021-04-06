package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/geziyor/geziyor"
	ppt "github.com/zerodoctor/goprettyprinter"
)

// Info about comic
type Info struct {
	Title      string
	Subscriber string
	Rating     string
	Summary    string
	Creators   []string // i.e written by ann | art by frank
}

var (
	// maps <url, title> which are used for downloading and creating pdfs
	episodeMap = make(map[string]string)
	info       = make(chan Info, 1)
	wait       = make(chan bool, 1)
	comic      Info
	start      int
	end        int
)

var args struct {
	Title   string `arg:"required,-t,--title" help:"desire title number to download"`
	Start   int    `arg:"-s,--start" help:"episode number to start from" default:"1"`
	End     int    `arg:"-e,--end" help:"episode number to end on" default:"50000"` // TODO: change 50000 to something else
	Verbose bool   `arg:"-v,--verbose" help:"some extra logging"`
}

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

func main() {
	ppt.DisplayWarning = false
	ppt.SetInfoColor(ppt.Cyan)
	ppt.Decorator("", "|", "")
	ppt.LoggerFlags = ppt.FILE | ppt.LINE
	ppt.LoggerPrefix = func() string {
		return " [" + time.Now().Format("15:04:05") + "|" + ppt.WhereAmI() + "]: "
	}

	// parse arguments
	arg.MustParse(&args)
	title := args.Title
	start = args.Start
	end = args.End
	if args.Verbose {
		ppt.SetCurrentLevel(ppt.VerboseLevel)
	}

	// * note: if webtoons update there url schema, we would have to figure out this all over again
	list := "https://www.webtoons.com/en/GENRE/TITLE/list?title_no=" + title
	url := "http://www.webtoons.com/en/GENRE/TITLE/CHAPTER/viewer?title_no=" + title + "&episode_no=1"

	// parse comic infomation like title, author, genre, etc.
	geziyor.NewGeziyor(&geziyor.Options{
		StartURLs:   []string{list},
		ParseFunc:   parseInfo,
		LogDisabled: !args.Verbose,
	}).Start()

	comic = <-info
	createInit(comic)

	// parse comic episode list
	geziyor.NewGeziyor(&geziyor.Options{
		StartURLs:   []string{url},
		ParseFunc:   parseComic,
		LogDisabled: !args.Verbose,
	}).Start()

	// ensures that the code above executes first before continuing
	<-wait

	// parse episode to get a list of panels to create final pdf
	for url := range episodeMap {
		geziyor.NewGeziyor(&geziyor.Options{
			StartURLs:   []string{url},
			ParseFunc:   parseEpisode,
			LogDisabled: !args.Verbose,
		}).Start()
	}

	ppt.Infoln("Done!")
}
