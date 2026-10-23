package testsupport

import (
	"encoding/json"
	"sync"
	"testing"
)

func TestTheStandInGhKeepsEveryCallWhenSeveralRunAtOnce(t *testing.T) {
	f := &FakeGh{Routes: map[string]json.RawMessage{"GET search/issues": json.RawMessage(`{"items":[]}`)}}
	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 20 {
				f.Run([]string{"api", "search/issues"}, nil, nil)
			}
		}()
	}
	wg.Wait()
	if len(f.Calls) != 1000 {
		t.Errorf("of 1000 calls at once, %d were kept", len(f.Calls))
	}
}
