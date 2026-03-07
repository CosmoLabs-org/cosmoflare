package visual

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestQuietMode(t *testing.T) {
	t.Run("DisableAnimations suppresses all visual output", func(t *testing.T) {
		DisableAnimations()
		defer EnableAnimations()

		// These should return instantly without sleeping
		start := time.Now()
		ShowStartupAnimation()
		ShowSuccess("test success")
		ShowError("test error")
		ShowSpinner("test", 5*time.Second)
		ShowResult("title", map[string]interface{}{"key": "val"})
		elapsed := time.Since(start)

		// With animations disabled, all calls should complete in under 100ms
		assert.Less(t, elapsed, 100*time.Millisecond, "animations should be suppressed in quiet mode")
	})

	t.Run("EnableAnimations re-enables output", func(t *testing.T) {
		DisableAnimations()
		assert.True(t, IsQuiet())
		EnableAnimations()
		assert.False(t, IsQuiet())
	})
}
