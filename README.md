# Webtoons

## Usage

```txt
Usage: webtoons.exe [--genre GENRE] [--title TITLE] [--start START] [--end END] [--workers WORKERS] [--verbose] TITLENUM

Positional arguments:
  TITLENUM                desire title number to download

Options:
    --genre, -g GENRE     genre specified in url [default: GENRE]

    --title, -t TITLE     title specified in url [default: TITLE]

    --start, -s START     episode number to start from [default: 1]

    --end, -e END         episode number to end on [default: -1]

    --workers, -w WORKERS number of files to download at the same time [default: 10]

    --verbose, -v         some extra logging

    --help, -h            display this help and exit
```

---

## Exmaple

webtoons.exe 1099 -s=1 -e=5

^^^ This will download gosus' episodes 1 through 5

## TODO

* handle errors gracefully
* cancel download gracefully
* add subscription to urls
* add bookmark support (should work with subscriptions)
* allow user to format titles as they come

## GOALS

* output in other format instead of only pdf (mid term goal)
* menu of comics when no arguments given (long term goal)

## Released Using Command

go build -ldflags="-s -w" . && upx --lzma webtoons.exe
