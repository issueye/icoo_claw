package activation

import "sort"

// Candidate is a generic activation candidate ranked by priority, score, and
// name. MutexKey keeps only the best candidate for a mutually exclusive group.
type Candidate[T any] struct {
	Item     T
	Name     string
	Priority int
	Score    float64
	MutexKey string
}

// Select orders candidates and filters mutually exclusive groups. Higher
// priority wins, then higher score, then lexical name for deterministic output.
func Select[T any](candidates []Candidate[T]) []Candidate[T] {
	if len(candidates) == 0 {
		return nil
	}
	out := append([]Candidate[T](nil), candidates...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Priority != out[j].Priority {
			return out[i].Priority > out[j].Priority
		}
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].Name < out[j].Name
	})

	selected := out[:0]
	seen := map[string]struct{}{}
	for _, candidate := range out {
		if candidate.MutexKey == "" {
			selected = append(selected, candidate)
			continue
		}
		if _, ok := seen[candidate.MutexKey]; ok {
			continue
		}
		seen[candidate.MutexKey] = struct{}{}
		selected = append(selected, candidate)
	}
	return selected
}
