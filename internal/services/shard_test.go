package services

import (
	"testing"
)

func TestHashConstancy(t *testing.T) {
	testShardFactor := 4
	original := CreateShardService(testShardFactor)
	testKey := "1F615C1D-6BAE-4D8F-EF0B-2FCDC247EF69"
	expectedHash := original.hash(testKey)
	expectedIndex := original.GetBucketIndex(testKey)
	for i := 0; i < 1; i++ {
		generationP := CreateShardService(testShardFactor)
		actualHash := generationP.hash(testKey)
		actualIndex := original.GetBucketIndex(testKey)
		if actualHash != expectedHash {
			t.Errorf("expected hash: %v, actual hash: %v\n", expectedHash, actualHash)
		}
		if actualIndex != expectedIndex {
			t.Errorf("expected bucket index: %v, actual bucket index: %v\n", expectedIndex, actualIndex)
		}
	}
}
