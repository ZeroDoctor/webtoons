package main

import (
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	ppt "github.com/zerodoctor/goprettyprinter"
)

// Info about comic
type Info struct {
	Title      string
	Subscriber string
	Rating     string
	Summary    string
	Creators   []string // i.e written by ann | art by frank
	End        int
	Episode    string
}

var (
	// maps <url, title> which are used for downloading and creating pdfs
	episodeMap = make(map[string]string)
	info       = make(chan Info, 1)
	wait       = make(chan bool, 1)
	comic      Info

	replacer *strings.Replacer = strings.NewReplacer(
		":", "_",
		"<", "[",
		">", "]",
		"|", "-",
		"\"", "",
		"/", ".",
		"\\", ".",
		"?", "",
		"*", "",
	)
)

var args struct {
	TitleNum string `arg:"positional" help:"desire title number to download"`
	Genre    string `arg:"-g,--genre" help:"genre specified in url" default:"GENRE"`
	Title    string `arg:"-t,--title" help:"title specified in url" default:"TITLE"`
	Start    int    `arg:"-s,--start" help:"episode number to start from" default:"1"`
	End      int    `arg:"-e,--end" help:"episode number to end on" default:"-1"`
	Workers  int    `arg:"-w,--workers" help:"number of files to download at the same time" default:"10"`
	Verbose  bool   `arg:"-v,--verbose" help:"some extra logging"`
}

func cleanString(str string) string {

	var b strings.Builder
	for _, r := range str {
		if r >= 32 && r <= 126 {
			b.WriteRune(r)
		}
	}

	str = replacer.Replace(b.String())

	return str
}

func addLog(msg string) {
	err := ioutil.WriteFile("./"+comic.Title+"/log.txt", []byte(msg), 0755)
	if err != nil {
		ppt.Errorln("failed to write to log file:", err.Error())
	}
}

func main() {
	// logging settings
	ppt.DisplayWarning = false
	ppt.Init()
	ppt.SetInfoColor(ppt.Cyan)
	ppt.Decorator("", "|", "")
	ppt.LoggerFlags = ppt.FILE | ppt.LINE
	ppt.LoggerPrefix = func() string {
		return " [" + time.Now().Format("15:04:05") + "|" + ppt.WhereAmI() + "]: "
	}

	// parse arguments
	arg.MustParse(&args)

	if args.End != -1 && args.End < args.Start {
		ppt.Errorln("ending number can not be smaller than starting number")
		os.Exit(1)
	}

	if args.Verbose {
		ppt.SetCurrentLevel(ppt.VerboseLevel)
		parse()
		return // exit program
	}

	initProgress()
	parse()
	ppt.Infoln("Done!")
}
