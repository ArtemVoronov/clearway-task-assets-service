package services

import (
	"hash/maphash"
)

const BUCKET_NUMBER = 65536
const DEFAULT_BUCKET_FACTOR = 2

type Interval struct {
	From uint64
	To   uint64
}

type ShardService struct {
	buckets    []Interval
	bucketsNum int
	hasher     *maphash.Hash
}

func CreateShardService(bucketFactor int) *ShardService {
	buckets := CreateIntervals(bucketFactor)
	return &ShardService{
		buckets:    buckets,
		bucketsNum: len(buckets),
		hasher:     &maphash.Hash{},
	}
}

func (s *ShardService) Shutdown() error {
	return nil
}

func (s *ShardService) GetBucketIndex(key string) uint64 {
	s.hasher.WriteString(key)
	hash := s.hasher.Sum64()
	bucketIndex := hash % BUCKET_NUMBER
	s.hasher.Reset()
	return bucketIndex
}

func (s *ShardService) GetBucketByIndex(bucketIndex uint64) int {
	for i := 0; i < s.bucketsNum; i++ {
		interval := s.buckets[i]
		if interval.From <= bucketIndex && bucketIndex < interval.To {
			return i
		}
	}
	return 0
}

func (s *ShardService) GetBucketByKey(key string) int {
	bucketIndex := s.GetBucketIndex(key)
	return s.GetBucketByIndex(bucketIndex)
}

func CreateIntervals(bucketFactor int) []Interval {
	result := []Interval{}
	unit := uint64(BUCKET_NUMBER / bucketFactor)
	var from uint64 = 0
	var to uint64 = unit
	for i := 0; i < bucketFactor; i++ {
		result = append(result, Interval{From: from, To: to})
		from += unit
		to += unit
	}

	result[bucketFactor-1].To = BUCKET_NUMBER // for odd numbers

	return result
}
