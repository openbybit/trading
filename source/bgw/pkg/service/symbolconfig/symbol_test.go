package symbolconfig

import (
	"github.com/tj/assert"
	"testing"
)

func TestGetManager(t *testing.T) {
	mgr := GetFutureManager()
	sc, _ := GetSymbolConfig()
	assert.NotNil(t, mgr)
	assert.NotNil(t, sc)
}
