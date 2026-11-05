package bot

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func RequireSentCount(tb testing.TB, spy *ClientSpy, want int) {
	tb.Helper()
	require.Len(tb, spy.SentCalls(), want)
}

func RequireLastSentText(tb testing.TB, spy *ClientSpy, want string) {
	tb.Helper()
	msg, ok := spy.LastSentMessage()
	require.True(tb, ok, "expected at least one SendMessage call")
	require.Equal(tb, want, msg.Text)
}
