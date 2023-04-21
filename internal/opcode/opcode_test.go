package opcode

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOPCode(t *testing.T) {
	assert.Equal(t, 82+32+32, len(opCodeInfos))
	assert.Equal(t, 82+32+32, len(opCodes))
}
