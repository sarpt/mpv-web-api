# mpv web api

remote mpv control through rest api proof of concept

### dependencies for running

- `mpv` - used for playback
- `ffprobe` (part of ffmpeg/libav collection of programs) - used for media files probing

### dependencies for builidng

- `go` (tested on 1.14, might build on earlier versions)

### endpoints

- `GET "/movies"` - returns information about the movies: their paths and video, audio & subtitles streams. Media files without video stream will not be returned here
- `POST "/playback"` - with `"path"` key and value as a string path to the file will play the file on the started by the api server mpv binary
- `GET "/playback"` - returns the state of mpv current playback. At the moment only information on the movie being played and it's filepath are present

### arguments

- `dir` - []string - directories that should be scanned for media files
- `allow-cors` - bool - (default: false) whether Cross Origin Requests should be allowed
- `addr` - string - (default: localhost:3001) which address should be used to host the server

### proof of concept execution

After building the the binary `mpv-web-api` simply run it. to check if it's working: `curl --data "path=/path/to/file.ext" http://localhost:3001/playback`