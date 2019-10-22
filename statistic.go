package main

import (
	"fmt"
	"gossip/app"
	"gossip/record"
	"math"
)

func GetMinValue() float64 {
	min := math.MaxFloat64
	nodeTable.Range(func(key, value interface{}) bool {
		node := value.(*app.Node)
		if node.Data < min {
			min = node.Data
		}
		return true
	})
	return min
}

func GetMaxValue() float64 {
	max := -math.MaxFloat64
	nodeTable.Range(func(key, value interface{}) bool {
		node := value.(*app.Node)
		if node.Data > max {
			max = node.Data
		}
		return true
	})
	return max
}

func GetMaxRound() int {
	max := 0
	nodeTable.Range(func(key, value interface{}) bool {
		node := value.(*app.Node)
		if len(node.Records) > max {
			max = len(node.Records)
		}
		return true
	})
	return max
}

func GetMinRound() int {
	min := math.MaxInt32
	nodeTable.Range(func(key, value interface{}) bool {
		node := value.(*app.Node)
		if len(node.Records) < min {
			min = len(node.Records)
		}
		return true
	})
	return min
}

func GetMean() float64 {
	total := 0.0
	count := 0.0
	nodeTable.Range(func(key, value interface{}) bool {
		node := value.(*app.Node)
		total = total + node.Data
		count++
		return true
	})
	return total / count
}

func GetMSE(correct float64) float64 {
	sum := 0.0
	count := 0.0
	nodeTable.Range(func(key, value interface{}) bool {
		node := value.(*app.Node)
		sum = sum + (node.Data-correct)*(node.Data-correct)
		count++
		return true
	})
	return sum / count
}

func GetDecay() float64 {
	decay := 0.0
	count := 0.0
	nodeTable.Range(func(key, value interface{}) bool {
		node := value.(*app.Node)
		decay = decay + node.Config.Decay
		count++
		return true
	})
	return decay / count
}

func GetMeanRound() float64 {
	round := 0.0
	count := 0.0
	nodeTable.Range(func(key, value interface{}) bool {
		node := value.(*app.Node)
		round = round + float64(len(node.Records))
		count++
		return true
	})
	return round / count
}

func GetRecordsByNodeAmount() map[string][]*record.ExperimentRecord {
	result := make(map[string][]*record.ExperimentRecord)
	record.Storage.Range(func(key, value interface{}) bool {
		experimentRecord := value.(*record.ExperimentRecord)
		result[fmt.Sprintf("%d", experimentRecord.NodeAmount)] = append(result[fmt.Sprintf("%d", experimentRecord.NodeAmount)], experimentRecord)
		return true
	})
	return result
}

func GetRecordsByDecay() map[string][]*record.ExperimentRecord {
	result := make(map[string][]*record.ExperimentRecord)
	record.Storage.Range(func(key, value interface{}) bool {
		experimentRecord := value.(*record.ExperimentRecord)
		result[fmt.Sprintf("%.2f", experimentRecord.Decay)] = append(result[fmt.Sprintf("%.2f", experimentRecord.Decay)], experimentRecord)
		return true
	})
	return result
}

func GetAllRecords() []*record.ExperimentRecord {
	result := make([]*record.ExperimentRecord, 0)
	record.Storage.Range(func(key, value interface{}) bool {
		result = append(result, value.(*record.ExperimentRecord))
		return true
	})
	return result
}
