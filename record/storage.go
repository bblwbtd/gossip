package record

import (
	"sync"
)

var Storage sync.Map

func AddRecord(record *ExperimentRecord) {
	Storage.Store(record.Name, record)
}
