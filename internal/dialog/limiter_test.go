package dialog

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func int32Ptr(v int32) *int32 { return &v }

func TestSessionLimiter_CheckLimit_NoLimit(t *testing.T) {
	l := NewSessionLimiter()
	session := Session{LimitType: LimitTypeNone, StartedAt: time.Now().Add(-time.Hour)}

	allowed, reason := l.CheckLimit(session)

	assert.True(t, allowed)
	assert.Empty(t, reason)
}

func TestSessionLimiter_CheckLimit_MessagesUnderLimit(t *testing.T) {
	l := NewSessionLimiter()
	session := Session{LimitType: LimitTypeMessages, LimitValue: int32Ptr(10), MessageCount: 5}

	allowed, reason := l.CheckLimit(session)

	assert.True(t, allowed)
	assert.Empty(t, reason)
}

func TestSessionLimiter_CheckLimit_MessagesAtLimit(t *testing.T) {
	l := NewSessionLimiter()
	session := Session{LimitType: LimitTypeMessages, LimitValue: int32Ptr(10), MessageCount: 10}

	allowed, reason := l.CheckLimit(session)

	assert.False(t, allowed)
	assert.Equal(t, EndReasonMessageLimit, reason)
}

func TestSessionLimiter_CheckLimit_TimeExpired(t *testing.T) {
	l := NewSessionLimiter()
	session := Session{LimitType: LimitTypeTime, LimitValue: int32Ptr(60), StartedAt: time.Now().Add(-2 * time.Minute)}

	allowed, reason := l.CheckLimit(session)

	assert.False(t, allowed)
	assert.Equal(t, EndReasonTimeLimit, reason)
}

func TestSessionLimiter_CheckLimit_TimeNotYetExpired(t *testing.T) {
	l := NewSessionLimiter()
	session := Session{LimitType: LimitTypeTime, LimitValue: int32Ptr(600), StartedAt: time.Now()}

	allowed, reason := l.CheckLimit(session)

	assert.True(t, allowed)
	assert.Empty(t, reason)
}

func TestSessionLimiter_RemainingLimit_Messages(t *testing.T) {
	l := NewSessionLimiter()
	session := Session{LimitType: LimitTypeMessages, LimitValue: int32Ptr(10), MessageCount: 4}

	remaining := l.RemainingLimit(session)

	require := assert.New(t)
	require.NotNil(remaining)
	require.Equal(int32(6), *remaining)
}

func TestSessionLimiter_RemainingLimit_NilWhenUnlimited(t *testing.T) {
	l := NewSessionLimiter()
	session := Session{LimitType: LimitTypeNone}

	assert.Nil(t, l.RemainingLimit(session))
}
