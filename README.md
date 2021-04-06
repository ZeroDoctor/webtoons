# Webtoons

## Usage

```txt
Usage: webtoons --title TITLE --start START [--end END] [--workers WORKERS] [--verbose]

Options:
    --title, -t TITLE     desire title number to download

    --start, -s START     episode number to start from

    --workers, -w WORKERS number of files to download async [default: 5]

    --end, -e END         episode number to end on [default: 50000]

    --verbose, -v         some extra logging

    --help, -h            display this help and exit
```

---

## Exmaple

webtoons.exe -t=1099 -s=1 -e=5

^^^ This will download gosus' episodes 1 through 5

## TODO

* correct the size of each panel
* menu of comics when no arguments given (long term goal)
