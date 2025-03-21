package ws

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTopics(t *testing.T) {
	tt := newTopics()
	tt.Add("t1")
	assert.Equal(t, 1, tt.Size())
	assert.Equal(t, []string{"t1"}, tt.Values())
	tt.Add("t2")
	assert.Equal(t, 2, tt.Size())
	assert.True(t, tt.Contains("t1"))
	assert.True(t, tt.ContainsAny([]string{"t1", "t3"}))
	assert.False(t, tt.ContainsAny([]string{"t3"}))
	assert.False(t, tt.Contains("t3"))

	// test clone
	t2 := tt.Clone()
	assert.ElementsMatch(t, []string{"t1", "t2"}, t2.Values())

	// test merge
	t3 := newTopics()
	t3.Add("t3")
	tt.Merge(t3)
	assert.ElementsMatch(t, []string{"t1", "t2", "t3"}, tt.Values())

	// test remove
	tt.Remove("t1")
	assert.ElementsMatch(t, []string{"t2", "t3"}, tt.Values())

	// test clear
	tt.Clear()
	assert.Equal(t, 0, tt.Size())
}
