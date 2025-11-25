/*
Package performance provides comprehensive performance testing for R2Go2

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package performance

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	r2api "github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/api"
	"github.com/CosmoLabs-org/CosmoDev-R2Go2/tests/helpers"
)

// AdvancedPerformanceTestSuite provides comprehensive performance testing
type AdvancedPerformanceTestSuite struct {
	suite.Suite
	testConfig helpers.TestConfig
}

// SetupSuite sets up the advanced performance test suite
func (suite *AdvancedPerformanceTestSuite) SetupSuite() {
	// Create test configuration
	suite.testConfig = *helpers.SetupTest(suite.T())
}

// PerformanceMetrics holds performance measurement data
type PerformanceMetrics struct {
	OperationCount    int64
	TotalDuration     time.Duration
	MinDuration       time.Duration
	MaxDuration       time.Duration
	AvgDuration       time.Duration
	ThroughputQPS     float64
	MemoryUsageMB     float64
	SuccessCount      int64
	ErrorCount        int64
	ErrorRate         float64
	P95Duration       time.Duration
	P99Duration       time.Duration
}

// LoadTestResult holds load test results
type LoadTestResult struct {
	Metrics          PerformanceMetrics
	ConcurrentUsers  int
	TestDuration     time.Duration
	TargetThroughput float64
	AchievedGoal     bool
}

// TestConcurrentOperations tests performance under concurrent load
func (suite *AdvancedPerformanceTestSuite) TestConcurrentOperations() {
	suite.Run("Client Creation Performance", func() {
		const iterations = 1000
		const concurrentWorkers = 50

		var wg sync.WaitGroup
		results := make(chan time.Duration, iterations)

		start := time.Now()

		for i := 0; i < concurrentWorkers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < iterations/concurrentWorkers; j++ {
					opStart := time.Now()

					// Create R2 API client
					opts := &r2api.ClientOptions{
						AccountID: "test-account",
						APIToken:  "test-token",
					}
					client, err := r2api.NewClient(opts)

					require.NoError(suite.T(), err, "Client creation should not fail")
					assert.NotNil(suite.T(), client, "Client should not be nil")

					results <- time.Since(opStart)
				}
			}()
		}

		wg.Wait()
		close(results)

		totalDuration := time.Since(start)

		// Calculate metrics
		var totalOpTime time.Duration
		var minTime, maxTime time.Duration = time.Hour, 0
		opTimes := make([]time.Duration, 0, iterations)

		for opTime := range results {
			totalOpTime += opTime
			opTimes = append(opTimes, opTime)
			if opTime < minTime {
				minTime = opTime
			}
			if opTime > maxTime {
				maxTime = opTime
			}
		}

		metrics := PerformanceMetrics{
			OperationCount: int64(iterations),
			TotalDuration:   totalDuration,
			MinDuration:     minTime,
			MaxDuration:     maxTime,
			AvgDuration:     totalOpTime / time.Duration(iterations),
			ThroughputQPS:   float64(iterations) / totalDuration.Seconds(),
			SuccessCount:    int64(iterations),
		}

		// Calculate percentiles
		if len(opTimes) > 0 {
			metrics.P95Duration = calculatePercentile(opTimes, 0.95)
			metrics.P99Duration = calculatePercentile(opTimes, 0.99)
		}

		// Performance assertions
		assert.Less(suite.T(), metrics.AvgDuration, 10*time.Millisecond, "Average client creation should be fast")
		assert.Greater(suite.T(), metrics.ThroughputQPS, 100.0, "Should achieve high throughput")
		assert.Less(suite.T(), metrics.P95Duration, 50*time.Millisecond, "95th percentile should be reasonable")

		// Log performance metrics
		suite.T().Logf("Client Creation Performance:")
		suite.T().Logf("  Operations: %d", metrics.OperationCount)
		suite.T().Logf("  Throughput: %.2f ops/sec", metrics.ThroughputQPS)
		suite.T().Logf("  Avg Duration: %v", metrics.AvgDuration)
		suite.T().Logf("  P95 Duration: %v", metrics.P95Duration)
		suite.T().Logf("  P99 Duration: %v", metrics.P99Duration)
	})

	suite.Run("Memory Efficiency Test", func() {
		// Test memory usage with many clients
		const clientCount = 1000

		runtime.GC()
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)

		clients := make([]*r2api.Client, clientCount)
		for i := 0; i < clientCount; i++ {
			opts := &r2api.ClientOptions{
				AccountID: fmt.Sprintf("account-%d", i),
				APIToken:  fmt.Sprintf("token-%d", i),
			}
			client, err := r2api.NewClient(opts)
			require.NoError(suite.T(), err)
			clients[i] = client
		}

		runtime.GC()
		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)

		memoryUsedMB := float64(m2.Alloc-m1.Alloc) / 1024 / 1024
		memoryPerClientKB := float64(m2.Alloc-m1.Alloc) / float64(clientCount) / 1024

		suite.T().Logf("Memory Usage for %d clients:", clientCount)
		suite.T().Logf("  Total Memory: %.2f MB", memoryUsedMB)
		suite.T().Logf("  Memory per Client: %.2f KB", memoryPerClientKB)

		// Memory efficiency assertions
		assert.Less(suite.T(), memoryPerClientKB, 10.0, "Each client should use minimal memory")
		assert.Less(suite.T(), memoryUsedMB, 50.0, "Total memory usage should be reasonable")

		// Test client operations
		for i, client := range clients {
			assert.Equal(suite.T(), fmt.Sprintf("account-%d", i), client.GetAccountID())
		}
	})
}

// TestThroughputBenchmarks tests throughput under various conditions
func (suite *AdvancedPerformanceTestSuite) TestThroughputBenchmarks() {
	suite.Run("High Frequency Operations", func() {
		const operations = 10000
		const duration = 10 * time.Second

		ctx, cancel := context.WithTimeout(context.Background(), duration)
		defer cancel()

		var (
			successCount int64
			errorCount   int64
			opCount      int64
			totalTime    time.Duration
			mu           sync.Mutex
		)

		start := time.Now()

		// Worker pool
		const workers = 10
		var wg sync.WaitGroup
		opsPerWorker := operations / workers

		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				for j := 0; j < opsPerWorker; j++ {
					select {
					case <-ctx.Done():
						return
					default:
						opStart := time.Now()

						// Simulate operation
						opts := &r2api.ClientOptions{
							AccountID: "benchmark-account",
							APIToken:  "benchmark-token",
						}
						_, err := r2api.NewClient(opts)

						opDuration := time.Since(opStart)

						mu.Lock()
						opCount++
						totalTime += opDuration
						if err != nil {
							errorCount++
						} else {
							successCount++
						}
						mu.Unlock()
					}
				}
			}()
		}

		wg.Wait()
		actualDuration := time.Since(start)

		mu.Lock()
		metrics := PerformanceMetrics{
			OperationCount: opCount,
			TotalDuration:   actualDuration,
			AvgDuration:     totalTime / time.Duration(opCount),
			ThroughputQPS:   float64(opCount) / actualDuration.Seconds(),
			SuccessCount:    successCount,
			ErrorCount:      errorCount,
			ErrorRate:       float64(errorCount) / float64(opCount) * 100,
		}
		mu.Unlock()

		suite.T().Logf("High Frequency Operations Results:")
		suite.T().Logf("  Operations: %d", metrics.OperationCount)
		suite.T().Logf("  Duration: %v", metrics.TotalDuration)
		suite.T().Logf("  Throughput: %.2f ops/sec", metrics.ThroughputQPS)
		suite.T().Logf("  Success Rate: %.2f%%", 100-metrics.ErrorRate)
		suite.T().Logf("  Avg Operation Time: %v", metrics.AvgDuration)

		// Performance assertions
		assert.Greater(suite.T(), metrics.ThroughputQPS, 1000.0, "Should achieve high throughput")
		assert.Less(suite.T(), metrics.ErrorRate, 1.0, "Error rate should be very low")
		assert.Less(suite.T(), metrics.AvgDuration, 5*time.Millisecond, "Average operation should be fast")
	})

	suite.Run("Sustained Load Test", func() {
		const testDuration = 30 * time.Second
		const targetQPS = 100.0

		ctx, cancel := context.WithTimeout(context.Background(), testDuration)
		defer cancel()

		var (
			opsCompleted int64
			totalTime    time.Duration
			mu           sync.Mutex
		)

		start := time.Now()
		ticker := time.NewTicker(time.Duration(float64(time.Second) / targetQPS))
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				goto done
			case <-ticker.C:
				go func() {
					opStart := time.Now()

					// Simulate operation
					opts := &r2api.ClientOptions{
						AccountID: "load-test-account",
						APIToken:  "load-test-token",
					}
					client, err := r2api.NewClient(opts)

					opDuration := time.Since(opStart)

					mu.Lock()
					opsCompleted++
					totalTime += opDuration
					if err == nil {
						assert.NotNil(suite.T(), client)
					}
					mu.Unlock()
				}()
			}
		}

	done:
		actualDuration := time.Since(start)

		mu.Lock()
		achievedQPS := float64(opsCompleted) / actualDuration.Seconds()
		avgOpTime := totalTime / time.Duration(opsCompleted)
		mu.Unlock()

		suite.T().Logf("Sustained Load Test Results:")
		suite.T().Logf("  Duration: %v", actualDuration)
		suite.T().Logf("  Target QPS: %.2f", targetQPS)
		suite.T().Logf("  Achieved QPS: %.2f", achievedQPS)
		suite.T().Logf("  Operations: %d", opsCompleted)
		suite.T().Logf("  Avg Operation Time: %v", avgOpTime)

		// Performance assertions
		assert.GreaterOrEqual(suite.T(), achievedQPS, targetQPS*0.95, "Should achieve target throughput within 5%")
		assert.Less(suite.T(), avgOpTime, 50*time.Millisecond, "Average operation time should be reasonable")
	})
}

// TestScalabilityLimits tests system behavior under stress
func (suite *AdvancedPerformanceTestSuite) TestScalabilityLimits() {
	suite.Run("Connection Pool Stress Test", func() {
		const maxConnections = 1000
		const batchCount = 10

		results := make([]LoadTestResult, batchCount)

		for batch := 0; batch < batchCount; batch++ {
			connectionCount := (batch + 1) * (maxConnections / batchCount)

			start := time.Now()

			// Create connections concurrently
			var wg sync.WaitGroup
			connections := make([]*r2api.Client, connectionCount)
			errors := make(chan error, connectionCount)

			for i := 0; i < connectionCount; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()

					opts := &r2api.ClientOptions{
						AccountID: fmt.Sprintf("stress-test-%d", id),
						APIToken:  fmt.Sprintf("stress-token-%d", id),
					}

					client, err := r2api.NewClient(opts)
					connections[id] = client
					errors <- err
				}(i)
			}

			wg.Wait()
			close(errors)

			duration := time.Since(start)

			// Count errors
			errorCount := 0
			for err := range errors {
				if err != nil {
					errorCount++
				}
			}

			successCount := int64(connectionCount - errorCount)
			successRate := float64(successCount) / float64(connectionCount) * 100
			throughput := float64(connectionCount) / duration.Seconds()

			results[batch] = LoadTestResult{
				Metrics: PerformanceMetrics{
					OperationCount: int64(connectionCount),
					TotalDuration:   duration,
					ThroughputQPS:   throughput,
					SuccessCount:    successCount,
					ErrorCount:      int64(errorCount),
					ErrorRate:       float64(errorCount) / float64(connectionCount) * 100,
				},
				ConcurrentUsers: connectionCount,
				TestDuration:    duration,
			}

			suite.T().Logf("Batch %d (%d connections):", batch+1, connectionCount)
			suite.T().Logf("  Duration: %v", duration)
			suite.T().Logf("  Success Rate: %.2f%%", successRate)
			suite.T().Logf("  Throughput: %.2f conn/sec", throughput)

			// Verify connections were created successfully
			if errorCount > 0 {
				suite.T().Logf("  Errors: %d", errorCount)
			}

			// Test connection functionality
			for i, client := range connections {
				if client != nil {
					assert.NotEmpty(suite.T(), client.GetAccountID(), "Client should have account ID")
					assert.Contains(suite.T(), client.GetAccountID(), "stress-test", "Should have expected account ID")
				} else {
					suite.T().Logf("Connection %d failed to create", i)
				}
			}
		}

		// Analyze scalability trends
		for i, result := range results {
			if i > 0 {
				prevResult := results[i-1]
				throughputRatio := result.Metrics.ThroughputQPS / prevResult.Metrics.ThroughputQPS
				connectionRatio := float64(result.ConcurrentUsers) / float64(prevResult.ConcurrentUsers)

				suite.T().Logf("Scalability Batch %d: Throughput ratio: %.2f, Connection ratio: %.2f",
					i+1, throughputRatio, connectionRatio)

				// Throughput should scale reasonably with connections
				assert.Greater(suite.T(), throughputRatio, 0.5, "Throughput should scale reasonably")
			}
		}
	})

	suite.Run("Resource Utilization Test", func() {
		const testDuration = 10 * time.Second
		const workerCount = 50
		const operationsPerWorker = 100

		ctx, cancel := context.WithTimeout(context.Background(), testDuration)
		defer cancel()

		// Measure initial memory
		runtime.GC()
		var initialMem runtime.MemStats
		runtime.ReadMemStats(&initialMem)

		var wg sync.WaitGroup
		operationTimes := make(chan time.Duration, workerCount*operationsPerWorker)

		start := time.Now()

		for i := 0; i < workerCount; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for j := 0; j < operationsPerWorker; j++ {
					select {
					case <-ctx.Done():
						return
					default:
						opStart := time.Now()

						// Resource-intensive operation
						opts := &r2api.ClientOptions{
							AccountID: fmt.Sprintf("resource-test-%d-%d", workerID, j),
							APIToken:  fmt.Sprintf("resource-token-%d-%d", workerID, j),
						}
						client, err := r2api.NewClient(opts)

						if err == nil {
							_ = client.GetAccountID() // Access the client
						}

						operationTimes <- time.Since(opStart)
					}
				}
			}(i)
		}

		wg.Wait()
		close(operationTimes)

		actualDuration := time.Since(start)

		// Measure final memory
		runtime.GC()
		var finalMem runtime.MemStats
		runtime.ReadMemStats(&finalMem)

		// Calculate metrics
		var totalOpTime time.Duration
		var minTime, maxTime time.Duration = time.Hour, 0
		opCount := 0

		for opTime := range operationTimes {
			totalOpTime += opTime
			opCount++
			if opTime < minTime {
				minTime = opTime
			}
			if opTime > maxTime {
				maxTime = opTime
			}
		}

		memoryUsedMB := float64(finalMem.Alloc-initialMem.Alloc) / 1024 / 1024
		memoryPerOpKB := float64(finalMem.Alloc-initialMem.Alloc) / float64(opCount) / 1024

		metrics := PerformanceMetrics{
			OperationCount: int64(opCount),
			TotalDuration:   actualDuration,
			MinDuration:     minTime,
			MaxDuration:     maxTime,
			AvgDuration:     totalOpTime / time.Duration(opCount),
			ThroughputQPS:   float64(opCount) / actualDuration.Seconds(),
			MemoryUsageMB:   memoryUsedMB,
			SuccessCount:    int64(opCount),
		}

		suite.T().Logf("Resource Utilization Test Results:")
		suite.T().Logf("  Operations: %d", metrics.OperationCount)
		suite.T().Logf("  Duration: %v", metrics.TotalDuration)
		suite.T().Logf("  Throughput: %.2f ops/sec", metrics.ThroughputQPS)
		suite.T().Logf("  Memory Used: %.2f MB", metrics.MemoryUsageMB)
		suite.T().Logf("  Memory per Operation: %.2f KB", memoryPerOpKB)
		suite.T().Logf("  Avg Operation Time: %v", metrics.AvgDuration)
		suite.T().Logf("  Min Operation Time: %v", metrics.MinDuration)
		suite.T().Logf("  Max Operation Time: %v", metrics.MaxDuration)

		// Resource utilization assertions
		assert.Less(suite.T(), memoryPerOpKB, 5.0, "Memory per operation should be minimal")
		assert.Greater(suite.T(), metrics.ThroughputQPS, 100.0, "Should maintain good throughput")
		assert.Less(suite.T(), metrics.AvgDuration, 100*time.Millisecond, "Average operation should be fast")
	})
}

// TestAdvancedPerformanceTestSuite runs the complete advanced performance test suite
func TestAdvancedPerformanceTestSuite(t *testing.T) {
	suite.Run(t, new(AdvancedPerformanceTestSuite))
}

// TestPerformanceBenchmarking provides additional performance benchmarks
func TestPerformanceBenchmarking(t *testing.T) {
	t.Run("Operation Latency Distribution", func(t *testing.T) {
		const operations = 1000
		latencies := make([]time.Duration, operations)

		for i := 0; i < operations; i++ {
			start := time.Now()

			opts := &r2api.ClientOptions{
				AccountID: fmt.Sprintf("latency-test-%d", i),
				APIToken:  fmt.Sprintf("latency-token-%d", i),
			}
			client, err := r2api.NewClient(opts)

			require.NoError(t, err)
			assert.NotNil(t, client)

			latencies[i] = time.Since(start)
		}

		// Calculate statistics
		p50 := calculatePercentile(latencies, 0.50)
		p90 := calculatePercentile(latencies, 0.90)
		p95 := calculatePercentile(latencies, 0.95)
		p99 := calculatePercentile(latencies, 0.99)

		t.Logf("Latency Distribution (microseconds):")
		t.Logf("  P50: %d", p50.Microseconds())
		t.Logf("  P90: %d", p90.Microseconds())
		t.Logf("  P95: %d", p95.Microseconds())
		t.Logf("  P99: %d", p99.Microseconds())

		// Latency assertions
		assert.Less(t, p95, 10*time.Millisecond, "95th percentile latency should be under 10ms")
		assert.Less(t, p99, 50*time.Millisecond, "99th percentile latency should be under 50ms")
	})

	t.Run("Garbage Collection Impact", func(t *testing.T) {
		const iterations = 10000

		// Force GC before test
		runtime.GC()
		var beforeGC runtime.MemStats
		runtime.ReadMemStats(&beforeGC)

		// Perform operations
		for i := 0; i < iterations; i++ {
			opts := &r2api.ClientOptions{
				AccountID: "gc-test-account",
				APIToken:  "gc-test-token",
			}
			client, err := r2api.NewClient(opts)
			if err == nil {
				_ = client.GetAccountID()
			}
		}

		// Force GC after test
		runtime.GC()
		var afterGC runtime.MemStats
		runtime.ReadMemStats(&afterGC)

		gcCycles := afterGC.NumGC - beforeGC.NumGC
		memoryAllocated := afterGC.TotalAlloc - beforeGC.TotalAlloc

		t.Logf("Garbage Collection Impact:")
		t.Logf("  Operations: %d", iterations)
		t.Logf("  GC Cycles: %d", gcCycles)
		t.Logf("  Memory Allocated: %d bytes", memoryAllocated)
		t.Logf("  Memory per Operation: %.2f bytes", float64(memoryAllocated)/float64(iterations))

		// GC impact assertions
		assert.Less(t, gcCycles, uint32(100), "Should not trigger excessive GC cycles")
	})
}

// Helper function to calculate percentiles
func calculatePercentile(durations []time.Duration, percentile float64) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	// Simple implementation - in production, you'd want a more efficient algorithm
	// This is just for demonstration purposes
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)

	// Simple bubble sort for demonstration (inefficient for large arrays)
	for i := 0; i < len(sorted); i++ {
		for j := 0; j < len(sorted)-1-i; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	index := int(float64(len(sorted)) * percentile)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return sorted[index]
}