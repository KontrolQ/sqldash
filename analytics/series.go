package analytics

import (
	"time"
)

type Point struct {
	At       time.Time
	Total    int64
	Reads    int64
	Writes   int64
	Failures int64
	Bytes    int64
	P50      float64
	P95      float64
	P99      float64
}

func SeriesFor(databaseName string, window Window) ([]Point, error) {
	records, findError := rollupsIn(databaseName, window)
	if findError != nil {
		return nil, findError
	}

	ends := time.Now().UTC()
	starts := ends.Add(-window.Since)
	width := window.Since / SeriesSlots

	slots := make([]Summary, SeriesSlots)

	for _, record := range records {
		offset := record.BucketStart.UTC().Sub(starts)
		if offset < 0 {
			continue
		}

		index := int(offset / width)
		if index < 0 || index >= SeriesSlots {
			continue
		}

		slots[index].absorb(record)
	}

	points := make([]Point, 0, SeriesSlots)

	for index := range slots {
		summary := slots[index]

		points = append(points, Point{
			At:       starts.Add(time.Duration(index) * width),
			Total:    summary.Count,
			Writes:   summary.Writes,
			Reads:    summary.Count - summary.Writes,
			Failures: summary.Failures,
			Bytes:    summary.BytesIn + summary.BytesOut,
			P50:      summary.Histogram.Percentile(0.50),
			P95:      summary.Histogram.Percentile(0.95),
			P99:      summary.Histogram.Percentile(0.99),
		})
	}

	return points, nil
}
