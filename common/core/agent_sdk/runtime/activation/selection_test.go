package activation

import "testing"

func TestSelectOrdersByPriorityScoreNameAndFiltersMutex(t *testing.T) {
	selected := Select([]Candidate[string]{
		{Item: "low", Name: "low", Priority: 1, Score: 1},
		{Item: "mutex-loser", Name: "b", Priority: 2, Score: 0.4, MutexKey: "group"},
		{Item: "mutex-winner", Name: "a", Priority: 2, Score: 0.9, MutexKey: "group"},
		{Item: "tie-name", Name: "c", Priority: 2, Score: 0.9},
	})

	got := make([]string, len(selected))
	for i, candidate := range selected {
		got[i] = candidate.Item
	}
	want := []string{"mutex-winner", "tie-name", "low"}
	if len(got) != len(want) {
		t.Fatalf("selected = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("selected = %#v, want %#v", got, want)
		}
	}
}
