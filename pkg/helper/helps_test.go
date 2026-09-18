package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmpty(t *testing.T) {
	assert := assert.New(t)
	assert.True(Empty(0.0))
	assert.True(Empty(0))
	assert.True(Empty(""))
	assert.True(Empty(nil))
	assert.True(Empty(map[string]interface{}{}))
	assert.True(Empty([]string{}))
	assert.False(Empty(' '))
}
