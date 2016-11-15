package firebase

import (
	"sync"
	"testing"
)

func TestGeneratePushID(t *testing.T) {
	a, b := GeneratePushID(), GeneratePushID()
	if len(a) != 20 {
		t.Errorf("length of a should be 20, got: %d", len(a))
	}

	if len(b) != 20 {
		t.Errorf("length of b should be 20, got: %d", len(b))
	}

	if a == b {
		t.Errorf("a (%s) and b (%s) should not be same", a, b)
	}

	if !(a < b) {
		t.Errorf("a (%s) should be < than b (%s)", a, b)
	}
}

func TestGeneratePushIDMany(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)

		go func(t *testing.T, wg *sync.WaitGroup) {
			defer wg.Done()

			id, prev := "", ""
			ids := make(map[string]bool)
			for i := 0; i < 1000000; i++ {
				id = GeneratePushID()
				if len(id) != 20 {
					t.Fatalf("length of id should be 20, got: %d", len(id))
				}

				if _, exists := ids[id]; exists {
					t.Fatalf("should not have generated duplicate id %s", id)
				}

				if !(prev < id) {
					t.Fatalf("prev id %s is not < than generated id %s", prev, id)
				}

				ids[id] = true
				prev = id
			}
		}(t, &wg)
	}
	wg.Wait()
}
