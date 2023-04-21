package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Ceil32(t *testing.T) {
	var testCases = []struct {
		Value    int64
		Expected int64
	}{
		{1, 32},
		{31, 32},
		{33, 64},
		{1023, 1024},
		{0, 0},
		{-1, -32},
		{-2, -32},
		{-66, -96},
	}
	for _, tc := range testCases {
		assert.Equal(t, tc.Expected, Ceil32(tc.Value))
	}
}
