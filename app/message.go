package app

const RejectUpdate = -1
const Updated = 0
const Skip = -2

type Message struct {
	Data     float64 `json:"data"`
	From     string  `json:"from"`
	Callback chan *Response
}

type Response struct {
	Code int
	Data float64
}
