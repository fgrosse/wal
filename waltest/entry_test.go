package waltest

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExampleEntry1(t *testing.T) {
	original := &ExampleEntry1{
		ID:    5546132,
		Point: []float32{1, 2, 3, 4, 5},
	}

	assert.Equal(t, ExampleEntry1Type, original.Type())

	var encoded []byte
	encoded = original.EncodePayload(encoded)
	r := bytes.NewBuffer(encoded)

	decoded := new(ExampleEntry1)
	input, err := decoded.ReadPayload(r)
	require.NoError(t, err)

	err = decoded.DecodePayload(input)
	require.NoError(t, err)

	assert.Equal(t, original, decoded)
}

func TestExampleEntry2(t *testing.T) {
	original := &ExampleEntry2{
		Test: true,
		Name: "Ada Lovelace",
	}

	var encoded []byte
	encoded = original.EncodePayload(encoded)
	r := bytes.NewBuffer(encoded)

	decoded := new(ExampleEntry2)
	input, err := decoded.ReadPayload(r)
	require.NoError(t, err)

	err = decoded.DecodePayload(input)
	require.NoError(t, err)

	assert.Equal(t, original, decoded)
}
