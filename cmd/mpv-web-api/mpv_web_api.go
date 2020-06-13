package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/sarpt/mpv-web-api/pkg/api"
)

const (
	address       = "localhost:3001"
	mpvSocketPath = "/tmp/mpvsocket"
)

func main() {
	cmd := exec.Command("mpv", "--idle", fmt.Sprintf("--input-ipc-server=%s", mpvSocketPath))
	err := cmd.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)

		return
	}

	server, err := api.NewServer([]string{}, mpvSocketPath)
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
