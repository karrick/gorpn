package gorpn

import (
	"math"
	"testing"
	"time"
)

func epoch(epoch int64) time.Time {
	return time.Unix(epoch, 0).UTC()
}

func TestSparseSeriesBucketNoSourceData(t *testing.T) {
	s := &SparseSeries{
		Label: "t1",
	}

	startTime := epoch(60)
	endTime := epoch(62)
	step := time.Second

	def, err := s.Bucket(startTime, endTime, step, Last)
	if err != nil {
		t.Fatal(err)
	}

	if actual, expected := len(def.Values), 3; actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
		t.Log(def)
	}
	if actual := def.Values[0]; !math.IsNaN(actual) {
		t.Errorf("Actual: %#v; Expected: %#v", actual, math.NaN())
	}
	if actual := def.Values[1]; !math.IsNaN(actual) {
		t.Errorf("Actual: %#v; Expected: %#v", actual, math.NaN())
	}
	if actual := def.Values[2]; !math.IsNaN(actual) {
		t.Errorf("Actual: %#v; Expected: %#v", actual, math.NaN())
	}
}

func TestSparseSeriesBucketRequestAllBefore(t *testing.T) {
	s := &SparseSeries{
		Label:  "t1",
		Times:  []time.Time{epoch(60), epoch(61), epoch(62)},
		Values: []float64{13, 42, 99},
	}

	startTime := epoch(50)
	endTime := epoch(53)
	step := time.Second

	def, err := s.Bucket(startTime, endTime, step, Last)
	if err != nil {
		t.Fatal(err)
	}

	// Expect:
	//   50  51  52  53
	//   NaN NaN NaN NaN

	if actual, expected := len(def.Values), 4; actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
		t.Log(def)
	}
	if actual := def.Values[0]; !math.IsNaN(actual) {
		t.Errorf("Actual: %#v; Expected: %#v", actual, math.NaN())
	}
	if actual := def.Values[1]; !math.IsNaN(actual) {
		t.Errorf("Actual: %#v; Expected: %#v", actual, math.NaN())
	}
	if actual := def.Values[2]; !math.IsNaN(actual) {
		t.Errorf("Actual: %#v; Expected: %#v", actual, math.NaN())
	}
	if actual := def.Values[3]; !math.IsNaN(actual) {
		t.Errorf("Actual: %#v; Expected: %#v", actual, math.NaN())
	}
}

func TestSparseSeriesBucketRequestAllAfter(t *testing.T) {
	s := &SparseSeries{
		Label:  "t1",
		Times:  []time.Time{epoch(60), epoch(61), epoch(62)},
		Values: []float64{13, 42, 99},
	}

	startTime := epoch(120)
	endTime := epoch(123)
	step := time.Second

	def, err := s.Bucket(startTime, endTime, step, Last)
	if err != nil {
		t.Fatal(err)
	}

	// Expect:
	//   120 121 122 123
	//   NaN NaN NaN NaN

	if actual, expected := len(def.Values), 4; actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
		t.Log(def)
	}
	if actual := def.Values[0]; !math.IsNaN(actual) {
		t.Errorf("Actual: %#v; Expected: %#v", actual, math.NaN())
	}
	if actual := def.Values[1]; !math.IsNaN(actual) {
		t.Errorf("Actual: %#v; Expected: %#v", actual, math.NaN())
	}
	if actual := def.Values[2]; !math.IsNaN(actual) {
		t.Errorf("Actual: %#v; Expected: %#v", actual, math.NaN())
	}
	if actual := def.Values[3]; !math.IsNaN(actual) {
		t.Errorf("Actual: %#v; Expected: %#v", actual, math.NaN())
	}
}

func TestSparseSeriesBucketRequestSinglePointKnown(t *testing.T) {
	s := &SparseSeries{
		Label:  "t1",
		Times:  []time.Time{epoch(60), epoch(61), epoch(62)},
		Values: []float64{13, 42, 99},
	}

	startTime := epoch(61)
	endTime := epoch(61)
	step := time.Second

	def, err := s.Bucket(startTime, endTime, step, Last)
	if err != nil {
		t.Fatal(err)
	}

	// Expect:
	//   61
	//   42

	if actual, expected := len(def.Values), 1; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
		t.Log(def)
	}
	if actual, expected := def.Values[0], float64(42); actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestSparseSeriesBucketRequestSinglePointUnknown(t *testing.T) {
	s := &SparseSeries{
		Label:  "t1",
		Times:  []time.Time{epoch(50), epoch(70)},
		Values: []float64{13, 42},
	}

	startTime := epoch(60)
	endTime := epoch(60)
	step := time.Second

	def, err := s.Bucket(startTime, endTime, step, Last)
	if err != nil {
		t.Fatal(err)
	}

	// Expect:
	//   60
	//   NaN

	if actual, expected := len(def.Values), 1; actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
		t.Log(def)
	}
	if actual := def.Values[0]; !math.IsNaN(actual) {
		t.Errorf("Actual: %#v; Expected: %#v", actual, math.NaN())
	}
}

func TestSparseSeriesBucketLabelStartStep(t *testing.T) {
	s := &SparseSeries{
		Label:  "t1",
		Times:  []time.Time{epoch(60), epoch(61), epoch(62)},
		Values: []float64{13, 42, 99},
	}

	startTime := epoch(60)
	endTime := epoch(62)
	step := time.Second

	def, err := s.Bucket(startTime, endTime, step, Last)
	if err != nil {
		t.Fatal(err)
	}
	if actual, expected := def.Label, s.Label; actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
		t.Log(def)
	}
	if actual, expected := def.Start, startTime; !actual.Equal(expected) {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := def.Step, step; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestSparseSeriesBucketBeforeNoNaN(t *testing.T) {
	s := &SparseSeries{
		Label:  "t1",
		Times:  []time.Time{epoch(60), epoch(61), epoch(62)},
		Values: []float64{13, 42, 99},
	}

	startTime := epoch(60)
	endTime := epoch(62)
	step := time.Second

	def, err := s.Bucket(startTime, endTime, step, Last)
	if err != nil {
		t.Fatal(err)
	}

	// Expect:
	//   60  61  62
	//   13  42  99

	if actual, expected := len(def.Values), 3; actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
		t.Log(def)
	}
	if actual, expected := def.Values[0], float64(13); actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := def.Values[1], float64(42); actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := def.Values[2], float64(99); actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestSparseSeriesBucketBeforeOneNaN(t *testing.T) {
	s := &SparseSeries{
		Label:  "t1",
		Times:  []time.Time{epoch(60), epoch(61)},
		Values: []float64{13, 42},
	}

	startTime := epoch(59)
	endTime := epoch(61)
	step := time.Second

	def, err := s.Bucket(startTime, endTime, step, Last)
	if err != nil {
		t.Fatal(err)
	}

	// Expect:
	//   59  60  61
	//   NaN 13  42

	if actual, expected := len(def.Values), 3; actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
		t.Log(def)
	}
	if actual := def.Values[0]; !math.IsNaN(actual) {
		t.Errorf("Actual: %#v; Expected: %#v", actual, math.NaN())
	}
	if actual, expected := def.Values[1], float64(13); actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := def.Values[2], float64(42); actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestSparseSeriesBucketBeforeTwoNaN(t *testing.T) {
	s := &SparseSeries{
		Label:  "t1",
		Times:  []time.Time{epoch(60), epoch(61)},
		Values: []float64{13, 42},
	}

	startTime := epoch(58)
	endTime := epoch(61)
	step := time.Second

	def, err := s.Bucket(startTime, endTime, step, Last)
	if err != nil {
		t.Fatal(err)
	}

	// Expect:
	//   58  59  60  61
	//   NaN NaN 13  42

	if actual, expected := len(def.Values), 4; actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
		t.Log(def)
	}
	if actual := def.Values[0]; !math.IsNaN(actual) {
		t.Errorf("Actual: %#v; Expected: %#v", actual, math.NaN())
	}
	if actual := def.Values[1]; !math.IsNaN(actual) {
		t.Errorf("Actual: %#v; Expected: %#v", actual, math.NaN())
	}
	if actual, expected := def.Values[2], float64(13); actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := def.Values[3], float64(42); actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestSparseSeriesBucketMiddleOneNaN(t *testing.T) {
	s := &SparseSeries{
		Label:  "t1",
		Times:  []time.Time{epoch(60), epoch(62)},
		Values: []float64{13, 42},
	}

	startTime := epoch(60)
	endTime := epoch(62)
	step := time.Second

	def, err := s.Bucket(startTime, endTime, step, Last)
	if err != nil {
		t.Fatal(err)
	}

	// Expect:
	//   60  61  62
	//   13  NaN 42

	if actual, expected := len(def.Values), 3; actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
		t.Log(def)
	}
	if actual, expected := def.Values[0], float64(13); actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual := def.Values[1]; !math.IsNaN(actual) {
		t.Errorf("Actual: %#v; Expected: %#v", actual, math.NaN())
	}
	if actual, expected := def.Values[2], float64(42); actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestSparseSeriesBucketMiddleTwoNaN(t *testing.T) {
	s := &SparseSeries{
		Label:  "t1",
		Times:  []time.Time{epoch(60), epoch(63)},
		Values: []float64{13, 42},
	}

	startTime := epoch(60)
	endTime := epoch(63)
	step := time.Second

	def, err := s.Bucket(startTime, endTime, step, Last)
	if err != nil {
		t.Fatal(err)
	}

	// Expect:
	//   60  61  62  63
	//   13  NaN NaN 42

	if actual, expected := len(def.Values), 4; actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
		t.Log(def)
	}
	if actual, expected := def.Values[0], float64(13); actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual := def.Values[1]; !math.IsNaN(actual) {
		t.Errorf("Actual: %#v; Expected: %#v", actual, math.NaN())
	}
	if actual := def.Values[2]; !math.IsNaN(actual) {
		t.Errorf("Actual: %#v; Expected: %#v", actual, math.NaN())
	}
	if actual, expected := def.Values[3], float64(42); actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestSparseSeriesBucketAfterOneNaN(t *testing.T) {
	s := &SparseSeries{
		Label:  "t1",
		Times:  []time.Time{epoch(60), epoch(61)},
		Values: []float64{13, 42},
	}

	startTime := epoch(60)
	endTime := epoch(62)
	step := time.Second

	def, err := s.Bucket(startTime, endTime, step, Last)
	if err != nil {
		t.Fatal(err)
	}

	// Expect:
	//   60  61  62
	//   13  42  NaN

	if actual, expected := len(def.Values), 3; actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
		t.Log(def)
	}
	if actual, expected := def.Values[0], float64(13); actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := def.Values[1], float64(42); actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual := def.Values[2]; !math.IsNaN(actual) {
		t.Errorf("Actual: %#v; Expected: %#v", actual, math.NaN())
	}
}

func TestSparseSeriesBucketAfterTwoNaN(t *testing.T) {
	s := &SparseSeries{
		Label:  "t1",
		Times:  []time.Time{epoch(60), epoch(61)},
		Values: []float64{13, 42},
	}

	startTime := epoch(60)
	endTime := epoch(63)
	step := time.Second

	def, err := s.Bucket(startTime, endTime, step, Last)
	if err != nil {
		t.Fatal(err)
	}

	// Expect:
	//   60  61  62  63
	//   13  42  NaN NaN

	if actual, expected := len(def.Values), 4; actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
		t.Log(def)
	}
	if actual, expected := def.Values[0], float64(13); actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual, expected := def.Values[1], float64(42); actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
	if actual := def.Values[2]; !math.IsNaN(actual) {
		t.Errorf("Actual: %#v; Expected: %#v", actual, math.NaN())
	}
	if actual := def.Values[3]; !math.IsNaN(actual) {
		t.Errorf("Actual: %#v; Expected: %#v", actual, math.NaN())
	}
}

func TestSparseSeriesBucketMaxAllNegInf(t *testing.T) {
	s := &SparseSeries{
		Label:  "t1",
		Times:  []time.Time{epoch(60), epoch(61), epoch(62)},
		Values: []float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)},
	}

	startTime := epoch(60)
	endTime := epoch(69)
	step := 10 * time.Second

	def, err := s.Bucket(startTime, endTime, step, Last)
	if err != nil {
		t.Fatal(err)
	}

	// Expect:
	//   60   61  62  63
	//   -Inf NaN

	if actual, expected := len(def.Values), 1; actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
		t.Log(def)
	}
	if actual, expected := def.Values[0], math.Inf(-1); actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestSparseSeriesBucketMinAllInf(t *testing.T) {
	s := &SparseSeries{
		Label:  "t1",
		Times:  []time.Time{epoch(60), epoch(61), epoch(62)},
		Values: []float64{math.Inf(1), math.Inf(1), math.Inf(1)},
	}

	startTime := epoch(60)
	endTime := epoch(69)
	step := 10 * time.Second

	def, err := s.Bucket(startTime, endTime, step, Last)
	if err != nil {
		t.Fatal(err)
	}

	// Expect:
	//   60  61  62  63
	//   Inf NaN

	if actual, expected := len(def.Values), 1; actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
		t.Log(def)
	}
	if actual, expected := def.Values[0], math.Inf(1); actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestSparseSeriesBucketAvg(t *testing.T) {
	s := &SparseSeries{
		Label:  "t1",
		Times:  []time.Time{epoch(60), epoch(61), epoch(62)},
		Values: []float64{13, math.NaN(), 42},
	}

	startTime := epoch(60)
	endTime := epoch(69)
	step := 10 * time.Second

	def, err := s.Bucket(startTime, endTime, step, Avg)
	if err != nil {
		t.Fatal(err)
	}

	// Expect:
	//   60
	//   27.5

	if actual, expected := len(def.Values), 1; actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
		t.Log(def)
	}
	if actual, expected := def.Values[0], 27.5; actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestSparseSeriesBucketLast(t *testing.T) {
	s := &SparseSeries{
		Label:  "t1",
		Times:  []time.Time{epoch(60), epoch(61), epoch(62)},
		Values: []float64{13, math.NaN(), 42},
	}

	startTime := epoch(60)
	endTime := epoch(69)
	step := 10 * time.Second

	def, err := s.Bucket(startTime, endTime, step, Last)
	if err != nil {
		t.Fatal(err)
	}

	// Expect:
	//   60
	//   42

	if actual, expected := len(def.Values), 1; actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
		t.Log(def)
	}
	if actual, expected := def.Values[0], float64(42); actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestSparseSeriesBucketMin(t *testing.T) {
	s := &SparseSeries{
		Label:  "t1",
		Times:  []time.Time{epoch(60), epoch(61), epoch(62)},
		Values: []float64{13, math.NaN(), 42},
	}

	startTime := epoch(60)
	endTime := epoch(69)
	step := 10 * time.Second

	def, err := s.Bucket(startTime, endTime, step, Min)
	if err != nil {
		t.Fatal(err)
	}

	// Expect:
	//   60
	//   13

	if actual, expected := len(def.Values), 1; actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
		t.Log(def)
	}
	if actual, expected := def.Values[0], float64(13); actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
}

func TestSparseSeriesBucketMax(t *testing.T) {
	s := &SparseSeries{
		Label:  "t1",
		Times:  []time.Time{epoch(60), epoch(61), epoch(62)},
		Values: []float64{13, math.NaN(), 42},
	}

	startTime := epoch(60)
	endTime := epoch(69)
	step := 10 * time.Second

	def, err := s.Bucket(startTime, endTime, step, Max)
	if err != nil {
		t.Fatal(err)
	}

	// Expect:
	//   60
	//   42

	if actual, expected := len(def.Values), 1; actual != expected {
		t.Fatalf("Actual: %#v; Expected: %#v", actual, expected)
		t.Log(def)
	}
	if actual, expected := def.Values[0], float64(42); actual != expected {
		t.Errorf("Actual: %#v; Expected: %#v", actual, expected)
	}
}
