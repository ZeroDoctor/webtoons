# Webtoons

## Usage

```txt
Usage: webtoons --title TITLE --start START [--end END] [--verbose]

Options:
    --title TITLE, -t TITLE desire title number to download

    --start START, -s START episode number to start from

    --end END, -e END       episode number to end on [default: 50000]

    --verbose, -v           some extra logging

    --help, -h              display this help and exit
```

---

## Exmaple

webtoons.exe -t=1099 -s=1 -e=5

^^^ This will download gosus' episodes 1 through 5

## TODO

* add progress bar
* remove hard value on end variable
* add more options?
* correct the size of each panel
* menu of comics when no arguments given
