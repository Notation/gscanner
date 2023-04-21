package solidity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_replaceAddress(t *testing.T) {
	var testCases = []struct {
		Code     string
		Expected string
	}{
		{
			"(__aa.dddddddddddddddddddddddddddddddddddddd)",
			"(aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaddd)",
		},
		{
			"(__55.999ddddddddddddddddddddddddddddd123)",
			"(aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa)",
		},
	}
	for _, tc := range testCases {
		assert.Equal(t, tc.Expected, replaceAddress(tc.Code))
	}
}
