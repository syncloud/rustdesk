package server

import (
	"encoding/base64"
	"encoding/json"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientConfig_EncodeMatchesRustDeskClientExport(t *testing.T) {
	config := ClientConfig{Host: "device.example.com", Key: "Wu6lAXTYIs4bIQK+5vE8WRiHLy0VcKKinut5JWx+Wcw="}

	encoded, err := config.Encode()
	require.NoError(t, err)

	assert.Equal(t,
		"9JSP3N2Vrg3VKVDd15WaLt0YWBTeMhUaSdFOFZXNrsUUJJGNzlUWUhVQsZTdXJiOikXZrJCLiIiOikGchJCLiIiOikXYsVmciwiIt92YuUGbw1WY4VmLlNWa2VGZiojI0N3boJye",
		encoded)
}

func TestClientConfig_EncodeIsReversedBase64Json(t *testing.T) {
	config := ClientConfig{Host: "device.example.com", Relay: "relay.example.com", Key: "key"}

	encoded, err := config.Encode()
	require.NoError(t, err)

	reversed := []byte(encoded)
	slices.Reverse(reversed)
	body, err := base64.URLEncoding.DecodeString(string(reversed))
	require.NoError(t, err)

	var decoded ClientConfig
	require.NoError(t, json.Unmarshal(body, &decoded))
	assert.Equal(t, config, decoded)
}
