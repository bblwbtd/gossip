package model

type NodeConfig struct {
	Max      float64 `json:"max"`
	Min      float64 `json:"min"`
	Delay    int     `json:"delay"`
	Decay    float64 `json:"decay"`
	MaxRound int     `json:"max_round"`
}
