package memorymanager

import (
	"math"

	"github.com/w-h-a/agent/memory_manager/providers/storer"
)

func Select(records []storer.Record, vec []float32, limit int, relevance float64) []storer.Record {
	if len(records) <= limit {
		return records
	}

	if relevance < 0 {
		relevance = 0
	} else if relevance > 1 {
		relevance = 1
	}

	selected := make([]storer.Record, 0, limit)
	copied := append([]storer.Record(nil), records...)

	for len(selected) < limit && len(copied) > 0 {
		bestIdx := -1
		best := math.Inf(-1)

		for i, cand := range copied {
			score := float64(cand.Score)
			maxSim := 0.0

			for _, sel := range selected {
				if sim := CosineSimilarity(cand.Embedding, sel.Embedding); sim > maxSim {
					maxSim = sim
				}
			}

			current := (relevance * score) - ((1 - relevance) * maxSim) // reward minus redundant

			// if we want pure diversity after similarity matching, we want the item closest to 0
			// also if selected is empty then we keep maxSim at 0
			if relevance == 0 && len(selected) > 0 {
				current = -maxSim
			}

			if current > best {
				best = current
				bestIdx = i
			}
		}

		if bestIdx != -1 {
			selected = append(selected, copied[bestIdx])
			copied = append(copied[:bestIdx], copied[bestIdx+1:]...)
		} else {
			break
		}
	}

	return selected
}

func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 || len(b) == 0 {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

func NormalizeWeights(weights Weights) (sim, rec float64) {
	sum := weights.Similarity + weights.Recency
	if sum == 0 {
		return 0.5, 0.5
	}
	sim = weights.Similarity / sum
	rec = weights.Recency / sum
	return
}
