package dialog

import "time"

// SessionLimiter decides whether a session's configured limit (time or
// message count) has been exceeded.
type SessionLimiter struct{}

func NewSessionLimiter() *SessionLimiter {
	return &SessionLimiter{}
}

// CheckLimit reports whether session may still receive messages. When
// allowed is false, reason names which limit was hit (EndReasonTimeLimit or
// EndReasonMessageLimit) so the caller can end the session accordingly.
func (l *SessionLimiter) CheckLimit(session Session) (allowed bool, reason string) {
	switch session.LimitType {
	case LimitTypeTime:
		if session.LimitValue == nil {
			return true, ""
		}
		elapsed := time.Since(session.StartedAt)
		limit := time.Duration(*session.LimitValue) * time.Second
		if elapsed >= limit {
			return false, EndReasonTimeLimit
		}
	case LimitTypeMessages:
		if session.LimitValue == nil {
			return true, ""
		}
		if session.MessageCount >= *session.LimitValue {
			return false, EndReasonMessageLimit
		}
	}
	return true, ""
}

// RemainingLimit reports how much of the session's limit is left, or nil if
// the session has no limit. For time limits this is whole seconds
// remaining; for message limits it's messages remaining.
func (l *SessionLimiter) RemainingLimit(session Session) *int32 {
	if session.LimitValue == nil {
		return nil
	}
	switch session.LimitType {
	case LimitTypeTime:
		limit := time.Duration(*session.LimitValue) * time.Second
		remaining := limit - time.Since(session.StartedAt)
		secs := int32(remaining.Seconds())
		if secs < 0 {
			secs = 0
		}
		return &secs
	case LimitTypeMessages:
		remaining := *session.LimitValue - session.MessageCount
		if remaining < 0 {
			remaining = 0
		}
		return &remaining
	}
	return nil
}
