package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/sarpt/mpv-web-api/pkg/api"
)

const (
	address       = "localhost:3001"
	mpvSocketPath = "/tmp/mpvsocket"

	mpvName           = "mpv"
	idleArg           = "--idle"
	inputIpcServerArg = "--input-ipc-server"
)

var (
	dirFlag *string
)

func init() {
	dirFlag = flag.String("dir", "", "directory containing movies. when left empty, current working directory will be used")
	flag.Parse()
}

func main() {
	cmd := exec.Command(mpvName, idleArg, fmt.Sprintf("%s=%s", inputIpcServerArg, mpvSocketPath))
	err := cmd.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)

		return
	}

	var moviesDirectories []string

	if *dirFlag == "" {
		wd, err := os.Getwd()
		if err == nil {
			moviesDirectories = append(moviesDirectories, fmt.Sprintf("%s/", wd))
		}
	} else {
		moviesDirectories = append(moviesDirectories, fmt.Sprintf("%s/", *dirFlag)) // TODO: multiple dir arguments handling for multiple directories
	}

	fmt.Fprintf(os.Stdout, "Directories being watched for movie files:\n%s\n", strings.Join(moviesDirectories, "\n"))

	server, err := api.NewServer(moviesDirectories, mpvSocketPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)

		return
	}
	defer server.Close()

	err = server.Serve()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)

		return
	}
}
