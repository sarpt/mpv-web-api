package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/sarpt/goutils/pkg/listflag"

	"github.com/sarpt/mpv-web-api/pkg/api"
	"github.com/sarpt/mpv-web-api/pkg/state"
)

const (
	logPrefix = "cmd.main#"
)

const (
	defaultAddress          = "localhost:3001"
	defaultMpvSocketPath    = "/tmp/mpvsocket"
	defaultSocketTimeoutSec = 15

	addressFlag          = "addr"
	allowCorsFlag        = "allow-cors"
	dirFlag              = "dir"
	mpvSocketPathFlag    = "mpv-socket-path"
	socketTimeoutSecFlag = "socket-timeout"
	startMpvInstanceFlag = "start-mpv-instance"
	watchDirFlag         = "watch-dir"
)

var (
	address          *string
	allowCORS        *bool
	dir              *listflag.StringList
	watchDir         *listflag.StringList
	mpvSocketPath    *string
	socketTimeoutSec *int64
	startMpvInstance *bool
)

func init() {
	dir = listflag.NewStringList([]string{})
	watchDir = listflag.NewStringList([]string{})

	flag.Var(dir, dirFlag, "directory containing media files - the directory is not watched for changes")
	address = flag.String(addressFlag, defaultAddress, "address on which server should listen on")
	allowCORS = flag.Bool(allowCorsFlag, false, "when not provided, Cross Origin Site Requests will be rejected")
	mpvSocketPath = flag.String(mpvSocketPathFlag, defaultMpvSocketPath, "path to a socket file used by a MPV instance to listen for commands")
	socketTimeoutSec = flag.Int64(socketTimeoutSecFlag, defaultSocketTimeoutSec, "maximum allowed time in seconds for retrying connection to MPV instance")
	startMpvInstance = flag.Bool(startMpvInstanceFlag, true, "controls whether the application should create and manage its own MPV instance")
	flag.Var(watchDir, watchDirFlag, "directory containing media files - the directory will be watched for changes. When left empty and no --dir arguments are specified, current working directory will be used")

	flag.Parse()
}

func main() {
	errLog := log.New(os.Stderr, logPrefix, log.LstdFlags)
	outLog := log.New(os.Stdout, logPrefix, log.LstdFlags)

	socketConnectionTimeout := time.Duration(time.Duration(*socketTimeoutSec) * time.Second)
	cfg := api.Config{
		MpvSocketPath:           *mpvSocketPath,
		Address:                 *address,
		AllowCORS:               *allowCORS,
		StartMpvInstance:        *startMpvInstance,
		SocketConnectionTimeout: socketConnectionTimeout,
	}
	server, err := api.NewServer(cfg)
	if err != nil {
		errLog.Printf("could not start API server due to an error: %s\n", err)

		return
	}

	var mediaFilesDirectories []state.Directory
	if len(dir.Values()) == 0 && len(watchDir.Values()) == 0 {
		wd, err := os.Getwd()
		if err != nil {
			errLog.Printf("could not start API server due to error while getting working directory: %s\n", err)

			return
		}

		outLog.Printf("No directories specified for server - watching working directory '%s'\n", wd)
		mediaFilesDirectories = append(mediaFilesDirectories, state.Directory{
			Path:    state.EnsureDirectoryPath(wd),
			Watched: true,
		})
	} else {
		for _, dir := range watchDir.Values() {
			mediaFilesDirectories = append(mediaFilesDirectories, state.Directory{
				Path:    state.EnsureDirectoryPath(dir),
				Watched: true,
			})
		}

		for _, dir := range dir.Values() {
			mediaFilesDirectories = append(mediaFilesDirectories, state.Directory{
				Path: state.EnsureDirectoryPath(dir),
			})
		}
	}

	go func() {
		server.AddDirectories(mediaFilesDirectories)
	}()

	err = server.Serve()
	if err != nil {
		errLog.Printf("API server finished with following error: %s\n", err)
	}
}
