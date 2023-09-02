package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/sarpt/goutils/pkg/listflag"

	"github.com/sarpt/mpv-web-api/internal/rest"
	"github.com/sarpt/mpv-web-api/internal/sse"
	"github.com/sarpt/mpv-web-api/pkg/api"
	"github.com/sarpt/mpv-web-api/pkg/state"
	"github.com/sarpt/mpv-web-api/pkg/state/pkg/directories"
)

const (
	logPrefix         = "cmd.main#"
	pathMappingSymbol = ":"
)

const (
	defaultAddress           = ":3001"
	defaultSocketTimeoutSec  = 15
	defaultMpvSocketFilename = "mpvsocket"
	defaultAppDirName        = "mwa"

	addressFlag          = "addr"
	allowCorsFlag        = "allow-cors"
	dirFlag              = "dir"
	dirRecursiveFlag     = "dir-recursive"
	mpvSocketPathFlag    = "mpv-socket-path"
	pathMappingsFlag     = "path-mappings"
	playlistPrefixFlag   = "playlist-prefix"
	socketTimeoutSecFlag = "socket-timeout"
	startMpvInstanceFlag = "start-mpv-instance"
	appDirFlag           = "app-dir"
	watchDirFlag         = "watch-dir"
)

var (
	defaultMpvSocketPath = fmt.Sprintf("%s%c%s", os.TempDir(), os.PathSeparator, defaultMpvSocketFilename)

	address          *string
	allowCORS        *bool
	dir              *listflag.StringList
	dirRecursive     *bool
	mpvSocketPath    *string
	pathMappings     *listflag.StringList
	playlistPrefix   *listflag.StringList
	socketTimeoutSec *int64
	startMpvInstance *bool
	appDir           *string
	watchDir         *bool
)

func init() {
	dir = listflag.NewStringList([]string{})
	pathMappings = listflag.NewStringList([]string{})
	playlistPrefix = listflag.NewStringList([]string{})

	appDir = flag.String(appDirFlag, "", "path which should be used for persistence storage by the server for saving unnamed playlists, configs, caches, etc.")
	flag.Var(dir, dirFlag, "directory containing media files. When not provided current working directory for the process is being used")
	dirRecursive = flag.Bool(dirRecursiveFlag, true, "when not provided, directories provided to --dir (or working directory when --dir is absent) will only be checked on the first level and any directories within will be ignored")
	address = flag.String(addressFlag, defaultAddress, "address on which server should listen on")
	allowCORS = flag.Bool(allowCorsFlag, false, "when not provided, Cross Origin Site Requests will be rejected")
	mpvSocketPath = flag.String(mpvSocketPathFlag, defaultMpvSocketPath, "path to a socket file used by a MPV instance to listen for commands")
	flag.Var(pathMappings, pathMappingsFlag, "path parts to be replaced when providing them to mpv process. The mapping is in a form of <path-to-be-replaced>:<replacement-path>. Each path is matched against every replacement provided, even when previous replacements matched")
	flag.Var(playlistPrefix, playlistPrefixFlag, "prefix for JSON files to be treated as playlists. The JSON file itself has to have in the root object property 'MpvWebApiPlaylist' set to true to be treated as a playlist")
	socketTimeoutSec = flag.Int64(socketTimeoutSecFlag, defaultSocketTimeoutSec, "maximum allowed time in seconds for retrying connection to MPV instance")
	startMpvInstance = flag.Bool(startMpvInstanceFlag, true, "controls whether the application should create and manage its own MPV instance")
	watchDir = flag.Bool(watchDirFlag, false, "when not provided, directories provided to --dir (or working directory when --dir is absent) will only be checked once at a startup and files adding/removal in these directories during runtime will be ignored")

	flag.Parse()
}

func main() {
	errWriter := os.Stderr
	outWriter := os.Stdout

	errLog := log.New(errWriter, logPrefix, log.LstdFlags)
	outLog := log.New(outWriter, logPrefix, log.LstdFlags)

	parsedAppDir, err := handleAppDir(*appDir)
	if err != nil {
		errLog.Printf("could not use \"%s\" as an application directory - reason: %s", parsedAppDir, err)
		os.Exit(1)
	}

	outLog.Printf("server uses as \"%s\" as an application directory", parsedAppDir)

	statesRepository := state.NewRepository()
	sseCfg := sse.Config{
		ErrWriter:        errWriter,
		OutWriter:        outWriter,
		StatesRepository: statesRepository,
	}
	sseServer := sse.NewServer(sseCfg)

	restCfg := rest.Config{
		AllowCORS:        *allowCORS,
		ErrWriter:        errWriter,
		OutWriter:        outWriter,
		StatesRepository: statesRepository,
	}
	restServer := rest.NewServer(restCfg)

	socketConnectionTimeout := time.Duration(time.Duration(*socketTimeoutSec) * time.Second)

	pathMappingsList := []api.PathMapping{}
	for _, replacement := range pathMappings.Values() {
		split := strings.Split(replacement, pathMappingSymbol)
		pathMappingsList = append(pathMappingsList, api.PathMapping{To: split[1], From: split[0]})
	}

	cfg := api.Config{
		Address:               *address,
		AppDir:                parsedAppDir,
		AllowCORS:             *allowCORS,
		MpvSocketPath:         *mpvSocketPath,
		PathMappings:          pathMappingsList,
		PlaylistFilesPrefixes: playlistPrefix.Values(),
		PluginServers: map[string]api.PluginServer{
			sseServer.Name():  sseServer,
			restServer.Name(): restServer,
		},
		StartMpvInstance:        *startMpvInstance,
		StatesRepository:        statesRepository,
		SocketConnectionTimeout: socketConnectionTimeout,
	}

	server, err := api.NewServer(cfg)
	if err != nil {
		errLog.Printf("could not start API server due to an error: %s\n", err)

		return
	}

	var mediaFilesDirectories []directories.Entry
	watchWorkingDir := len(dir.Values()) == 0
	if watchWorkingDir {
		wd, err := os.Getwd()
		if err != nil {
			errLog.Printf("could not start API server due to error while getting working directory: %s\n", err)

			return
		}

		outLog.Printf("no directories specified for server - using working directory '%s'\n", wd)
		mediaFilesDirectories = append(mediaFilesDirectories, directories.Entry{
			Path:      wd,
			Recursive: *dirRecursive,
			Watched:   *watchDir,
		})
	} else {
		for _, dir := range dir.Values() {
			mediaFilesDirectories = append(mediaFilesDirectories, directories.Entry{
				Path:      dir,
				Recursive: *dirRecursive,
				Watched:   *watchDir,
			})
		}
	}

	// add root directories but do not wait for server to start serving
	go func() {
		server.AddRootDirectories(mediaFilesDirectories)
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-signals
		err := server.StopServing(sig.String())
		if err != nil {
			errLog.Printf("stop of API server finished with an error: %s", err)
		}
	}()

	err = server.Serve()
	if err != nil {
		errLog.Printf("API server finished with following error: %s\n", err)
	} else {
		outLog.Println("API server finished successfully")
	}
}

func handleAppDir(appDir string) (string, error) {
	if appDir == "" {
		appDir = getDefaultAppDir()
	}

	return appDir, nil
}

func getDefaultAppDir() string {
	var appPathDefaultBase string
	homeDir, err := os.UserHomeDir()
	if err != nil {
		appPathDefaultBase = os.TempDir()
	} else {
		appPathDefaultBase = homeDir
	}

	return fmt.Sprintf("%s%c%s", appPathDefaultBase, os.PathSeparator, defaultAppDirName)
}
