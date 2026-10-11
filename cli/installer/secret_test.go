package installer

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetOrCreateSecret_Stable(t *testing.T) {
	file := path.Join(t.TempDir(), ".secret")

	first, err := getOrCreateSecret(file)
	assert.NoError(t, err)
	assert.Len(t, first, 64)

	second, err := getOrCreateSecret(file)
	assert.NoError(t, err)
	assert.Equal(t, first, second)
}
