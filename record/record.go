package record

type ExperimentRecord struct {
	Name       string  `json:"name"`
	MaxValue   float64 `json:"max_value"`
	MinValue   float64 `json:"min_value"`
	NodeAmount int     `json:"node_amount"`
	Decay      float64 `json:"decay"`
	Mse        float64 `json:"mse"`
	MeanLost   float64 `json:"mean_lost"`
	MeanRound  float64 `json:"mean_round"`
}
