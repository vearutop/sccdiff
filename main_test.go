package main

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test(t *testing.T) {
	old := os.Stdout // keep backup of the real stdout

	r, w, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = w

	defer func() { os.Stdout = old }()

	buf := bytes.NewBuffer(nil)
	done := make(chan struct{})

	go func() {
		_, err := io.Copy(buf, r)
		require.NoError(t, err)

		close(done)
	}()

	main()
	require.NoError(t, w.Close())

	<-done

	assert.NotEmpty(t, buf.String())
}
