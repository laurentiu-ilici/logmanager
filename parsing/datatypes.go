package parsing

type LogLine struct {
	Id          string
	Start       string
	End         string
	ServiceName string
	Caller      string
	Callee      string
}

//Call type to marshal to json
type call struct {
	Start   string `json:"start"`
	End     string `json:"end"`
	Service string `json:"service"`
	Span    string `json:"span"`
	Calls   []call `json:"calls"`
}

//LogResult type to marshal to json
type logResult struct {
	Id   string `json:"id"`
	Root call   `json:"root"`
}
