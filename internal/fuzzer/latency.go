package fuzzer

import (
	"math"
	"math/bits"
	"time"
)

// latencyBuckets covers 0 µs to about 12 days, far past any request timeout.
const latencyBuckets = 320

// Latency summarizes the response times of one target. It keeps a histogram
// instead of every sample, so its memory doesn't grow with the requests. Min,
// Max and Mean are exact; percentiles are off by about 6% at most, since each
// bucket spans an eighth of a power of two.
type Latency struct {
	Count    int
	Min, Max time.Duration
	total    time.Duration
	buckets  *[latencyBuckets]uint32 // nil until the first sample
}

func (l *Latency) add(d time.Duration) {
	if l.buckets == nil {
		l.buckets = new([latencyBuckets]uint32)
		l.Min = d
	}
	l.Count++
	l.total += d
	l.Min = min(l.Min, d)
	l.Max = max(l.Max, d)
	l.buckets[latencyBucket(d)]++
}

// Mean returns the average response time, or 0 without samples.
func (l *Latency) Mean() time.Duration {
	if l.Count == 0 {
		return 0
	}
	return l.total / time.Duration(l.Count)
}

// Percentile returns the response time p percent of the samples don't exceed,
// or 0 without samples.
func (l *Latency) Percentile(p float64) time.Duration {
	if l.Count == 0 {
		return 0
	}
	rank := max(uint64(math.Ceil(float64(l.Count)*p/100)), 1)
	var seen uint64
	for i, count := range l.buckets {
		if seen += uint64(count); seen >= rank {
			return min(max(latencyBucketEnd(i), l.Min), l.Max)
		}
	}
	return l.Max
}

// latencyBucket keeps the top four bits of the time in microseconds: exact below
// 16 µs, then eight buckets per power of two.
func latencyBucket(d time.Duration) int {
	us := uint64(max(d.Microseconds(), 0))
	if us < 16 {
		return int(us)
	}
	shift := bits.Len64(us) - 4
	return min(shift*8+int(us>>shift), latencyBuckets-1)
}

// latencyBucketEnd returns the largest time of a bucket.
func latencyBucketEnd(i int) time.Duration {
	if i < 16 {
		return time.Duration(i) * time.Microsecond
	}
	shift := i/8 - 1
	top := uint64(i%8 + 8)
	return time.Duration((top+1)<<shift-1) * time.Microsecond
}
