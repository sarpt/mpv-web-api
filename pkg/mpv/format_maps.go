package mpv

type PlaylistMap struct {
	Filename string `json:"filename"`
	Current  bool   `json:"current"`
	Playing  bool   `json:"playing"`
	Title    string `json:"title"`
	ID       int64  `json:"id"`
}
