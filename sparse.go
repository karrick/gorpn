package gorpn

import (
	"math"
	"time"

	"github.com/pkg/errors"
)

// SparseSeries is a possible sparse set of time-value tuples. The purpose of a SpareSeries is to
// receive collection of possibly sparse data tuples, and convert them into a Def, a non-sparse
// collection of tuples, by bucketing values using the requested consolidation function.  There are
// two requirements for proper operation:
//
//   1. Times[i+1] > Times[i]
//   2. Values[i] is the value associated with Times[i]
type SparseSeries struct {
	Label  string
	Times  []time.Time
	Values []float64
}

// NOTE: Using the Avg consolidation function does not make sense for discrete values.

const (
	Avg  = iota // time slice's average value; WARNING: doesn't make sense for discrete values
	Last        // time slice's last value
	Max         // time slice's maximum value
	Min         // time slice's minimum value
)

// Bucket converts a SparseSeries to a Def, bucketing values into slots based on some
// consolidation function. Resulting Def may be empty if SparseSeries does not contain values for
// start and end parameters.
func (s *SparseSeries) Bucket(start, end time.Time, step time.Duration, cf int) (*Def, error) {
	if lt, lv := len(s.Times), len(s.Values); lt != lv {
		return nil, errors.Errorf("cannot bucket with non-matching lengths of Times and Values: %d != %d", lt, lv)
	}

	nan := math.NaN() // likely will need this value a lot
	bucketStart := start.Truncate(step)
	bucketEnd := bucketStart.Add(step)
	t := bucketEnd
	bucketCount := 1

	// NOTE: calculate number of buckets response requires
	if !bucketEnd.After(end) {
		// multiple data points
		t = end.Truncate(step)
		if t.Before(end) {
			t = t.Add(step)
		}
		bucketCount = int((int64(t.UnixNano()-bucketStart.UnixNano()) / int64(step)) + 1)
	}

	def := &Def{
		Label:  s.Label,
		Start:  bucketStart,
		Step:   step,
		Values: make([]float64, bucketCount),
	}
	var di int // destination index within def.Values

	if len(s.Times) > 0 {
		// PRE: t is final bucketEnd
		if !(s.Times[0].After(t) || s.Times[len(s.Times)-1].Before(bucketStart)) {
			var value float64

			// Per-bucket statistics
			var bucketDatumCount, bucketDatumSum float64
			bucketMax := math.Inf(-1)
			bucketMin := math.Inf(1)

			// NOTE: function to calculate and append consolidated value
			emit := func() {
				consolidatedValue := nan
				if bucketDatumCount > 0 { // if at least one non-NaN value in this bucket
					switch cf {
					case Avg:
						consolidatedValue = bucketDatumSum / bucketDatumCount
					case Min:
						consolidatedValue = bucketMin
					case Max:
						consolidatedValue = bucketMax
					case Last:
						consolidatedValue = value
					}
				}
				def.Values[di] = consolidatedValue
				di++
			}

			i, t := binarySearchTimes(bucketStart, s.Times)

			// NOTE: emit NaN values for before first known datum
			for di < bucketCount && bucketStart.Before(t) {
				def.Values[di] = nan
				di++
				bucketStart = bucketEnd
				bucketEnd = bucketStart.Add(step)
			}

			// enumerate through values
			for {
				if value = s.Values[i]; !math.IsNaN(value) {
					// update bucket statistics for non-NaN values
					bucketDatumCount++
					bucketDatumSum += value
					if bucketMax < value {
						bucketMax = value
					}
					if bucketMin > value {
						bucketMin = value
					}
				}

				// advance to next element
				i++
				if i == len(s.Times) {
					break
				}

				t = s.Times[i]
				if t.After(end) {
					break
				}
				if !t.Before(bucketEnd) {
					emit()

					// reset statistics
					bucketDatumCount = 0
					bucketDatumSum = 0
					bucketMax = math.Inf(-1)
					bucketMin = math.Inf(1)

					// advance to next bucket
					bucketStart = bucketEnd
					bucketEnd = bucketStart.Add(step)

					// NOTE: fill in missing NaN values
					for !t.Before(bucketEnd) {
						def.Values[di] = nan
						di++
						bucketStart = bucketEnd
						bucketEnd = bucketStart.Add(step)
					}
				}
			}
			if di < bucketCount {
				emit() // emit final consolidated value
			}
		}
	}

	// NOTE: emit final missing values
	for ; di < bucketCount; di++ {
		def.Values[di] = nan
	}

	return def, nil
}

func binarySearchTimes(key time.Time, times []time.Time) (int, time.Time) {
	var t time.Time
	var i, lo int
	hi := len(times) - 1
	for lo <= hi {
		i = (lo + hi) / 2
		t = times[i]
		if key.Before(t) {
			hi = i - 1
		} else if key.After(t) {
			lo = i + 1
		} else {
			break
		}
	}
	return i, t
}
