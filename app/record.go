package app

type Record struct {
	Sender   string  `json:"sender"`
	Receiver string  `json:"receiver"`
	Data     float64 `json:"data"`
}

func NewRecord(sender string, receiver string, data float64) *Record {
	return &Record{Sender: sender, Receiver: receiver, Data: data}
}
