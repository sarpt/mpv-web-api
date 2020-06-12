package main

import (
	"fmt"
	"os"

	"github.com/sarpt/mpv-web-api/pkg/api"
)

func main() {
	err := api.Serve()
	if err != nil {
		fmt.Fprintf(os.Stdout, "%s\n", err)
	}
}
