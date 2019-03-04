package pkg

import (
	"strings"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}
func Convert(file string) {
	if file == "" {
		return
	}

	var subFile SubFile
	err := subFile.Open(file)
	check(err)

	err = subFile.Run()
	check(err)

	err = subFile.Save(strings.Replace(file, ".sup", ".webvtt", -1))
	check(err)
}
