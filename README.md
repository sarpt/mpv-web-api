# MPV Web API

Remote MPV control through REST API with state notifications carried by Server Sent Events.

The server is a proof of concept/toy project and as such it sometimes works, and sometimes doesn't - it's a constant Work In Progress until it's not (if ever...).

It's main use is to be a backend for [mpv-web-front](https://github.com/sarpt/mpv-web-front), which is also a toy project and is in the same constant state of being Work In Progress - the compatibility between the two of them is not ensured yet, however I try to ensure that HEADs from both repos work with each other. If they don't, then the best bet is to try using HEAD^ from either one of them and check if they work with each other.

### Dependencies for running

- `mpv` - used for playback
- `ffprobe` (part of ffmpeg/libav collection of programs) - used for media files probing

### Dependencies for builidng

- `go` (tested on `1.21.0`. the docker image is built with `1.21.0`)

### Arguments

- `allow-cors` - bool - (default: `false`) whether Cross Origin Requests should be allowed
- `addr` - string - (default: `localhost:3001`) address used to host the server
- `dir` - []string - (default: current working directory) directories that should be scanned for media files. To specify more than one directory to be handled, multiple `--dir=<path>` arguments can be specified eg. `--dir=/path1 --dir=/path2`. The server will only handle paths provided by clients that start with one of the paths provided to `dir`. When not provided, current working directory for the process will be used to scan for media files. Recursive scan can be enabled with `--dir-recursive`. Watching for the changes to the provided directories can be enabled with `--watch-dir`.
- `dir-recursive` - bool - directories provided to `--dir` (or working directory when `--dir` is not provided) will be checked recursively.
- `mpv-socket-path` - string - (default: `/tmp/mpvsocket`) path to socket file used by MPV instance
- `path-replacements` - []string - list of path replacements mappings that will be used when communicating with mpv process. The mapping entry takes form of a `<from>:<to>` string, eg. `/some/path:/replacement/path`. When provided multiple times, the order of specified arguments will be the order in which server applies replacements to the paths. When path matches multiple (or even all) replacements, then all of matching replacements will be applied.
- `playlist-prefix` - []string - list of prefixes for playlist JSON files located in directories being handled by the server instance. For more informations on playlists please check related section.
- `socket-timeout` - int - (defualt: `15`) maximum allowed time in seconds for retrying connection to MPV socket
- `start-mpv-instance` - bool - (default: `true`) when set to true, `mpv-web-api` will create it's own MPV process. When set to false, `mpv-web-api` will only try to connect to MPV using file at `mpv-socket-path`. Particularly useful when trying to run `mpv-web-api` in docker and connecting to a local MPV instance
- `watch-dir` - bool - (default: `false`) directories provided to `--dir` (or working directory when `--dir` is not provided) will be watched for future changes to the underlying files (addition, deletion).

### Building & Execution

To build and install the application, in terminal navigate to `<repo-root>/cmd/mpv-web-api` and run `go build && go install`. 

After building the the binary `mpv-web-api` simply run it. To check if it's working: `curl --data "path=/path/to/file.ext" http://localhost:3001/playback` - the invocation should return current state of the playback (if anything's playing).

In case the server is ran to serve as a backend to `mpv-web-front` on a local machine it is recommended to pass `--allow-cors` argument, otherwise the communication might be blocked.

### Building & execution with Docker

Running a server instance in a docker container can be achieved by starting `mpv` on a host machine with specified socket path and mounting that socket path inside a running container - this way image will satisfy all build-time and runtime dependencies like `go` and `ffprobe`, except `mpv` which should be ran on the host.

From root repostiory dir run `docker build -t mpv-web-api:latest -f ./build/package/Dockerfile .` to build the application image.

The resulting image will automatically apply `--start-mpv-instance=false`, since in most (all?) cases the `mpv` should be running on a host machine, not inside a container. That however requires passing `mpv-socket-path` if the socket is mapped to a different path inside a running container than the path mpv checks by default (`/tmp/mpvsocket`).

Minimal `mpv` instance prepared for communication with the server can be spawned with a following command:

`mpv --input-ipc-server=/tmp/mpvsocket --idle=yes`

Having started `mpv` process, the next step is to run `mpv-web-api:latest` image created earlier with necessary mountpoints and arguments. Few things that need to be taken into consideration when running the image:
- port (by default `3001`) has to be mapped from the container into a host
- directory with media files has to be mapped from host to a container and the appropriate dir argument has to be provied that points to the mapped point inside the container
- unless the mapped mountpoints mentioned in the previous point result in the same absolute paths in the container as they are on the host, the `path-replacements` argument should be used to translate paths from a mountpoint path to a host path, otherwise `mpv` running on the host will be fed with mapped mountpoints which might result in file loading errors due to incorrect paths
- socket on which `mpv` listens has to be mapped from host to a continer and the appropriate `mpv-socket-path` argument must be provided that points to the mapped socket inside the container, if it was mapped to a different path than the one the server checks by default (`/tmp/mpvsocket`)
- all cors rules apply in the same way as for the process running on host machine, which means when connecting from locally served frontend (`localhost`) it might be neccessary to pass `--allow-cors` 

The image itself does not assume any defaults as to where directories and socket are mapped inside the container (in contrast to defaults of the server itself) so those have to be taken care of when providing a `docker run` command.

`docker run -it --rm -v <host path to socket>:/tmp/mpvsocket -v <host path to mediafiles>:/mnt/videos -p <host port>:3001 mpv-web-api:latest --allow-cors --dir=/mnt/videos --path-replacements=/mnt/videos:<host path to mediafiles> --watch-dir`
e.g:
`docker run -it --rm -v /tmp/mpvsocket:/tmp/mpvsocket -v /home/user/videos:/mnt/videos -p 3001:3001 mpv-web-api:latest --allow-cors --dir=/mnt/videos --path-replacements=/mnt/videos:/home/user/videos --watch-dir`

### REST endpoints

Many REST endpoints are not implemented yet, since `mpv-web-front` mostly uses SSEs to sync it's state (REST would require some kind of polling), while using REST mainly for sending instructions to server (and by extension, mpv). Target implementation however has whole REST/SSE parity planned when it comes to information fetching (`GET`s).

- `DELETE "/directories"` - deletes directories that match paths provided as a URL query `path` parameters. Deleting a directory stops watch of new files and removes it's content from being played.
  - `path` - string[] - paths to directories client wishes to be deleted. In URI it takes the form of `/rest/directories?path=%2Fpath%2Fto%2Fdir%2`. The path should be escaped, although unescaped *may* work. The ending separator is not necessary to be present. 
- `GET "/directories"` - returns information about directories handled by the server instance: their paths and whether the directory is watched for changes
- `POST "/directories"` - add directory with media files for server to handle
  - `path` - string - path to a directory to be added (recursive) 
  - `watched` - bool (default: `false`) - whether directory should be watched for changes underneath (added/removed files), instead of being read once
- `GET "/media-files"` - returns information about the media files: their paths and video, audio & subtitles streams
- `POST "/playback"` - change current playback. Playback is a state which determines current file, playlist position, media timeline position, selection of subtitle and audio streams, etc.
  - `append` - bool (default: `false`) - when set to `true` with `path`, it append path as a next entry in currently played playlist (whether named/saved or not). When set to `false`, file under `path` will be played immediately, basically creating a new unnamed/empty playlist with only one item in it.
  - `audioID` - string - selects audio stream with the provided id. Although a string, mpv indexes its audio streams, so it will have numerical form.
  - `chapter` - int - selects chapter.
  - `chapters` - int[] - controls order of chapters playback. The argument takes form of a chapter indexes list (0-based) separated by `,` eg. `2,3,5`. When file is being looped, the chapters order will be enforced through every loop of the file, until next media file starts. The list accepts repetitions (eg. `2,2,4,5,5`). At the moment list enforces sorting of the indexes (eg. `1,4,5` is correct but `5,4,1` is not), but a target functionality of this features plans for sorting to be irrelevant. By default, the argument is not applied immediately, as such it will be enabled with the first chapter change in the file, so the current chapter can be finished without initial jump, but compound argument `force` can be used to force chapters order restrictions immediately (currently played chapter will be changed if neccessary).
  - `force` - bool (default: `false`) - forces changes for applicable arguments: `chapters`
  - `fullscreen` - bool (default: `false`) - selects fullscreen state to enabled/disabled.
  - `loopFile` - bool (default: `false`) - selects looping of currently played file to enabled/disabled.
  - `path` - string - path of the currently played media. The `mpv-web-api` has to have access to this directory and the directory needs to be probed for media files.
  - `pause` - bool (default: `false`) - selects paused state of playback. It need to be noted that playback being `paused` is not equal to being `stopped` - the former will keep playback state, which means the mpv will pause the playback and will still show everything, while latter will just trigger idle mode in the mpv instance.
  - `playlistIdx` - int - changes currently played entry in a playlist.
  - `playlistUUID` - string - selects currently played playlist. UUID is a server-generated identifier and is transparent to an mpv instance.
  - `stop` - bool (default: `false`) - stops mpv playback, clearing the playback state and instructing mpv instance to go into idle.
  - `subtitleID` - string - selects subtitle stream with the provided id. Although a string, mpv indexes its subtitle streams, so it will have numerical form.
- `GET "/playback"` - returns the state of mpv current playback.
- `GET "/playlists"` - returns playlists handled by the api server.
- `GET "/sse/channels"` - registers client to the SSE channels, estabilishing long running connection.
  - `channel` - string[] - names of channels client wishes to be subscribed to. In URI it takes the form of `/sse/register?channel=name1&channel=name2&channel... etc`.
  - `replay` - bool (default: `false`) - when set to `true`, the first emitted event will be of `replay` type. More info on types of SSE events in a related section below.

### Server Sent Events
SSE is used by server to notify reactively of changes to it's various states, eg. change of currently played media file in playback, new media files added, new directories added, changes to the current playlist etc. SSE communication is optional and the idea is to have all states queryable by REST API endpoints with `GET`, but since this is a constant WIP it may not always be the case - SSE takes priority in implementation since `mpv-web-front` uses it primarily to get updates without polling the server constantly.

Events are aggregated into channels, to which client subscribes. Due to limits of simultaneous connections count to a singular target, client should have only one open SSE connection that receives updates to all of the subscribed channels (that's how `mpv-web-front` does it). Since a standard Server Sent Event has `name` and `data` fields, the grouping has to be incorporated into one of them. For transparency of mechanism and ease of use, `mpv-web-api` uses name field with a `.` separator to specify on which channel event was broadcasted, eg. `mediaFiles.added` is an `added` event on a `mediaFiles` channel etc.

In order to subsrcibe to channel(s), `/sse` REST endpoint should be used - how to use it is explained in the related REST endpoints section above.

Some notes about weird/unexpected behavior that will at some point be changed/solved/cleared:
- `replay` event is a special one, which is used to replay the whole state. It will be changed to `all` or something else - it's a legacy name that outlived it's temporary meaning and it's temporary solution implementation. It's used by `mpv-web-front` to "get a replay" of all data when connecting fresh or after a reconnect, instead of getting only chunks of data (get all media files to have possibility to handle added media files or updated media files additively). Since most of the events provide whole state anyway, it's usefullness and name are highly debatable. Although redundant, it's mentioned in all channels below (because why not).
- some events provide diferential state changes, while some always emit the whole state. Target behavior will be to have differential changes sent in all cases except for the currently ill-named `replay` events. That however will require a rewrite which I'm more willing to take when generics land in go (even if that means being dependent on the beta `go` release, like expected `go1.18 beta`). While generics clearly aren't "necessary" for rewrite, my intuition is telling me that resulting code will be easier on the eyes and soul rather than whatever form it will take without them (future will tell whether my intuition is right, exciting!).
- events that have ~~strikethrough~~ are to be implemented "soon-ish" - as soon as I find ~~interest~~ ~~energy~~ ~~faith~~ use for them...
- overall, server's implementation of SSEs is a chaotic mess right now and there's no guarantee for their behavior and contents. This section will be updated (hopefully) as soon as they are more "stable" ~~(or I find another cool way of providing updates to clients - I'm looking at you GraphQL)~~

Channels and their events:
- `directories` - events fire in response to changes in directories being handled by server's instance
  - `replay` - list of all directories
  - `added` - list of addded directories
  - `removed` - list of removed directories
- `mediaFiles` - events fire in response to changes in watched media files
  - `replay` - list of all media files 
  - `added` - list of added media files
  - ~~`updated` - list of added media files~~ 
  - `removed` - list of removed media files
- `playback` (all events provide whole playback state) - events fire mostly in response to mpv changing it's playback-related properties
  - `replay` - whole playback state
  - `fullscreenChange` -  mpv changed it's `fullscreen` property
  - `loopFileChange` - mpv changed it's `loop-file` property
  - `pauseChange` - mpv changed it's `pause` property
  - `audioIdChange` - mpv changed it's `aid` property
  - `playbackStoppedChange` - mpv changed it's `path` property but did not provide a new path (path is empty) 
  - `subtitleIdChange` - mpv changed it's `sid` property
  - ~~`currentChapterIndexChange` - mpv changed it's `chapter` property~~
  - `mediaFileChange` - mpv changed it's `path` property. Name of the event is ill-named, will be changed either to `pathChanged` or `fileChanged`
  - `playbackTimeChange` - mpv changed it's `playback-time` property
  - `playlistSelectionChange` - mpv changed it's `playlist` format node property - currently played playlist changed. This event is only partially mapped to mpv behavior, since playlists management is partially managed by the server.
  - `playlistCurrentIdxChange` - mpv changed it's `playlist-playing-pos` format node property - currently played entry in a playlist changed
- `playlists` (all events provide whole playlist state) - events fire in response to external (and internal) requests to server related to playlists handling
  - `replay` - list of all playlists
  - `added` - a new playlist was added either by server itself (default/unnamed playlist) or an external client
  - `itemsChange` - set of playlist entries/items changed
- `status` - (all events provide whole status state) -events fire in response to changes in server's runtime state
  - `replay` - whole status state
  - `client-observer-added` - a new SSE client observer was added
  - `client-observer-removed` - SSE client observer was removed (most probably disconnected on it's own, but not guarenteed)
  - ~~`mpv-process-changed` - when server manages it's own mpv process, this event fires when server creates mpv process (changed not necesarilly means that a process existed beforehand)~~

### Playlists

For a file to be considered a playlist it has to:
- be a valid JSON file
- have a name which matches any of the passed prefixes with `--playlist-prefix`
- have property `MpvWebApiPlaylist` set to `true`

A playlist file is a file that contains a valid JSON object. The properties at the top level of the object:
- `MpvWebApiPlaylist` - bool - a flag that specifies whether `mpv-web-api` should treat this file as a playlist. The property is used as a failsafe in case other valid JSON files are present in the directory and match specified playlist prefix
- `Entries` - []PlaylistEntry - a list of objects specifying entries to be treated to be played by a playlist.
- `Name` - string - a name of the playlist (usage up to the client)
- `Description` - string - a description of the playlist (usage up to the client)
- `DirectoryContentsAsEntries` - bool - when set to true, `Entries` field is ignored and replaced with media files found in the directory the playlist file is in.

Example:
```
{
	"MpvWebApiPlaylist": true,
	"Entries": [],
	"Name": "Example playlist",
	"Description": "Some description"
}
```

Entries are instances of an object with the following fields:
- `Path` - absolute path to the playlist entry
- `PlaybackTimestamp` - timestamp from which the entry should start playing
- `AudioID` - audio id for playlist entry
- `SubtitleID` - subtitle id for playlist entry