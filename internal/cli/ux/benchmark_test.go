package ux

import (
	"fmt"
	"testing"
)

func BenchmarkClassifyError(b *testing.B) {
	err := fmt.Errorf("connection refused by remote host")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ClassifyError(err)
	}
}

func BenchmarkClassifyError_Unknown(b *testing.B) {
	err := fmt.Errorf("something happened")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ClassifyError(err)
	}
}

func BenchmarkCalculateDelay_Exponential(b *testing.B) {
	s := DefaultRetryStrategy()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calculateDelay(s, i%10)
	}
}

func BenchmarkContainsAny(b *testing.B) {
	substrings := []string{"connection refused", "timeout", "network unreachable"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		containsAny("some error connection refused", substrings)
	}
}
