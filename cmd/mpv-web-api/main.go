package main

import (
	"flag"
	"fmt"
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
	defaultAddress           = "localhost:3001"
	defaultSocketTimeoutSec  = 15
	defaultMpvSocketFilename = "mpvsocket"
	addressFlag              = "addr"
	allowCorsFlag            = "allow-cors"
	dirFlag                  = "dir"
	dirRecursiveFlag         = "dir-recursive"
	mpvSocketPathFlag        = "mpv-socket-path"
	playlistPrefixFlag       = "playlist-prefix"
	socketTimeoutSecFlag     = "socket-timeout"
	startMpvInstanceFlag     = "start-mpv-instance"
	watchDirFlag             = "watch-dir"
	watchDirRecursiveFlag    = "watch-dir-recursive"
)

var (
	address              *string
	allowCORS            *bool
	defaultMpvSocketPath = fmt.Sprintf("%s%c%s", os.TempDir(), os.PathSeparator, defaultMpvSocketFilename)
	dir                  *listflag.StringList
	dirRecursive         *listflag.StringList
	mpvSocketPath        *string
	playlistPrefix       *listflag.StringList
	socketTimeoutSec     *int64
	startMpvInstance     *bool
	watchDir             *listflag.StringList
	watchDirRecursive    *listflag.StringList
)

func init() {
	dir = listflag.NewStringList([]string{})
	dirRecursive = listflag.NewStringList([]string{})
	watchDir = listflag.NewStringList([]string{})
	watchDirRecursive = listflag.NewStringList([]string{})
	playlistPrefix = listflag.NewStringList([]string{})

	flag.Var(dir, dirFlag, "directory containing media files - the directory is not watched for changes")
	flag.Var(dirRecursive, dirRecursiveFlag, "recursive variant of --dir flag: handles all directories in the tree from the path downards")
	address = flag.String(addressFlag, defaultAddress, "address on which server should listen on")
	allowCORS = flag.Bool(allowCorsFlag, false, "when not provided, Cross Origin Site Requests will be rejected")
	mpvSocketPath = flag.String(mpvSocketPathFlag, defaultMpvSocketPath, "path to a socket file used by a MPV instance to listen for commands")
	flag.Var(playlistPrefix, playlistPrefixFlag, "prefix for JSON files to be treated as playlists. The JSON file itself has to have in the root object property 'MpvWebApiPlaylist' set to true to be treated as a playlist")
	socketTimeoutSec = flag.Int64(socketTimeoutSecFlag, defaultSocketTimeoutSec, "maximum allowed time in seconds for retrying connection to MPV instance")
	startMpvInstance = flag.Bool(startMpvInstanceFlag, true, "controls whether the application should create and manage its own MPV instance")
	flag.Var(watchDir, watchDirFlag, "directory containing media files - the directory will be watched for changes. When left empty and no --dir arguments are specified, current working directory will be used")
	flag.Var(watchDirRecursive, watchDirRecursiveFlag, "recursive variant of --watch-dir flag: handles all directories in the tree from the path downards")

	flag.Parse()
}

func main() {
	errLog := log.New(os.Stderr, logPrefix, log.LstdFlags)
	outLog := log.New(os.Stdout, logPrefix, log.LstdFlags)

	socketConnectionTimeout := time.Duration(time.Duration(*socketTimeoutSec) * time.Second)
	cfg := api.Config{
		Address:                 *address,
		AllowCORS:               *allowCORS,
		MpvSocketPath:           *mpvSocketPath,
		PlaylistFilesPrefixes:   playlistPrefix.Values(),
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
			Path:      wd,
			Recursive: true,
			Watched:   true,
		})
	} else {
		for _, dir := range watchDir.Values() {
			mediaFilesDirectories = append(mediaFilesDirectories, state.Directory{
				Path:    dir,
				Watched: true,
			})
		}

		for _, dir := range watchDirRecursive.Values() {
			mediaFilesDirectories = append(mediaFilesDirectories, state.Directory{
				Path:      dir,
				Recursive: true,
				Watched:   true,
			})
		}

		for _, dir := range dir.Values() {
			mediaFilesDirectories = append(mediaFilesDirectories, state.Directory{
				Path: dir,
			})
		}

		for _, dir := range dirRecursive.Values() {
			mediaFilesDirectories = append(mediaFilesDirectories, state.Directory{
				Path:      dir,
				Recursive: true,
			})
		}
	}

	go func() {
		server.AddRootDirectories(mediaFilesDirectories)
	}()

	err = server.Serve()
	if err != nil {
		errLog.Printf("API server finished with following error: %s\n", err)
	}
}
