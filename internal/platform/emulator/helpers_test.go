package emulator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeCsv(t *testing.T, path, src string) string {
	t.Helper()

	fullPath := filepath.Join(t.TempDir(), path)
	err := os.WriteFile(fullPath, []byte(src), 0o644)
	require.NoError(t, err)
	return fullPath
}
