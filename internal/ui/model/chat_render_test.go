package model

import (
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/ui/chat"
	"github.com/charmbracelet/crush/internal/ui/list"
	"github.com/charmbracelet/crush/internal/ui/styles"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/stretchr/testify/require"
)

// countingMessageItem is a finished three-line message that counts renders.
type countingMessageItem struct {
	*list.Versioned
	id      string
	renders int
}

func newCountingItems(count int) ([]*countingMessageItem, []chat.MessageItem) {
	items := make([]*countingMessageItem, 0, count)
	msgs := make([]chat.MessageItem, 0, count)
	for idx := range count {
		item := &countingMessageItem{Versioned: list.NewVersioned(), id: strconv.Itoa(idx)}
		items = append(items, item)
		msgs = append(msgs, item)
	}
	return items, msgs
}

func (m *countingMessageItem) ID() string { return m.id }

func (m *countingMessageItem) Render(int) string {
	m.renders++
	return strings.Repeat(m.id+"\n", 2) + m.id
}

func (m *countingMessageItem) RawRender(width int) string { return m.Render(width) }
func (m *countingMessageItem) Finished() bool             { return true }

func TestChatSetSize_ReservesScrollbarColumn(t *testing.T) {
	t.Parallel()

	cases := []struct {
		mode  string
		width int
	}{
		{config.ScrollbarDefault, 79},
		{config.ScrollbarAlways, 79},
		{config.ScrollbarNever, 80},
	}
	for _, tc := range cases {
		t.Run(tc.mode, func(t *testing.T) {
			t.Parallel()
			chatView := NewChat(newTestUI().com, tc.mode)
			chatView.SetSize(80, 10)
			require.Equal(t, tc.width, chatView.list.Width(), "empty chat")

			_, msgs := newCountingItems(20)
			chatView.SetMessages(msgs...)
			chatView.SetSize(80, 10)
			require.Equal(t, tc.width, chatView.list.Width(), "overflowing chat")
		})
	}
}

// TestChat_OverflowKeepsRenders pins that content growing past the viewport
// renders nothing again: the list width does not change at the threshold.
func TestChat_OverflowKeepsRenders(t *testing.T) {
	t.Parallel()

	const width, height = 80, 10
	chatView := NewChat(newTestUI().com, config.ScrollbarAlways)
	chatView.SetSize(width, height)
	scr := uv.NewScreenBuffer(width, height)
	items, msgs := newCountingItems(10)
	for _, msg := range msgs {
		chatView.AppendMessages(msg)
		chatView.ScrollToBottom()
		chatView.Draw(scr, scr.Bounds())
		chatView.SetSize(width, height)
		chatView.Draw(scr, scr.Bounds())
	}
	require.True(t, chatView.list.Overflows(height))
	for _, item := range items {
		require.Equal(t, 1, item.renders, "item %s", item.id)
	}
}

// TestApplyTheme_RendersOnlyVisibleItems pins that a theme swap keeps
// cached heights: the next frame, scrollbar included, renders only the
// items on screen.
func TestApplyTheme_RendersOnlyVisibleItems(t *testing.T) {
	t.Parallel()

	const width, height = 80, 10
	ui := newHostThemeTestUI(t, nil)
	ui.chat.scrollbarMode = config.ScrollbarAlways
	ui.chat.SetSize(width, height)
	items, msgs := newCountingItems(50)
	ui.chat.SetMessages(msgs...)
	scr := uv.NewScreenBuffer(width, height)
	ui.chat.Draw(scr, scr.Bounds())
	ui.chat.list.TotalHeight()
	before := make([]int, len(items))
	for idx, item := range items {
		before[idx] = item.renders
	}

	theme, err := styles.LoadTheme("gruvbox-dark")
	require.NoError(t, err)
	ui.applyTheme(theme)
	ui.chat.Draw(scr, scr.Bounds())

	start, end := ui.chat.list.VisibleItemIndices()
	require.Greater(t, start, 0)
	for idx, item := range items {
		want := before[idx]
		if idx >= start && idx <= end {
			want++
		}
		require.Equal(t, want, item.renders, "item %d", idx)
	}
}
