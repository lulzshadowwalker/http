package headers_test

import (
	"testing"

	pkg "github.com/lulzshadowwalker/http/internal/headers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeaders(t *testing.T) {
	// Test: Valid single header
	headers := pkg.NewHeaders()
	data := []byte("Host: localhost:42069\r\n\r\n")
	n, done, err := headers.Parse(data)
	require.NoError(t, err)
	require.NotNil(t, headers)
	assert.Equal(t, "localhost:42069", headers["host"])
	assert.Equal(t, 23, n)
	assert.False(t, done)

	// Test: Valid single header with capital letters in key
	headers = pkg.NewHeaders()
	data = []byte("CONTENT-TYPE: application/json\r\n\r\n")
	n, done, err = headers.Parse(data)
	require.NoError(t, err)
	assert.Equal(t, "application/json", headers["content-type"])
	assert.Equal(t, 32, n)
	assert.False(t, done)

	// Test: Valid single header with mixed case key
	headers = pkg.NewHeaders()
	data = []byte("Content-TyPe: application/json\r\n\r\n")
	n, done, err = headers.Parse(data)
	require.NoError(t, err)
	assert.Equal(t, "application/json", headers["content-type"])
	assert.Equal(t, 32, n)
	assert.False(t, done)

	// Test: Valid single header with extra whitespace
	headers = pkg.NewHeaders()
	data = []byte("Host:           localhost:42069    \r\n\r\n")
	n, done, err = headers.Parse(data)
	require.NoError(t, err)
	assert.Equal(t, "localhost:42069", headers["host"])
	assert.Equal(t, 37, n)
	assert.False(t, done)

	// Test: Valid 2 headers with existing headers
	headers = pkg.NewHeaders()
	headers["existing"] = "existing-value"
	data = []byte("Host: localhost:42069\r\nContent-Type: application/json\r\n\r\n")
	n, done, err = headers.Parse(data)
	require.NoError(t, err)
	assert.Equal(t, "localhost:42069", headers["host"])
	assert.Equal(t, "existing-value", headers["existing"])
	assert.Equal(t, 23, n)
	assert.False(t, done)

	// Parse the second header from the remaining data
	n, done, err = headers.Parse(data[n:])
	require.NoError(t, err)
	assert.Equal(t, "application/json", headers["content-type"])
	assert.Equal(t, 32, n)
	assert.False(t, done)

	// Test: Valid done
	headers = pkg.NewHeaders()
	data = []byte("\r\n\r\n")
	n, done, err = headers.Parse(data)
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.True(t, done)

	// Test: Invalid spacing header (leading whitespace)
	headers = pkg.NewHeaders()
	data = []byte("       Host: localhost:42069\r\n\r\n")
	n, done, err = headers.Parse(data)
	require.Error(t, err)
	assert.Equal(t, 0, n)
	assert.False(t, done)

	// Test: Invalid spacing header (space before colon)
	headers = pkg.NewHeaders()
	data = []byte("Host : localhost:42069\r\n\r\n")
	n, done, err = headers.Parse(data)
	require.Error(t, err)
	assert.Equal(t, 0, n)
	assert.False(t, done)

	// Test: Invalid spacing header (tab before colon)
	headers = pkg.NewHeaders()
	data = []byte("Host\t: localhost:42069\r\n\r\n")
	n, done, err = headers.Parse(data)
	require.Error(t, err)
	assert.Equal(t, 0, n)
	assert.False(t, done)

	// Test: Invalid character in header key
	headers = pkg.NewHeaders()
	data = []byte("H©st: localhost:42069\r\n\r\n")
	n, done, err = headers.Parse(data)
	require.Error(t, err)
	assert.Equal(t, 0, n)
	assert.False(t, done)

	// Test: No CRLF yet - need more data
	headers = pkg.NewHeaders()
	data = []byte("Host: localhost:42069")
	n, done, err = headers.Parse(data)
	require.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.False(t, done)
	assert.Empty(t, headers)

	// Test: Missing colon
	headers = pkg.NewHeaders()
	data = []byte("Host localhost42069\r\n\r\n")
	n, done, err = headers.Parse(data)
	require.Error(t, err)
	assert.Equal(t, 0, n)
	assert.False(t, done)

	// Test: Empty field name
	headers = pkg.NewHeaders()
	data = []byte(": localhost:42069\r\n\r\n")
	n, done, err = headers.Parse(data)
	require.Error(t, err)
	assert.Equal(t, 0, n)
	assert.False(t, done)

	// Test: Empty field value is allowed
	headers = pkg.NewHeaders()
	data = []byte("X-Empty:\r\n\r\n")
	n, done, err = headers.Parse(data)
	require.NoError(t, err)
	assert.Equal(t, "", headers["x-empty"])
	assert.Equal(t, 10, n)
	assert.False(t, done)

	// Test: Multiple calls accumulate headers
	headers = pkg.NewHeaders()
	data = []byte("Host: localhost\r\nAccept: text/html\r\nUser-Agent: test\r\n\r\n")
	var total int
	for {
		n, done, err = headers.Parse(data[total:])
		require.NoError(t, err)
		total += n
		if done {
			break
		}
	}
	assert.Equal(t, "localhost", headers["host"])
	assert.Equal(t, "text/html", headers["accept"])
	assert.Equal(t, "test", headers["user-agent"])
	assert.Equal(t, len(data), total)

	// Test: Valid header with existing key appends value
	headers = pkg.NewHeaders()
	headers["set-person"] = "lane-loves-go"
	data = []byte("Set-Person: prime-loves-zig\r\n\r\n")
	n, done, err = headers.Parse(data)
	require.NoError(t, err)
	assert.Equal(t, "lane-loves-go, prime-loves-zig", headers["set-person"])
	assert.Equal(t, 29, n)
	assert.False(t, done)
}
