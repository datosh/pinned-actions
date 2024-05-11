package main

import (
	"os"

	"github.com/go-git/go-git/v5"
)

func main() {
	_, err := git.PlainClone("linux", true, &git.CloneOptions{
		URL:      "https://github.com/torvalds/linux.git",
		Progress: os.Stdout,
		Depth:    1,
	})
	if err != nil {
		panic(err)
	}
}
