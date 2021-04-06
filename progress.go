package main

import (
	"sync"
	"time"

	"github.com/vbauerster/mpb/v6"
	"github.com/vbauerster/mpb/v6/decor"
	ppt "github.com/zerodoctor/goprettyprinter"
)

var totalEp float64
var total int64 = 100

type Process struct {
	url  string
	name string
	task string
	f    func(Process, *mpb.Bar)
}

func initProgress() {
	if args.End == -1 {
		totalEp = 50
		return
	}
	totalEp = float64(args.End-args.Start) + 2.0
}

func addProgress(name string, task string, f func(*mpb.Bar)) {
	if args.Verbose {
		f(nil)
		return
	}

	p := mpb.New()
	bar := p.Add(total,
		// progress bar filler with customized style
		mpb.NewBarFiller(""),
		mpb.PrependDecorators(
			// display our name with one space on the right
			decor.Name(name, decor.WC{W: len(name) + 1, C: decor.DidentRight}),
			decor.Name(task, decor.WCSyncSpaceR),
			decor.CountersNoUnit("%d / %d", decor.WCSyncWidth),
			// replace ETA decorator with "done" message, OnComplete event
			decor.OnComplete(
				decor.AverageETA(decor.ET_STYLE_GO, decor.WC{W: 4}), " done",
			),
		),
		mpb.AppendDecorators(decor.Percentage()),
	)

	f(bar)

	p.Wait()
}

func addMultiProgress(mp []Process) {
	if args.Verbose {
		for _, p := range mp {
			p.f(p, nil)
		}
		return
	}

	var wg sync.WaitGroup
	currQueue := 0
	waitLimit := args.Workers
	if len(mp) < args.Workers {
		waitLimit = len(mp)
	}
	wg.Add(waitLimit)
	p := mpb.New(mpb.WithWaitGroup(&wg))

	for i, proc := range mp {
		bar := p.Add(total,
			// progress bar filler with customized style
			mpb.NewBarFiller(""),
			mpb.PrependDecorators(
				// display our name with one space on the right
				decor.Name(proc.name, decor.WC{W: len(proc.name) + 1, C: decor.DidentRight}),
				decor.Name(proc.task, decor.WCSyncSpaceR),
				decor.CountersNoUnit("%d / %d", decor.WCSyncWidth),
				// replace ETA decorator with "done" message, OnComplete event
				decor.OnComplete(
					decor.EwmaETA(decor.ET_STYLE_GO, 60), " done",
				),
			),
			mpb.AppendDecorators(decor.Percentage()),
		)

		go func(proc Process) {
			proc.f(proc, bar)
			wg.Done()
		}(proc)

		currQueue++
		if currQueue >= args.Workers && i != len(mp) {
			currQueue = 0
			ppt.Infof("currently: (%d/%d)\n", i+1, len(mp))

			waitLimit := args.Workers
			if len(mp)-(i+1) < args.Workers {
				waitLimit = (len(mp) - 1) - i
			}
			ppt.Infoln("next:", waitLimit)
			time.Sleep(time.Millisecond * 100)
			p.Wait()
			wg.Add(waitLimit)
			p = mpb.New(mpb.WithWaitGroup(&wg))
		}
	}

	ppt.Infof("currently: (%d/%d)\n", len(mp), len(mp))
	p.Wait()
}

func inc(bar *mpb.Bar, step int) {
	if bar != nil {
		bar.IncrBy(step)
	}
}
