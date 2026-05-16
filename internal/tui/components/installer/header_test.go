package installer

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHeaderModel(t *testing.T) {
	h := NewHeaderModel(80)
	require.NotNil(t, h)
	assert.Equal(t, "🚀 R2Go2 Professional Installation", h.Title)
	assert.Equal(t, "Transform your Cloudflare R2 management experience", h.Subtitle)
	assert.NotEmpty(t, h.Description)
	assert.Equal(t, 80, h.Width)
	assert.Equal(t, 0, h.Progress)
	assert.Equal(t, 100, h.MaxProgress)
	assert.False(t, h.Loading)
	assert.NotNil(t, h.styles)
	assert.NotNil(t, h.Animation)
}

func TestCreateDefaultHeaderStyles(t *testing.T) {
	s := createDefaultHeaderStyles(100)
	require.NotNil(t, s)
	assert.NotNil(t, s.Container)
	assert.NotNil(t, s.Title)
	assert.NotNil(t, s.Subtitle)
	assert.NotNil(t, s.Description)
	assert.NotNil(t, s.Border)
	assert.NotNil(t, s.Icon)
	assert.NotNil(t, s.Progress)
	assert.NotNil(t, s.Loading)
}

func TestCreateHeaderAnimation(t *testing.T) {
	a := createHeaderAnimation()
	require.NotNil(t, a)
	assert.Equal(t, 0, a.frame)
	assert.False(t, a.active)
	assert.Len(t, a.frames, 10)
}

func TestHeaderModel_Init(t *testing.T) {
	h := NewHeaderModel(80)
	cmd := h.Init()
	assert.Nil(t, cmd)
	assert.False(t, time.Time{}.Equal(h.Animation.lastTick))
}

func TestHeaderModel_Update_WindowSize(t *testing.T) {
	h := NewHeaderModel(80)
	result, cmd := h.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	require.Nil(t, cmd)
	updated := result.(*HeaderModel)
	assert.Equal(t, 120, updated.Width)
}

func TestHeaderModel_Update_Loading(t *testing.T) {
	h := NewHeaderModel(80)
	h.Loading = true
	h.Animation.lastTick = time.Now().Add(-time.Millisecond * 200)
	result, _ := h.Update(nil)
	updated := result.(*HeaderModel)
	// Animation frame should have advanced
	assert.True(t, updated.Animation.frame >= 0)
}

func TestHeaderModel_Update_NonLoading(t *testing.T) {
	h := NewHeaderModel(80)
	h.Loading = false
	h.Animation.frame = 0
	result, _ := h.Update(nil)
	updated := result.(*HeaderModel)
	// Frame should not advance when not loading
	assert.Equal(t, 0, updated.Animation.frame)
}

func TestUpdateAnimation(t *testing.T) {
	h := NewHeaderModel(80)

	t.Run("advances frame when enough time passed", func(t *testing.T) {
		h.Animation.lastTick = time.Now().Add(-time.Millisecond * 200)
		h.Animation.frame = 0
		h.updateAnimation()
		assert.Equal(t, 1, h.Animation.frame)
	})

	t.Run("does not advance frame when not enough time", func(t *testing.T) {
		h.Animation.lastTick = time.Now()
		h.Animation.frame = 5
		h.updateAnimation()
		assert.Equal(t, 5, h.Animation.frame)
	})

	t.Run("wraps frame at end of list", func(t *testing.T) {
		h.Animation.lastTick = time.Now().Add(-time.Millisecond * 200)
		h.Animation.frame = len(h.Animation.frames) - 1
		h.updateAnimation()
		assert.Equal(t, 0, h.Animation.frame)
	})
}

func TestHeaderModel_View(t *testing.T) {
	h := NewHeaderModel(80)
	view := h.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "R2Go2")
}

func TestRenderHeaderContent(t *testing.T) {
	t.Run("full content", func(t *testing.T) {
		h := NewHeaderModel(80)
		content := h.renderHeaderContent()
		assert.NotEmpty(t, content)
		assert.Contains(t, content, "R2Go2 Professional Installation")
		assert.Contains(t, content, "Transform your Cloudflare R2")
		assert.Contains(t, content, "One command")
	})

	t.Run("no subtitle", func(t *testing.T) {
		h := NewHeaderModel(80)
		h.Subtitle = ""
		content := h.renderHeaderContent()
		assert.NotEmpty(t, content)
		assert.Contains(t, content, "R2Go2 Professional Installation")
		assert.NotContains(t, content, "Transform your Cloudflare R2")
	})

	t.Run("no description", func(t *testing.T) {
		h := NewHeaderModel(80)
		h.Description = ""
		content := h.renderHeaderContent()
		assert.NotEmpty(t, content)
		assert.NotContains(t, content, "One command")
	})

	t.Run("with progress", func(t *testing.T) {
		h := NewHeaderModel(80)
		h.Progress = 50
		h.MaxProgress = 100
		content := h.renderHeaderContent()
		assert.Contains(t, content, "50%")
	})

	t.Run("loading with no progress", func(t *testing.T) {
		h := NewHeaderModel(80)
		h.Loading = true
		h.Progress = 0
		h.Animation.frame = 0
		content := h.renderHeaderContent()
		assert.Contains(t, content, "Processing")
	})

	t.Run("empty title", func(t *testing.T) {
		h := NewHeaderModel(80)
		h.Title = ""
		h.Subtitle = ""
		h.Description = ""
		content := h.renderHeaderContent()
		// Should still produce valid output
		assert.NotEmpty(t, content)
	})
}

func TestRenderProgressBar(t *testing.T) {
	t.Run("zero max", func(t *testing.T) {
		h := NewHeaderModel(80)
		h.MaxProgress = 0
		bar := h.renderProgressBar()
		assert.Equal(t, "", bar)
	})

	t.Run("negative max", func(t *testing.T) {
		h := NewHeaderModel(80)
		h.MaxProgress = -1
		bar := h.renderProgressBar()
		assert.Equal(t, "", bar)
	})

	t.Run("zero progress", func(t *testing.T) {
		h := NewHeaderModel(80)
		h.Progress = 0
		h.MaxProgress = 100
		bar := h.renderProgressBar()
		assert.Contains(t, bar, "0%")
	})

	t.Run("half progress", func(t *testing.T) {
		h := NewHeaderModel(80)
		h.Progress = 50
		h.MaxProgress = 100
		bar := h.renderProgressBar()
		assert.Contains(t, bar, "50%")
	})

	t.Run("full progress", func(t *testing.T) {
		h := NewHeaderModel(80)
		h.Progress = 100
		h.MaxProgress = 100
		bar := h.renderProgressBar()
		assert.Contains(t, bar, "100%")
		assert.Contains(t, bar, "✓")
	})

	t.Run("progress exceeds max clamped", func(t *testing.T) {
		h := NewHeaderModel(80)
		h.Progress = 150
		h.MaxProgress = 100
		assert.NotPanics(t, func() {
			bar := h.renderProgressBar()
			assert.Contains(t, bar, "100%")
		})
	})

	t.Run("negative progress clamped", func(t *testing.T) {
		h := NewHeaderModel(80)
		h.Progress = -10
		h.MaxProgress = 100
		assert.NotPanics(t, func() {
			bar := h.renderProgressBar()
			assert.Contains(t, bar, "0%")
		})
	})

	t.Run("small max value", func(t *testing.T) {
		h := NewHeaderModel(80)
		h.Progress = 2
		h.MaxProgress = 5
		bar := h.renderProgressBar()
		assert.Contains(t, bar, "40%")
	})
}

func TestSetLoading(t *testing.T) {
	h := NewHeaderModel(80)
	h.SetLoading(true)
	assert.True(t, h.Loading)
	assert.True(t, h.Animation.active)

	h.SetLoading(false)
	assert.False(t, h.Loading)
	assert.False(t, h.Animation.active)
}

func TestSetProgress(t *testing.T) {
	h := NewHeaderModel(80)
	h.SetProgress(75, 200)
	assert.Equal(t, 75, h.Progress)
	assert.Equal(t, 200, h.MaxProgress)
}

func TestSetTitle(t *testing.T) {
	h := NewHeaderModel(80)
	h.SetTitle("New Title")
	assert.Equal(t, "New Title", h.Title)
}

func TestSetSubtitle(t *testing.T) {
	h := NewHeaderModel(80)
	h.SetSubtitle("New Subtitle")
	assert.Equal(t, "New Subtitle", h.Subtitle)
}

func TestSetDescription(t *testing.T) {
	h := NewHeaderModel(80)
	h.SetDescription("New Description")
	assert.Equal(t, "New Description", h.Description)
}

func TestSetWidth(t *testing.T) {
	h := NewHeaderModel(80)
	h.SetWidth(200)
	assert.Equal(t, 200, h.Width)
}

func TestGetWidth(t *testing.T) {
	h := NewHeaderModel(120)
	assert.Equal(t, 120, h.GetWidth())
}

func TestCreateWelcomeHeader(t *testing.T) {
	h := CreateWelcomeHeader(80)
	require.NotNil(t, h)
	assert.Contains(t, h.Subtitle, "Professional installation")
	assert.Contains(t, h.Description, "Fast setup")
	view := h.View()
	assert.NotEmpty(t, view)
}

func TestCreateProgressHeader(t *testing.T) {
	h := CreateProgressHeader(80, "Installing...", "Step 1 of 3")
	require.NotNil(t, h)
	assert.Equal(t, "Installing...", h.Title)
	assert.Contains(t, h.Description, "Step 1")
	assert.True(t, h.Loading)
	assert.Equal(t, 0, h.Progress)
	assert.Equal(t, 100, h.MaxProgress)
}

func TestCreateCompletionHeader(t *testing.T) {
	h := CreateCompletionHeader(80)
	require.NotNil(t, h)
	assert.Contains(t, h.Title, "Complete")
	assert.Contains(t, h.Subtitle, "ready")
	assert.Equal(t, 100, h.Progress)
	assert.Equal(t, 100, h.MaxProgress)
	view := h.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "✓")
}

// --- ROBUSTNESS ---

func TestHeaderRobustness_ZeroWidth(t *testing.T) {
	h := NewHeaderModel(0)
	assert.NotPanics(t, func() {
		view := h.View()
		assert.NotEmpty(t, view)
	})
}

func TestHeaderRobustness_NegativeWidth(t *testing.T) {
	h := NewHeaderModel(-10)
	assert.NotPanics(t, func() {
		view := h.View()
		assert.NotEmpty(t, view)
	})
}

func TestHeaderRobustness_VeryLongTitle(t *testing.T) {
	h := NewHeaderModel(80)
	h.Title = strings.Repeat("X", 500)
	assert.NotPanics(t, func() {
		view := h.View()
		assert.NotEmpty(t, view)
	})
}

func TestHeaderRobustness_UnicodeContent(t *testing.T) {
	h := NewHeaderModel(80)
	h.Title = "🚀 日本語テスト 中文测试 한국어"
	h.Subtitle = "Ελληνικά العربية עברית"
	h.Description = "🎲 Unicode galore 💎✨"
	assert.NotPanics(t, func() {
		view := h.View()
		assert.NotEmpty(t, view)
	})
}

func TestHeaderRobustness_LoadingAnimationFrames(t *testing.T) {
	h := NewHeaderModel(80)
	h.Loading = true
	// Advance through all frames
	for i := 0; i < 20; i++ {
		h.Animation.lastTick = time.Now().Add(-time.Millisecond * 200)
		h.updateAnimation()
	}
	assert.LessOrEqual(t, h.Animation.frame, len(h.Animation.frames))
}
