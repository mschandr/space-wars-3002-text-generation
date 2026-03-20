package stats

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Stats collects aggregate counters across the full run. All methods are safe for
// concurrent use from multiple worker goroutines.
type Stats struct {
	mu sync.Mutex

	VendorsProcessed int
	VendorsFailed    int
	BucketsCompleted int
	BucketsFailed    int
	LinesInserted    int
	RetryAttempts    int
	LinesByType      map[string]int
}

func New() *Stats {
	return &Stats{
		LinesByType: make(map[string]int),
	}
}

func (s *Stats) RecordVendorComplete() {
	s.mu.Lock()
	s.VendorsProcessed++
	s.mu.Unlock()
}

func (s *Stats) RecordVendorFailed() {
	s.mu.Lock()
	s.VendorsProcessed++
	s.VendorsFailed++
	s.mu.Unlock()
}

// RecordBucketComplete records a successful bucket. retries is the attempt index
// when success was reached (0 = first try, 1 = one retry, etc.).
func (s *Stats) RecordBucketComplete(lineType string, linesInserted, retries int) {
	s.mu.Lock()
	s.BucketsCompleted++
	s.LinesInserted += linesInserted
	s.RetryAttempts += retries
	s.LinesByType[lineType] += linesInserted
	s.mu.Unlock()
}

func (s *Stats) RecordBucketFailed(lineType string) {
	s.mu.Lock()
	s.BucketsFailed++
	s.mu.Unlock()
}

// Print writes a human-readable run summary to stdout.
func (s *Stats) Print() {
	s.mu.Lock()
	defer s.mu.Unlock()

	vendorsComplete := s.VendorsProcessed - s.VendorsFailed
	bucketsAttempted := s.BucketsCompleted + s.BucketsFailed

	sep := strings.Repeat("=", 52)
	fmt.Println()
	fmt.Println(sep)
	fmt.Println("                   Run Summary")
	fmt.Println(sep)
	fmt.Printf("  Vendors : %d processed  |  %d complete  |  %d failed\n",
		s.VendorsProcessed, vendorsComplete, s.VendorsFailed)
	fmt.Printf("  Buckets : %d attempted  |  %d complete  |  %d failed\n",
		bucketsAttempted, s.BucketsCompleted, s.BucketsFailed)
	fmt.Printf("  Lines   : %d inserted\n", s.LinesInserted)
	fmt.Printf("  Retries : %d total retry attempts\n", s.RetryAttempts)

	if len(s.LinesByType) > 0 {
		fmt.Println()
		fmt.Println("  Lines by type:")

		// Sort for consistent output order
		types := make([]string, 0, len(s.LinesByType))
		for t := range s.LinesByType {
			types = append(types, t)
		}
		sort.Strings(types)

		for _, t := range types {
			fmt.Printf("    %-24s %d\n", t, s.LinesByType[t])
		}
	}

	fmt.Println(sep)
}
