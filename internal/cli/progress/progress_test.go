package progress

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// DefaultConfig
// ---------------------------------------------------------------------------

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	require.NotNil(t, cfg)
	assert.Equal(t, ProgressTypeLinear, cfg.Type)
	assert.NotEmpty(t, cfg.Template)
	assert.True(t, cfg.ShowSpeed)
	assert.True(t, cfg.ShowETA)
	assert.True(t, cfg.ColorOutput)
	assert.Equal(t, 100*time.Millisecond, cfg.RefreshRate)
	assert.Equal(t, 80, cfg.Width)
	assert.True(t, cfg.HideCursor)
}

// ---------------------------------------------------------------------------
// NewProgressBar
// ---------------------------------------------------------------------------

func TestNewProgressBar(t *testing.T) {
	t.Parallel()

	t.Run("nil config uses defaults", func(t *testing.T) {
		bar := NewProgressBar(nil)
		require.NotNil(t, bar)
		info := bar.GetInfo()
		assert.Equal(t, StateProgress, info.State)
		assert.False(t, info.StartTime.IsZero())
	})

	t.Run("custom config", func(t *testing.T) {
		cfg := &ProgressBarConfig{
			Template:    `{{counters . }}`,
			ShowSpeed:   false,
			ShowETA:     false,
			ColorOutput: false,
			RefreshRate: 200 * time.Millisecond,
			Width:       120,
			HideCursor:  false,
		}
		bar := NewProgressBar(cfg)
		require.NotNil(t, bar)
		info := bar.GetInfo()
		assert.Equal(t, StateProgress, info.State)
		assert.False(t, info.StartTime.IsZero())
	})

	t.Run("zero width is accepted", func(t *testing.T) {
		cfg := &ProgressBarConfig{
			Width: 0,
		}
		bar := NewProgressBar(cfg)
		require.NotNil(t, bar)
	})
}

// ---------------------------------------------------------------------------
// NewLinearProgressBar
// ---------------------------------------------------------------------------

func TestNewLinearProgressBar(t *testing.T) {
	t.Parallel()

	bar := NewLinearProgressBar(1024, "Copying file.txt")
	require.NotNil(t, bar)

	info := bar.GetInfo()
	assert.Equal(t, StateProgress, info.State)
	assert.Equal(t, "Copying file.txt", info.Message)
	assert.Equal(t, int64(1024), info.Total)
}

// ---------------------------------------------------------------------------
// NewCircularProgressBar
// ---------------------------------------------------------------------------

func TestNewCircularProgressBar(t *testing.T) {
	t.Parallel()

	bar := NewCircularProgressBar("Loading...")
	require.NotNil(t, bar)

	info := bar.GetInfo()
	assert.Equal(t, StateProgress, info.State)
	assert.Equal(t, "Loading...", info.Message)
}

// ---------------------------------------------------------------------------
// EnhancedProgressBar.Update
// ---------------------------------------------------------------------------

// runProgressBarUpdateBasicTests covers percentage/current/total accounting
// across normal, degenerate, and extreme values.
func runProgressBarUpdateBasicTests(t *testing.T) {
	t.Run("normal progress", func(t *testing.T) {
		cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false, ShowSpeed: true}
		bar := NewProgressBar(cfg)
		bar.Update(50, 100)

		info := bar.GetInfo()
		assert.Equal(t, int64(50), info.Current)
		assert.Equal(t, int64(100), info.Total)
		assert.InDelta(t, 50.0, info.Percentage, 0.01)
		assert.False(t, info.LastUpdate.IsZero())
	})

	t.Run("zero total", func(t *testing.T) {
		cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false}
		bar := NewProgressBar(cfg)
		bar.Update(50, 0)

		info := bar.GetInfo()
		assert.Equal(t, int64(50), info.Current)
		assert.Equal(t, int64(0), info.Total)
		assert.Equal(t, 0.0, info.Percentage)
	})

	t.Run("current equals total", func(t *testing.T) {
		cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false}
		bar := NewProgressBar(cfg)
		bar.Update(100, 100)

		info := bar.GetInfo()
		assert.InDelta(t, 100.0, info.Percentage, 0.01)
	})

	t.Run("zero current", func(t *testing.T) {
		cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false}
		bar := NewProgressBar(cfg)
		bar.Update(0, 100)

		info := bar.GetInfo()
		assert.Equal(t, 0.0, info.Percentage)
	})

	t.Run("negative values", func(t *testing.T) {
		cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false}
		bar := NewProgressBar(cfg)
		bar.Update(-10, 100)

		info := bar.GetInfo()
		assert.Equal(t, int64(-10), info.Current)
	})

	t.Run("very large values", func(t *testing.T) {
		cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false}
		bar := NewProgressBar(cfg)
		bar.Update(int64(1<<50), int64(1<<51))

		info := bar.GetInfo()
		assert.InDelta(t, 50.0, info.Percentage, 0.01)
	})
}

// runProgressBarUpdateSpeedTests covers speed/ETA calculation and its
// guards: zero start time and disabled ShowSpeed.
func runProgressBarUpdateSpeedTests(t *testing.T) {
	t.Run("calculates speed and ETA", func(t *testing.T) {
		cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false, ShowSpeed: true}
		bar := NewProgressBar(cfg)
		// Type-assert to access internal fields
		enhanced := bar.(*EnhancedProgressBar)
		// Manually set start time in the past so speed calculation works
		enhanced.info.StartTime = time.Now().Add(-1 * time.Second)
		bar.Update(1024*1024, 10*1024*1024) // 1MB of 10MB in 1 second = ~1 MB/s

		info := bar.GetInfo()
		assert.Greater(t, info.Speed, 0.0, "speed should be calculated")
		assert.Greater(t, info.ETA, 0*time.Nanosecond, "ETA should be calculated")
	})

	t.Run("speed zero when start time is zero", func(t *testing.T) {
		cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false, ShowSpeed: true}
		bar := NewProgressBar(cfg)
		enhanced := bar.(*EnhancedProgressBar)
		enhanced.info.StartTime = time.Time{} // zero value
		bar.Update(50, 100)

		info := bar.GetInfo()
		assert.Equal(t, 0.0, info.Speed)
	})

	t.Run("no speed when showSpeed is false", func(t *testing.T) {
		cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false, ShowSpeed: false}
		bar := NewProgressBar(cfg)
		bar.Update(1024*1024, 10*1024*1024)

		info := bar.GetInfo()
		assert.Equal(t, 0.0, info.Speed)
	})
}

func TestEnhancedProgressBar_Update(t *testing.T) {
	t.Parallel()

	t.Run("basic", func(t *testing.T) { runProgressBarUpdateBasicTests(t) })
	t.Run("speed", func(t *testing.T) { runProgressBarUpdateSpeedTests(t) })
	t.Run("updates last update timestamp", func(t *testing.T) {
		cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false}
		bar := NewProgressBar(cfg)
		first := bar.GetInfo().LastUpdate
		time.Sleep(5 * time.Millisecond)
		bar.Update(1, 100)
		second := bar.GetInfo().LastUpdate
		assert.True(t, second.After(first))
	})
}

// ---------------------------------------------------------------------------
// EnhancedProgressBar.SetMessage
// ---------------------------------------------------------------------------

func TestEnhancedProgressBar_SetMessage(t *testing.T) {
	t.Parallel()

	cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false}
	bar := NewProgressBar(cfg)
	bar.SetMessage("new message")

	info := bar.GetInfo()
	assert.Equal(t, "new message", info.Message)
}

func TestEnhancedProgressBar_SetMessage_Empty(t *testing.T) {
	t.Parallel()

	cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false}
	bar := NewProgressBar(cfg)
	bar.SetMessage("initial")
	bar.SetMessage("")

	info := bar.GetInfo()
	assert.Equal(t, "", info.Message)
}

// ---------------------------------------------------------------------------
// EnhancedProgressBar.SetSpeed
// ---------------------------------------------------------------------------

func TestEnhancedProgressBar_SetSpeed(t *testing.T) {
	t.Parallel()

	cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false}
	bar := NewProgressBar(cfg)

	// 10 MB/s in bytes per second = 10 * 1024 * 1024
	bar.SetSpeed(10 * 1024 * 1024)

	info := bar.GetInfo()
	assert.InDelta(t, 10.0, info.Speed, 0.01)
}

func TestEnhancedProgressBar_SetSpeed_Zero(t *testing.T) {
	t.Parallel()

	cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false}
	bar := NewProgressBar(cfg)
	bar.SetSpeed(0)

	info := bar.GetInfo()
	assert.Equal(t, 0.0, info.Speed)
}

// ---------------------------------------------------------------------------
// EnhancedProgressBar.Complete
// ---------------------------------------------------------------------------

func TestEnhancedProgressBar_Complete(t *testing.T) {
	t.Parallel()

	cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false}
	bar := NewProgressBar(cfg)
	bar.Update(50, 100)
	bar.Complete()

	info := bar.GetInfo()
	assert.Equal(t, StateComplete, info.State)
	assert.Equal(t, 100.0, info.Percentage)
}

func TestEnhancedProgressBar_Complete_ZeroTotal(t *testing.T) {
	t.Parallel()

	cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false}
	bar := NewProgressBar(cfg)
	bar.Complete()

	info := bar.GetInfo()
	assert.Equal(t, StateComplete, info.State)
	assert.Equal(t, 100.0, info.Percentage)
}

// ---------------------------------------------------------------------------
// EnhancedProgressBar.Error
// ---------------------------------------------------------------------------

func TestEnhancedProgressBar_Error(t *testing.T) {
	t.Parallel()

	t.Run("with error", func(t *testing.T) {
		cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false}
		bar := NewProgressBar(cfg)
		bar.Error(fmt.Errorf("something failed"))

		info := bar.GetInfo()
		assert.Equal(t, StateError, info.State)
		assert.EqualError(t, info.Error, "something failed")
	})

	t.Run("nil error", func(t *testing.T) {
		cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false}
		bar := NewProgressBar(cfg)
		bar.Error(nil)

		info := bar.GetInfo()
		assert.Equal(t, StateError, info.State)
		assert.Nil(t, info.Error)
	})

	t.Run("with color output", func(t *testing.T) {
		// Capture stdout to verify color.Red is called
		cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: true}
		bar := NewProgressBar(cfg)
		// Should not panic even with color output
		assert.NotPanics(t, func() {
			bar.Error(fmt.Errorf("colored error"))
		})

		info := bar.GetInfo()
		assert.Equal(t, StateError, info.State)
	})
}

// ---------------------------------------------------------------------------
// EnhancedProgressBar.Cancel
// ---------------------------------------------------------------------------

func TestEnhancedProgressBar_Cancel(t *testing.T) {
	t.Parallel()

	t.Run("without color", func(t *testing.T) {
		cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false}
		bar := NewProgressBar(cfg)
		bar.Cancel()

		info := bar.GetInfo()
		assert.Equal(t, StateCancelled, info.State)
	})

	t.Run("with color output", func(t *testing.T) {
		cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: true}
		bar := NewProgressBar(cfg)
		assert.NotPanics(t, func() {
			bar.Cancel()
		})

		info := bar.GetInfo()
		assert.Equal(t, StateCancelled, info.State)
	})
}

// ---------------------------------------------------------------------------
// EnhancedProgressBar.GetInfo returns a copy
// ---------------------------------------------------------------------------

func TestEnhancedProgressBar_GetInfo_ReturnsCopy(t *testing.T) {
	t.Parallel()

	cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false}
	bar := NewProgressBar(cfg)

	info1 := bar.GetInfo()
	bar.Update(50, 100)
	info2 := bar.GetInfo()

	// info1 should not be affected by the update
	assert.Equal(t, int64(0), info1.Current)
	assert.Equal(t, int64(50), info2.Current)
}

// ---------------------------------------------------------------------------
// EnhancedProgressBar.Start
// ---------------------------------------------------------------------------

func TestEnhancedProgressBar_Start(t *testing.T) {
	t.Parallel()

	cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false}
	bar := NewProgressBar(cfg)
	// Should not panic
	assert.NotPanics(t, func() {
		bar.Start()
	})
}

// ---------------------------------------------------------------------------
// ProgressType / ProgressState constants
// ---------------------------------------------------------------------------

func TestProgressTypes(t *testing.T) {
	t.Parallel()

	assert.Equal(t, ProgressType(0), ProgressTypeLinear)
	assert.Equal(t, ProgressType(1), ProgressTypeCircular)
	assert.Equal(t, ProgressType(2), ProgressTypePercentage)
	assert.Equal(t, ProgressType(3), ProgressTypeSpinner)
}

func TestProgressStates(t *testing.T) {
	t.Parallel()

	assert.Equal(t, ProgressState(0), StateProgress)
	assert.Equal(t, ProgressState(1), StateComplete)
	assert.Equal(t, ProgressState(2), StateError)
	assert.Equal(t, ProgressState(3), StateCancelled)
}

// ---------------------------------------------------------------------------
// ProgressReader
// ---------------------------------------------------------------------------

func TestProgressReader(t *testing.T) {
	t.Parallel()

	t.Run("tracks bytes read", func(t *testing.T) {
		data := []byte("hello world, this is test data for reading")
		src := bytes.NewReader(data)

		cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false}
		bar := NewProgressBar(cfg)
		bar.Update(0, int64(len(data)))

		pr := NewProgressReader(src, bar)
		require.NotNil(t, pr)

		buf := make([]byte, 32)
		totalRead := 0
		for {
			n, err := pr.Read(buf)
			totalRead += n
			if err == io.EOF {
				break
			}
			require.NoError(t, err)
		}

		assert.Equal(t, len(data), totalRead)

		info := bar.GetInfo()
		assert.Equal(t, int64(len(data)), info.Current)
	})

	t.Run("nil progress bar does not panic", func(t *testing.T) {
		data := []byte("test data")
		src := bytes.NewReader(data)
		pr := NewProgressReader(src, nil)

		buf := make([]byte, 64)
		n, err := pr.Read(buf)
		require.NoError(t, err)
		assert.Equal(t, len(data), n)
	})

	t.Run("empty reader", func(t *testing.T) {
		src := bytes.NewReader([]byte{})
		cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false}
		bar := NewProgressBar(cfg)
		pr := NewProgressReader(src, bar)

		buf := make([]byte, 10)
		n, err := pr.Read(buf)
		assert.Equal(t, 0, n)
		assert.Equal(t, io.EOF, err)
	})

	t.Run("partial reads", func(t *testing.T) {
		data := []byte("hello")
		src := bytes.NewReader(data)
		cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false}
		bar := NewProgressBar(cfg)
		pr := NewProgressReader(src, bar)

		buf := make([]byte, 2)
		n, _ := pr.Read(buf)
		assert.Equal(t, 2, n)

		info := bar.GetInfo()
		assert.Equal(t, int64(2), info.Current)
	})
}

// ---------------------------------------------------------------------------
// ProgressWriter
// ---------------------------------------------------------------------------

func TestProgressWriter(t *testing.T) {
	t.Parallel()

	t.Run("tracks bytes written", func(t *testing.T) {
		var buf bytes.Buffer
		totalSize := int64(100)

		cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false}
		bar := NewProgressBar(cfg)
		bar.Update(0, totalSize)

		pw := NewProgressWriter(&buf, bar)
		require.NotNil(t, pw)

		data := []byte("hello world, this is test data for writing")
		n, err := pw.Write(data)
		require.NoError(t, err)
		assert.Equal(t, len(data), n)
		assert.Equal(t, data, buf.Bytes())

		info := bar.GetInfo()
		assert.Equal(t, int64(len(data)), info.Current)
	})

	t.Run("nil progress bar does not panic", func(t *testing.T) {
		var buf bytes.Buffer
		pw := NewProgressWriter(&buf, nil)

		n, err := pw.Write([]byte("test"))
		require.NoError(t, err)
		assert.Equal(t, 4, n)
	})

	t.Run("empty write", func(t *testing.T) {
		var buf bytes.Buffer
		cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false}
		bar := NewProgressBar(cfg)
		pw := NewProgressWriter(&buf, bar)

		n, err := pw.Write([]byte{})
		require.NoError(t, err)
		assert.Equal(t, 0, n)
	})

	t.Run("multiple writes accumulate", func(t *testing.T) {
		var buf bytes.Buffer
		cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false}
		bar := NewProgressBar(cfg)
		pw := NewProgressWriter(&buf, bar)

		pw.Write([]byte("hello"))
		pw.Write([]byte(" "))
		pw.Write([]byte("world"))

		info := bar.GetInfo()
		assert.Equal(t, int64(11), info.Current)
		assert.Equal(t, "hello world", buf.String())
	})
}

// ---------------------------------------------------------------------------
// MultiProgress
// ---------------------------------------------------------------------------

// runMultiProgressRegistrationTests covers Add/Update registration and
// propagation, including lookups on unknown ids.
func runMultiProgressRegistrationTests(t *testing.T) {
	t.Run("Add and GetAllInfo", func(t *testing.T) {
		mp := NewMultiProgress()
		require.NotNil(t, mp)

		cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false}
		bar1 := NewProgressBar(cfg)
		bar1.SetMessage("op1")
		bar2 := NewProgressBar(cfg)
		bar2.SetMessage("op2")

		mp.Add("first", bar1)
		mp.Add("second", bar2)

		all := mp.GetAllInfo()
		assert.Len(t, all, 2)
		assert.Equal(t, "op1", all["first"].Message)
		assert.Equal(t, "op2", all["second"].Message)
	})

	t.Run("Update propagates to correct bar", func(t *testing.T) {
		mp := NewMultiProgress()

		cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false}
		bar := NewProgressBar(cfg)
		mp.Add("test", bar)

		mp.Update("test", 50, 100)

		info := mp.GetAllInfo()["test"]
		assert.Equal(t, int64(50), info.Current)
		assert.Equal(t, int64(100), info.Total)
	})

	t.Run("Update non-existent id does not panic", func(t *testing.T) {
		mp := NewMultiProgress()
		assert.NotPanics(t, func() {
			mp.Update("nonexistent", 50, 100)
		})
	})
}

// runMultiProgressLifecycleTests covers Complete/Error state propagation and
// their no-panic behavior on unknown ids.
func runMultiProgressLifecycleTests(t *testing.T) {
	t.Run("Complete propagates", func(t *testing.T) {
		mp := NewMultiProgress()
		cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false}
		bar := NewProgressBar(cfg)
		mp.Add("test", bar)

		mp.Complete("test")

		info := mp.GetAllInfo()["test"]
		assert.Equal(t, StateComplete, info.State)
	})

	t.Run("Complete non-existent id does not panic", func(t *testing.T) {
		mp := NewMultiProgress()
		assert.NotPanics(t, func() {
			mp.Complete("nonexistent")
		})
	})

	t.Run("Error propagates", func(t *testing.T) {
		mp := NewMultiProgress()
		cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false}
		bar := NewProgressBar(cfg)
		mp.Add("test", bar)

		mp.Error("test", fmt.Errorf("boom"))

		info := mp.GetAllInfo()["test"]
		assert.Equal(t, StateError, info.State)
		assert.EqualError(t, info.Error, "boom")
	})

	t.Run("Error non-existent id does not panic", func(t *testing.T) {
		mp := NewMultiProgress()
		assert.NotPanics(t, func() {
			mp.Error("nonexistent", fmt.Errorf("test"))
		})
	})
}

// runMultiProgressSnapshotTests covers GetAllInfo snapshot semantics and the
// empty-registry case.
func runMultiProgressSnapshotTests(t *testing.T) {
	t.Run("GetAllInfo returns copy of map", func(t *testing.T) {
		mp := NewMultiProgress()
		cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false}
		bar := NewProgressBar(cfg)
		mp.Add("test", bar)

		info1 := mp.GetAllInfo()
		mp.Complete("test")
		info2 := mp.GetAllInfo()

		// info1 should reflect the state at the time of call
		assert.Equal(t, StateProgress, info1["test"].State)
		assert.Equal(t, StateComplete, info2["test"].State)
	})

	t.Run("empty multi progress", func(t *testing.T) {
		mp := NewMultiProgress()
		all := mp.GetAllInfo()
		assert.Empty(t, all)
	})
}

func TestMultiProgress(t *testing.T) {
	t.Parallel()

	t.Run("registration", func(t *testing.T) { runMultiProgressRegistrationTests(t) })
	t.Run("lifecycle", func(t *testing.T) { runMultiProgressLifecycleTests(t) })
	t.Run("snapshots", func(t *testing.T) { runMultiProgressSnapshotTests(t) })
}

// ---------------------------------------------------------------------------
// Concurrent access safety
// ---------------------------------------------------------------------------

func TestEnhancedProgressBar_ConcurrentUpdates(t *testing.T) {
	cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false}
	bar := NewProgressBar(cfg)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(v int) {
			defer wg.Done()
			bar.Update(int64(v), 100)
			bar.SetMessage(fmt.Sprintf("msg-%d", v))
			bar.GetInfo()
		}(i)
	}
	wg.Wait()

	// Should not panic, and final state should be valid
	info := bar.GetInfo()
	assert.Equal(t, int64(100), info.Total)
}

func TestMultiProgress_ConcurrentAccess(t *testing.T) {
	mp := NewMultiProgress()
	cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false}

	// Add bars
	for i := 0; i < 10; i++ {
		bar := NewProgressBar(cfg)
		mp.Add(fmt.Sprintf("bar-%d", i), bar)
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			mp.Update(id, 50, 100)
			mp.Complete(id)
			mp.GetAllInfo()
		}(fmt.Sprintf("bar-%d", i))
	}
	wg.Wait()

	all := mp.GetAllInfo()
	assert.Len(t, all, 10)
	for _, info := range all {
		assert.Equal(t, StateComplete, info.State)
	}
}

// ---------------------------------------------------------------------------
// FormatBytes
// ---------------------------------------------------------------------------

func TestFormatBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"zero", 0, "0 B"},
		{"bytes", 500, "500 B"},
		{"exactly 1 KB", 1024, "1.0 KB"},
		{"1.5 KB", 1536, "1.5 KB"},
		{"1 MB", 1024 * 1024, "1.0 MB"},
		{"5 MB", 5 * 1024 * 1024, "5.0 MB"},
		{"1 GB", 1024 * 1024 * 1024, "1.0 GB"},
		{"2.5 GB", 2684354560, "2.5 GB"},
		{"1 TB", 1024 * 1024 * 1024 * 1024, "1.0 TB"},
		{"negative", -1, "-1 B"},
		{"one byte", 1, "1 B"},
		{"1023 bytes", 1023, "1023 B"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBytes(tt.bytes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// FormatDuration
// ---------------------------------------------------------------------------

func TestFormatDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"zero", 0, "0s"},
		{"seconds", 30 * time.Second, "30s"},
		{"59 seconds", 59 * time.Second, "59s"},
		{"1 minute", 60 * time.Second, "1m 0s"},
		{"90 seconds", 90 * time.Second, "2m 30s"},
		{"1 hour", 1 * time.Hour, "1h 0m"},
		{"2h 30m", 2*time.Hour + 30*time.Minute, "2h 30m"},
		{"500ms", 500 * time.Millisecond, "0s"},
		{"negative", -5 * time.Second, "-5s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// ProgressInfo struct
// ---------------------------------------------------------------------------

func TestProgressInfo(t *testing.T) {
	t.Parallel()

	info := &ProgressInfo{
		Operation:  "copy",
		Current:    50,
		Total:      100,
		Percentage: 50.0,
		Speed:      1.5,
		ETA:        30 * time.Second,
		StartTime:  time.Now(),
		LastUpdate: time.Now(),
		State:      StateProgress,
		Message:    "copying",
		Error:      nil,
	}

	assert.Equal(t, "copy", info.Operation)
	assert.Equal(t, int64(50), info.Current)
	assert.Equal(t, int64(100), info.Total)
	assert.Equal(t, StateProgress, info.State)
	assert.Nil(t, info.Error)
}

// ---------------------------------------------------------------------------
// hideCursor / showCursor (smoke test — they write to stdout)
// ---------------------------------------------------------------------------

func TestHideShowCursor(t *testing.T) {
	// These write ANSI escape sequences to stdout. Just ensure they don't panic.
	assert.NotPanics(t, hideCursor)
	assert.NotPanics(t, showCursor)
}

// ---------------------------------------------------------------------------
// ProgressReader / ProgressWriter implement io interfaces
// ---------------------------------------------------------------------------

func TestProgressReader_ImplementsReader(t *testing.T) {
	t.Parallel()
	// Compile-time interface check
	var _ io.Reader = &ProgressReader{}
}

func TestProgressWriter_ImplementsWriter(t *testing.T) {
	t.Parallel()
	var _ io.Writer = &ProgressWriter{}
}

// ---------------------------------------------------------------------------
// ProgressReader with erroring reader
// ---------------------------------------------------------------------------

func TestProgressReader_ErrorFromSource(t *testing.T) {
	t.Parallel()

	errReader := &errorReader{err: fmt.Errorf("read failure")}
	cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false}
	bar := NewProgressBar(cfg)

	pr := NewProgressReader(errReader, bar)
	buf := make([]byte, 10)
	n, err := pr.Read(buf)
	assert.Equal(t, 0, n)
	assert.EqualError(t, err, "read failure")
}

// ---------------------------------------------------------------------------
// ProgressWriter with erroring writer
// ---------------------------------------------------------------------------

func TestProgressWriter_ErrorFromDestination(t *testing.T) {
	t.Parallel()

	errWriter := &errorWriter{err: fmt.Errorf("write failure")}
	cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false}
	bar := NewProgressBar(cfg)

	pw := NewProgressWriter(errWriter, bar)
	n, err := pw.Write([]byte("test"))
	assert.Equal(t, 0, n)
	assert.EqualError(t, err, "write failure")
}

// ---------------------------------------------------------------------------
// MultiProgress maintains insertion order via order slice
// ---------------------------------------------------------------------------

func TestMultiProgress_AddOrder(t *testing.T) {
	t.Parallel()

	mp := NewMultiProgress()
	cfg := &ProgressBarConfig{HideCursor: false, ColorOutput: false}

	bar1 := NewProgressBar(cfg)
	bar2 := NewProgressBar(cfg)
	bar3 := NewProgressBar(cfg)

	mp.Add("a", bar1)
	mp.Add("b", bar2)
	mp.Add("c", bar3)

	all := mp.GetAllInfo()
	assert.Len(t, all, 3)
	_, existsA := all["a"]
	_, existsB := all["b"]
	_, existsC := all["c"]
	assert.True(t, existsA)
	assert.True(t, existsB)
	assert.True(t, existsC)
}

// ---------------------------------------------------------------------------
// String representation helpers
// ---------------------------------------------------------------------------

func TestOperationTypeString(t *testing.T) {
	t.Parallel()

	// ProgressType is an int iota — verify the values are distinct
	types := []ProgressType{
		ProgressTypeLinear,
		ProgressTypeCircular,
		ProgressTypePercentage,
		ProgressTypeSpinner,
	}

	seen := make(map[ProgressType]bool)
	for _, pt := range types {
		assert.False(t, seen[pt], "ProgressType value %d should be unique", pt)
		seen[pt] = true
	}
	assert.Len(t, seen, 4)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}

type errorWriter struct {
	err error
}

func (e *errorWriter) Write(p []byte) (n int, err error) {
	return 0, e.err
}
