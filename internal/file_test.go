package internal

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestNewFileParsing(t *testing.T) {
	file, err := NewFile(afero.NewOsFs(), "testdata/sample.md")

	require.NoError(t, err)

	require.NotEmpty(t, file.Title, "title should be populated")
	require.Equal(t, "Sample file for testing purpose", file.Title)

	require.NotEmpty(t, file.Attachments, "attachments should be populated")
	require.Len(t, file.Attachments, 1, "exactly one attachment")
	require.Contains(t, file.Attachments, "images/sample.png")
}
