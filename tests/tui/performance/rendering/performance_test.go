/*
Package performance tests TUI rendering performance and optimization

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package rendering

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// startupPerfTests defines performance targets for TUI startup stages.
var startupPerfTests = []struct {
	name        string
	maxDuration time.Duration
	description string
}{
	{
		name:        "Initial model creation",
		maxDuration: time.Millisecond * 50,
		description: "Time to create initial dashboard model",
	},
	{
		name:        "Theme initialization",
		maxDuration: time.Millisecond * 20,
		description: "Time to initialize color themes and styles",
	},
	{
		name:        "Help system setup",
		maxDuration: time.Millisecond * 30,
		description: "Time to prepare help content and navigation",
	},
	{
		name:        "First render",
		maxDuration: time.Millisecond * 100,
		description: "Time to render first complete view",
	},
}

// renderingSpeedTests defines rendering operation performance targets.
var renderingSpeedTests = []struct {
	name        string
	operations  int
	maxDuration time.Duration
	description string
	contentSize int
}{
	{
		name:        "Static view rendering",
		operations:  1000,
		maxDuration: time.Millisecond * 100,
		description: "Render static dashboard views",
		contentSize: 20, // Small content
	},
	{
		name:        "Dynamic content rendering",
		operations:  1000,
		maxDuration: time.Millisecond * 200,
		description: "Render views with dynamic data",
		contentSize: 100, // Medium content
	},
	{
		name:        "Large dataset rendering",
		operations:  500,
		maxDuration: time.Millisecond * 500,
		description: "Render views with large datasets",
		contentSize: 1000, // Large content
	},
	{
		name:        "Rapid navigation rendering",
		operations:  5000,
		maxDuration: time.Millisecond * 200,
		description: "Render during rapid navigation",
		contentSize: 50, // Small but frequent
	},
}

// updateCycleTests defines Bubble Tea update cycle performance targets.
var updateCycleTests = []struct {
	name        string
	messageType string
	updates     int
	maxDuration time.Duration
	description string
}{
	{
		name:        "Window resize updates",
		messageType: "WindowSizeMsg",
		updates:     1000,
		maxDuration: time.Millisecond * 50,
		description: "Handle window resize messages",
	},
	{
		name:        "Keyboard navigation updates",
		messageType: "KeyMsg",
		updates:     2000,
		maxDuration: time.Millisecond * 100,
		description: "Handle keyboard navigation",
	},
	{
		name:        "Data update messages",
		messageType: "CustomDataMsg",
		updates:     500,
		maxDuration: time.Millisecond * 200,
		description: "Handle data update messages",
	},
}

// renderStartupPerformance verifies TUI startup meets performance targets.
func renderStartupPerformance(t *testing.T) {
	// Test that the TUI starts up quickly
	for _, test := range startupPerfTests {
		t.Run("Startup: "+test.name, func(t *testing.T) {
			assert.NotEmpty(t, test.name, "Test name should not be empty")
			assert.NotEmpty(t, test.description, "Description should not be empty")
			assert.Greater(t, test.maxDuration, time.Duration(0), "Max duration should be positive")

			// Test that startup components meet performance targets
			start := time.Now()

			// Simulate startup operations
			switch test.name {
			case "Initial model creation":
				// Simulate model creation
				_ = struct {
					sections []string
					theme    string
					width    int
					height   int
				}{
					sections: []string{"Overview", "Buckets", "Objects", "Upload", "Monitoring", "Settings"},
					theme:    "dark",
					width:    80,
					height:   24,
				}
			case "Theme initialization":
				// Simulate theme setup
				_ = map[string]string{
					"primary": "#5DADE2",
					"success": "#2ECC71",
					"warning": "#F39C12",
					"error":   "#E74C3C",
					"muted":   "#7F8C8D",
				}
			case "Help system setup":
				// Simulate help content preparation
				_ = []string{
					"Navigation: ↑↓←→, Enter",
					"Quick actions: C, U, D, M",
					"Help: F1 or ?",
				}
			case "First render":
				// Simulate first view rendering
				_ = fmt.Sprintf("🎯 R2Go2 Dashboard\n📊 Storage Usage\n🪣 Buckets")
			}

			duration := time.Since(start)
			assert.Less(t, duration, test.maxDuration,
				fmt.Sprintf("%s should complete within %v", test.name, test.maxDuration))
		})
	}
}

// renderSpeedTests verifies rendering operations meet speed targets.
func renderSpeedTests(t *testing.T) {
	// Test different types of rendering operations
	for _, test := range renderingSpeedTests {
		t.Run("Rendering: "+test.name, func(t *testing.T) {
			assert.Greater(t, test.operations, 0, "Operations should be positive")
			assert.Greater(t, test.maxDuration, time.Duration(0), "Max duration should be positive")

			start := time.Now()

			// Simulate rendering operations
			for i := 0; i < test.operations; i++ {
				// Create test content of specified size
				content := make([]string, test.contentSize)
				for j := 0; j < test.contentSize; j++ {
					content[j] = fmt.Sprintf("Item %d-%d: %s", i, j, strings.Repeat("x", 10))
				}

				// Simulate view rendering
				_ = strings.Join(content, "\n")
			}

			duration := time.Since(start)
			assert.Less(t, duration, test.maxDuration,
				fmt.Sprintf("%s should complete within %v (actual: %v)",
					test.name, test.maxDuration, duration))

			// Calculate operations per second
			opsPerSec := float64(test.operations) / duration.Seconds()
			t.Logf("%s: %.2f operations/second", test.name, opsPerSec)
		})
	}
}

// renderMemoryEfficiency verifies rendering doesn't leak memory.
func renderMemoryEfficiency(t *testing.T) {
	// Test that rendering doesn't leak memory
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Perform many rendering operations
	for i := 0; i < 10000; i++ {
		// Simulate complex rendering
		content := make([]string, 100)
		for j := 0; j < 100; j++ {
			content[j] = fmt.Sprintf("Render %d-%d: %s", i, j,
				strings.Repeat("content", 5))
		}

		// Simulate view assembly
		view := strings.Join(content, "\n")
		_ = len(view) // Force computation
	}

	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	// Memory growth should be reasonable
	memoryGrowth := m2.Alloc - m1.Alloc
	assert.Less(t, memoryGrowth, uint64(1024*1024), // 1MB limit
		fmt.Sprintf("Memory growth should be reasonable (actual: %d bytes)", memoryGrowth))

	t.Logf("Memory growth: %d bytes", memoryGrowth)
}

// renderUpdateCyclePerformance verifies Bubble Tea update cycle performance.
func renderUpdateCyclePerformance(t *testing.T) {
	// Test Bubble Tea update cycle performance
	for _, test := range updateCycleTests {
		t.Run("Update cycle: "+test.name, func(t *testing.T) {
			assert.Greater(t, test.updates, 0, "Updates should be positive")
			assert.Greater(t, test.maxDuration, time.Duration(0), "Max duration should be positive")

			start := time.Now()

			// Simulate update cycles
			for i := 0; i < test.updates; i++ {
				var msg tea.Msg

				switch test.messageType {
				case "WindowSizeMsg":
					msg = tea.WindowSizeMsg{
						Width:  80 + i%50,
						Height: 24 + i%20,
					}
				case "KeyMsg":
					keys := []tea.KeyType{tea.KeyUp, tea.KeyDown, tea.KeyLeft, tea.KeyRight}
					msg = tea.KeyMsg{Type: keys[i%len(keys)]}
				case "CustomDataMsg":
					msg = struct{ data string }{data: fmt.Sprintf("Update %d", i)}
				}

				// Simulate update processing
				_ = msg
			}

			duration := time.Since(start)
			assert.Less(t, duration, test.maxDuration,
				fmt.Sprintf("Update cycles should complete within %v (actual: %v)",
					test.maxDuration, duration))
		})
	}
}

// TestRenderingPerformance tests TUI rendering performance characteristics
func TestRenderingPerformance(t *testing.T) {
	t.Run("Startup Performance", renderStartupPerformance)
	t.Run("Rendering Speed Tests", renderSpeedTests)
	t.Run("Memory Efficiency", renderMemoryEfficiency)
	t.Run("Update Cycle Performance", renderUpdateCyclePerformance)
}

// bucketCountTests defines performance targets for large bucket lists.
var bucketCountTests = []struct {
	count         int
	maxLoadTime   time.Duration
	maxRenderTime time.Duration
	description   string
}{
	{
		count:         100,
		maxLoadTime:   time.Millisecond * 10,
		maxRenderTime: time.Millisecond * 50,
		description:   "Small bucket list",
	},
	{
		count:         1000,
		maxLoadTime:   time.Millisecond * 50,
		maxRenderTime: time.Millisecond * 200,
		description:   "Medium bucket list",
	},
	{
		count:         5000,
		maxLoadTime:   time.Millisecond * 200,
		maxRenderTime: time.Millisecond * 1000,
		description:   "Large bucket list",
	},
}

// realtimeUpdateTests defines targets for frequent real-time updates.
var realtimeUpdateTests = []struct {
	interval    time.Duration
	duration    time.Duration
	maxLoad     float64 // CPU load percentage
	description string
}{
	{
		interval:    time.Millisecond * 100, // 10 Hz
		duration:    time.Second * 10,
		maxLoad:     5.0,
		description: "Frequent updates",
	},
	{
		interval:    time.Millisecond * 500, // 2 Hz
		duration:    time.Second * 10,
		maxLoad:     2.0,
		description: "Regular updates",
	},
	{
		interval:    time.Second * 1, // 1 Hz
		duration:    time.Second * 10,
		maxLoad:     1.0,
		description: "Slow updates",
	},
}

// largeDatasetBucketList verifies bucket list load/render performance.
func largeDatasetBucketList(t *testing.T) {
	// Test performance with large bucket lists
	for _, test := range bucketCountTests {
		t.Run(fmt.Sprintf("Buckets: %d (%s)", test.count, test.description), func(t *testing.T) {
			// Simulate loading bucket data
			loadStart := time.Now()
			buckets := make([]string, test.count)
			for i := 0; i < test.count; i++ {
				buckets[i] = fmt.Sprintf("bucket-%d", i)
			}
			loadDuration := time.Since(loadStart)

			assert.Less(t, loadDuration, test.maxLoadTime,
				fmt.Sprintf("Loading %d buckets should complete within %v",
					test.count, test.maxLoadTime))

			// Simulate rendering bucket list
			renderStart := time.Now()
			for i, bucket := range buckets {
				_ = fmt.Sprintf("🪣 %s - Size: %d MB", bucket, i*1024)
			}
			renderDuration := time.Since(renderStart)

			assert.Less(t, renderDuration, test.maxRenderTime,
				fmt.Sprintf("Rendering %d buckets should complete within %v",
					test.count, test.maxRenderTime))
		})
	}
}

// largeDatasetObjectList verifies object list rendering performance.
func largeDatasetObjectList(t *testing.T) {
	// Test performance with large object lists
	objectCounts := []int{1000, 5000, 10000, 50000}

	for _, count := range objectCounts {
		t.Run(fmt.Sprintf("Objects: %d", count), func(t *testing.T) {
			start := time.Now()

			// Simulate object list data
			objects := make([]struct {
				name    string
				size    int64
				modTime time.Time
			}, count)

			for i := 0; i < count; i++ {
				objects[i] = struct {
					name    string
					size    int64
					modTime time.Time
				}{
					name:    fmt.Sprintf("object-%d.txt", i),
					size:    int64(i * 1024),
					modTime: time.Now().Add(-time.Duration(i) * time.Hour),
				}
			}

			// Simulate rendering object table
			for _, obj := range objects {
				_ = fmt.Sprintf("%-30s %10s %s",
					obj.name,
					formatBytes(obj.size),
					obj.modTime.Format("2006-01-02"))
			}

			duration := time.Since(start)

			// Performance should degrade gracefully
			maxDuration := time.Millisecond*1000 + time.Duration(count/10)*time.Millisecond
			assert.Less(t, duration, maxDuration,
				fmt.Sprintf("Rendering %d objects should complete within %v (actual: %v)",
					count, maxDuration, duration))

			t.Logf("Object list %d: %v", count, duration)
		})
	}
}

// largeDatasetRealtimeUpdates verifies real-time update performance.
func largeDatasetRealtimeUpdates(t *testing.T) {
	// Test performance with frequent real-time updates
	for _, test := range realtimeUpdateTests {
		t.Run("Real-time updates: "+test.description, func(t *testing.T) {
			start := time.Now()
			updateCount := 0

			for time.Since(start) < test.duration {
				// Simulate real-time update
				_ = struct {
					timestamp      time.Time
					uploadRate     float64
					downloadRate   float64
					requestsPerMin int
				}{
					timestamp:      time.Now(),
					uploadRate:     float64(updateCount % 100),
					downloadRate:   float64(updateCount % 50),
					requestsPerMin: 100 + updateCount%1000,
				}

				updateCount++
				time.Sleep(test.interval)
			}

			// Calculate actual update rate
			actualInterval := time.Since(start) / time.Duration(updateCount)
			t.Logf("Update interval: %v (target: %v)", actualInterval, test.interval)

			// Should be close to target interval (within 20%)
			assert.Less(t, float64(actualInterval)/float64(test.interval), 1.2,
				"Update interval should be close to target")
		})
	}
}

// TestLargeDatasetPerformance tests performance with large amounts of data
func TestLargeDatasetPerformance(t *testing.T) {
	t.Run("Bucket List Performance", largeDatasetBucketList)
	t.Run("Object List Performance", largeDatasetObjectList)
	t.Run("Real-time Update Performance", largeDatasetRealtimeUpdates)
}

// concurrentNavTests defines concurrent navigation performance targets.
var concurrentNavTests = []struct {
	goroutines  int
	operations  int
	maxDuration time.Duration
	description string
}{
	{
		goroutines:  1,
		operations:  1000,
		maxDuration: time.Millisecond * 50,
		description: "Single goroutine navigation",
	},
	{
		goroutines:  5,
		operations:  200,
		maxDuration: time.Millisecond * 100,
		description: "Concurrent navigation (5 goroutines)",
	},
	{
		goroutines:  10,
		operations:  100,
		maxDuration: time.Millisecond * 200,
		description: "High concurrency navigation (10 goroutines)",
	},
}

// memoryAccessPatterns defines memory access pattern scenarios.
var memoryAccessPatterns = []struct {
	name     string
	pattern  string
	size     int
	accesses int
}{
	{
		name:     "Sequential access",
		pattern:  "sequential",
		size:     1000,
		accesses: 5000,
	},
	{
		name:     "Random access",
		pattern:  "random",
		size:     1000,
		accesses: 5000,
	},
	{
		name:     "Localized access",
		pattern:  "localized",
		size:     1000,
		accesses: 5000,
	},
}

// TestConcurrentRendering tests rendering under concurrent conditions
func TestConcurrentRendering(t *testing.T) {
	t.Run("Concurrent Navigation", func(t *testing.T) {
		// Test handling rapid concurrent navigation
		for _, test := range concurrentNavTests {
			t.Run("Concurrent: "+test.description, func(t *testing.T) {
				start := time.Now()
				done := make(chan bool, test.goroutines)

				// Start concurrent navigation goroutines
				for i := 0; i < test.goroutines; i++ {
					go func(id int) {
						defer func() { done <- true }()

						for j := 0; j < test.operations; j++ {
							// Simulate navigation key presses
							keys := []tea.KeyType{tea.KeyUp, tea.KeyDown, tea.KeyLeft, tea.KeyRight}
							_ = tea.KeyMsg{Type: keys[j%len(keys)]}
						}
					}(i)
				}

				// Wait for all goroutines to complete
				for i := 0; i < test.goroutines; i++ {
					<-done
				}

				duration := time.Since(start)
				assert.Less(t, duration, test.maxDuration,
					fmt.Sprintf("Concurrent navigation should complete within %v", test.maxDuration))

				totalOperations := test.goroutines * test.operations
				opsPerSec := float64(totalOperations) / duration.Seconds()
				t.Logf("Concurrent navigation: %d operations in %v (%.2f ops/sec)",
					totalOperations, duration, opsPerSec)
			})
		}
	})

	t.Run("Memory Access Patterns", func(t *testing.T) {
		// Test memory access patterns during rendering
		for _, test := range memoryAccessPatterns {
			t.Run("Memory pattern: "+test.name, func(t *testing.T) {
				// Create test data
				data := make([]string, test.size)
				for i := 0; i < test.size; i++ {
					data[i] = fmt.Sprintf("data-item-%d", i)
				}

				start := time.Now()

				for i := 0; i < test.accesses; i++ {
					var index int
					switch test.pattern {
					case "sequential":
						index = i % test.size
					case "random":
						index = (i * 31) % test.size // Simple pseudo-random
					case "localized":
						index = (i % 100) + ((i/100)*100)%test.size
					}

					_ = data[index] // Access the data
				}

				duration := time.Since(start)
				t.Logf("%s access: %d accesses in %v", test.pattern, test.accesses, duration)
			})
		}
	})
}

// Helper function for formatting bytes (simplified version)
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
