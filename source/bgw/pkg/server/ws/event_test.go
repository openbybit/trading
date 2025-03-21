package ws

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAction(t *testing.T) {
	at := ActionSessionOnline
	assert.True(t, at.IsAny(ActionSessionOnline, ActionSessionOffline))
	assert.False(t, at.IsAny(ActionSessionSub))

	a := newAction(ActionSessionOnline, 1, "", nil)
	assert.Equal(t, int64(1), a.UserID)
}

func TestEvent(t *testing.T) {
	e1 := NewForceSyncUserEvent(1, "")
	assert.Equal(t, EventTypeForceSyncUser, e1.Type())
	e2 := NewSyncOneUserEvent(newUser(1), nil)
	assert.Equal(t, EventTypeSyncOneUser, e2.Type())
	e4 := NewSyncAllUserEvent("")
	assert.Equal(t, EventTypeSyncAllUser, e4.Type())
	e5 := NewSyncConfigEvent("")
	assert.Equal(t, EventTypeSyncConfig, e5.Type())
	e6 := SyncInputEvent{}
	assert.Equal(t, EventTypeSyncInput, e6.Type())
}
