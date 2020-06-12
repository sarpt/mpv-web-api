# mpv web api

remote mpv control through rest api proof of concept

## proof of concept execution

after building the the binary `mpv-web-api` simply run it. to check if it's working: `curl --data "path=/path/to/file.ext" http://localhost:3001/playback`