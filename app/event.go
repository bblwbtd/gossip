package app

const ChangeState = 0
const UpdateValue = 1
const End = 2

type Event struct {
	From string      `json:"from"`
	Type int         `json:"type"`
	Data interface{} `json:"data"`
}
