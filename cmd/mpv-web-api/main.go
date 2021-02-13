package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sarpt/goutils/pkg/listflag"

	"github.com/sarpt/mpv-web-api/pkg/api"
)

const (
	defaultAddress = "localhost:3001"
	mpvSocketPath  = "/tmp/mpvsocket"

	dirFlag       = "dir"
	allowCorsFlag = "allow-cors"
	addrFlag      = "addr"
)

var (
	dir       *listflag.StringList
	allowCORS *bool
	address   *string
)

func init() {
	dir = listflag.NewStringList([]string{})

	flag.Var(dir, dirFlag, "directory containing movies. when left empty, current working directory will be used")
	allowCORS = flag.Bool(allowCorsFlag, false, "when not provided, Cross Origin Site Requests will be rejected")
	address = flag.String(addrFlag, defaultAddress, "address on which server should listen on. default is localhost:3001")

	flag.Parse()
}

func main() {
	cfg := api.Config{
		MpvSocketPath: mpvSocketPath,
		Address:       *address,
		AllowCORS:     *allowCORS,
	}
	server, err := api.NewServer(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)

		return
	}
	defer server.Close()

	var moviesDirectories []string
	if len(dir.Values()) == 0 {
		wd, err := os.Getwd()
		if err == nil {
			moviesDirectories = append(moviesDirectories, fmt.Sprintf("%s/", wd))
		}
	} else {
		for _, dir := range dir.Values() {
			moviesDirectories = append(moviesDirectories, fmt.Sprintf("%s/", dir))
		}
	}

	fmt.Fprintf(os.Stdout, "directories being watched for movie files:\n%s\n", strings.Join(moviesDirectories, "\n"))
	server.AddDirectories(moviesDirectories)

	err = server.Serve()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)

		return
	}
}
