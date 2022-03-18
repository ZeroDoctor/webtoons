package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

var ErrFolderNotFound error = errors.New("folder not found")

func GenerateFromFolder(path string) error {
	if !FolderExists(path) {
		return ErrFolderNotFound
	}

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}

	var errs []error
	for _, f := range files {
		if !FolderExists(path + "/" + f.Name()) {
			continue
		}

		err = generateChapterFromFolder(path + "/" + f.Name())
		if err != nil {
			errs = append(errs, err)
		}
	}

	if errs != nil {
		for _, e := range errs {
			err = fmt.Errorf("[%w]", e)
		}
	}

	return err
}

func generateChapterFromFolder(path string) error {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}

	var errs []error
	var panels []Panel
	for _, file := range files {
		if !IsImage(file.Name()) {
			continue
		}

		data, err := ioutil.ReadFile(path + "/" + file.Name())
		if err != nil {
			errs = append(errs, err)
			continue
		}

		panel, err := createPanel(data)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		panels = append(panels, panel)
	}

	if errs != nil {
		for _, e := range errs {
			err = fmt.Errorf("[%w]", e)
		}
	}

	return err
}

func IsImage(path string) bool {
	split := strings.Split(path, ".")
	if len(split) <= 1 {
		return false
	}

	ext := split[len(split)-1]
	switch ext {
	case "jpg", "jpeg", "png", "gif":
		return true
	}

	return false

}

// FolderExists checks if a folder exists
func FolderExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}

	return info.IsDir()
}
