package analytics

import (
	"encoding/json"
	"math"
)

type Histogram [BucketCount]int64

func BucketFor(durationMs float64) int {
	if durationMs <= FirstBoundaryMs {
		return 0
	}

	index := int(math.Floor(math.Log2(durationMs/FirstBoundaryMs) * BucketsPerOctave))

	if index < 0 {
		return 0
	}

	if index >= BucketCount {
		return BucketCount - 1
	}

	return index
}

func UpperBoundOf(index int) float64 {
	return FirstBoundaryMs * math.Pow(2, float64(index+1)/BucketsPerOctave)
}

func LowerBoundOf(index int) float64 {
	if index == 0 {
		return 0
	}

	return FirstBoundaryMs * math.Pow(2, float64(index)/BucketsPerOctave)
}

func (self *Histogram) Add(durationMs float64) {
	self[BucketFor(durationMs)]++
}

func (self *Histogram) Merge(other Histogram) {
	for index := range other {
		self[index] += other[index]
	}
}

func (self Histogram) Total() int64 {
	var total int64

	for _, count := range self {
		total += count
	}

	return total
}

func (self Histogram) Percentile(fraction float64) float64 {
	total := self.Total()
	if total == 0 {
		return 0
	}

	wanted := fraction * float64(total)
	var seen int64

	for index, count := range self {
		if count == 0 {
			continue
		}

		if float64(seen+count) >= wanted {
			within := (wanted - float64(seen)) / float64(count)
			lower := LowerBoundOf(index)

			return lower + (UpperBoundOf(index)-lower)*within
		}

		seen += count
	}

	return UpperBoundOf(BucketCount - 1)
}

func (self Histogram) Encode() string {
	encoded, encodeError := json.Marshal(self)
	if encodeError != nil {
		return EmptyHistogram
	}

	return string(encoded)
}

func DecodeHistogram(held string) Histogram {
	histogram := Histogram{}

	if held == "" {
		return histogram
	}

	json.Unmarshal([]byte(held), &histogram)

	return histogram
}
