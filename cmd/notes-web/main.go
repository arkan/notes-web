package main

import (
	"fmt"
	"os"

	"github.com/arkan/notes-web/internal/app"
)

func main() {
	if err := app.Main(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
