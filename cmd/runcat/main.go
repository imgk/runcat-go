// RunCat in Golang

package main

import (
	"log"

	"github.com/imgk/runcat-go/cmd"
)

func main() {
	if err := cmd.Run(); err != nil {
		log.Panic(err)
	}
}
