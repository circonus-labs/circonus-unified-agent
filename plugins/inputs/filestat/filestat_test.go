// +build !windows

package filestat

// TODO: Windows - should be enabled for Windows when super asterisk is fixed on Windows

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/circonus-labs/circonus-unified-agent/testutil"
	"github.com/stretchr/testify/assert"
)

var (
	testdataDir = getTestdataDir()
)

func TestGatherNoSHA256(t *testing.T) {
	fs := NewFileStat()
	fs.Log = testutil.Logger{}
	fs.Files = []string{
		filepath.Join(testdataDir, "log1.log"),
		filepath.Join(testdataDir, "log2.log"),
		filepath.Join(testdataDir, "non_existent_file"),
	}

	acc := testutil.Accumulator{}
	_ = acc.GatherError(fs.Gather)

	tags1 := map[string]string{
		"file": filepath.Join(testdataDir, "log1.log"),
	}
	require.True(t, acc.HasPoint("filestat", tags1, "size_bytes", int64(0)))
	require.True(t, acc.HasPoint("filestat", tags1, "exists", int64(1)))

	tags2 := map[string]string{
		"file": filepath.Join(testdataDir, "log2.log"),
	}
	require.True(t, acc.HasPoint("filestat", tags2, "size_bytes", int64(5)))
	require.True(t, acc.HasPoint("filestat", tags2, "exists", int64(1)))

	tags3 := map[string]string{
		"file": filepath.Join(testdataDir, "non_existent_file"),
	}
	require.True(t, acc.HasPoint("filestat", tags3, "exists", int64(0)))
}

func TestGatherExplicitFiles(t *testing.T) {
	fs := NewFileStat()
	fs.Log = testutil.Logger{}
	fs.SHA256 = true
	fs.Files = []string{
		filepath.Join(testdataDir, "log1.log"),
		filepath.Join(testdataDir, "log2.log"),
		filepath.Join(testdataDir, "non_existent_file"),
	}

	acc := testutil.Accumulator{}
	_ = acc.GatherError(fs.Gather)

	tags1 := map[string]string{
		"file": filepath.Join(testdataDir, "log1.log"),
	}
	require.True(t, acc.HasPoint("filestat", tags1, "size_bytes", int64(0)))
	require.True(t, acc.HasPoint("filestat", tags1, "exists", int64(1)))
	require.True(t, acc.HasPoint("filestat", tags1, "sha256_sum", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"))

	tags2 := map[string]string{
		"file": filepath.Join(testdataDir, "log2.log"),
	}
	require.True(t, acc.HasPoint("filestat", tags2, "size_bytes", int64(5)))
	require.True(t, acc.HasPoint("filestat", tags2, "exists", int64(1)))
	require.True(t, acc.HasPoint("filestat", tags2, "sha256_sum", "7c4604d03f399eac32a48edbb7be1710838b70c83ad0e94b60137920945d6c40"))

	tags3 := map[string]string{
		"file": filepath.Join(testdataDir, "non_existent_file"),
	}
	require.True(t, acc.HasPoint("filestat", tags3, "exists", int64(0)))
}

func TestGatherGlob(t *testing.T) {
	fs := NewFileStat()
	fs.Log = testutil.Logger{}
	fs.SHA256 = true
	fs.Files = []string{
		filepath.Join(testdataDir, "*.log"),
	}

	acc := testutil.Accumulator{}
	_ = acc.GatherError(fs.Gather)

	tags1 := map[string]string{
		"file": filepath.Join(testdataDir, "log1.log"),
	}
	require.True(t, acc.HasPoint("filestat", tags1, "size_bytes", int64(0)))
	require.True(t, acc.HasPoint("filestat", tags1, "exists", int64(1)))
	require.True(t, acc.HasPoint("filestat", tags1, "sha256_sum", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"))

	tags2 := map[string]string{
		"file": filepath.Join(testdataDir, "log2.log"),
	}
	require.True(t, acc.HasPoint("filestat", tags2, "size_bytes", int64(5)))
	require.True(t, acc.HasPoint("filestat", tags2, "exists", int64(1)))
	require.True(t, acc.HasPoint("filestat", tags2, "sha256_sum", "7c4604d03f399eac32a48edbb7be1710838b70c83ad0e94b60137920945d6c40"))
}

func TestGatherSuperAsterisk(t *testing.T) {
	fs := NewFileStat()
	fs.Log = testutil.Logger{}
	fs.SHA256 = true
	fs.Files = []string{
		filepath.Join(testdataDir, "**"),
	}

	acc := testutil.Accumulator{}
	_ = acc.GatherError(fs.Gather)

	tags1 := map[string]string{
		"file": filepath.Join(testdataDir, "log1.log"),
	}
	require.True(t, acc.HasPoint("filestat", tags1, "size_bytes", int64(0)))
	require.True(t, acc.HasPoint("filestat", tags1, "exists", int64(1)))
	require.True(t, acc.HasPoint("filestat", tags1, "sha256_sum", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"))

	tags2 := map[string]string{
		"file": filepath.Join(testdataDir, "log2.log"),
	}
	require.True(t, acc.HasPoint("filestat", tags2, "size_bytes", int64(5)))
	require.True(t, acc.HasPoint("filestat", tags2, "exists", int64(1)))
	require.True(t, acc.HasPoint("filestat", tags2, "sha256_sum", "7c4604d03f399eac32a48edbb7be1710838b70c83ad0e94b60137920945d6c40"))

	tags3 := map[string]string{
		"file": filepath.Join(testdataDir, "test.conf"),
	}
	require.True(t, acc.HasPoint("filestat", tags3, "size_bytes", int64(104)))
	require.True(t, acc.HasPoint("filestat", tags3, "exists", int64(1)))
	require.True(t, acc.HasPoint("filestat", tags3, "sha256_sum", "9b17fa34411e1ee1f1795b0e326f187ba7acde5ab6fc8e011ff3b0a550f9dbe2"))
}

func TestModificationTime(t *testing.T) {
	fs := NewFileStat()
	fs.Log = testutil.Logger{}
	fs.Files = []string{
		filepath.Join(testdataDir, "log1.log"),
	}

	acc := testutil.Accumulator{}
	_ = acc.GatherError(fs.Gather)

	tags1 := map[string]string{
		"file": filepath.Join(testdataDir, "log1.log"),
	}
	require.True(t, acc.HasPoint("filestat", tags1, "size_bytes", int64(0)))
	require.True(t, acc.HasPoint("filestat", tags1, "exists", int64(1)))
	require.True(t, acc.HasInt64Field("filestat", "modification_time"))
}

func TestNoModificationTime(t *testing.T) {
	fs := NewFileStat()
	fs.Log = testutil.Logger{}
	fs.Files = []string{
		filepath.Join(testdataDir, "non_existent_file"),
	}

	acc := testutil.Accumulator{}
	_ = acc.GatherError(fs.Gather)

	tags1 := map[string]string{
		"file": filepath.Join(testdataDir, "non_existent_file"),
	}
	require.True(t, acc.HasPoint("filestat", tags1, "exists", int64(0)))
	require.False(t, acc.HasInt64Field("filestat", "modification_time"))
}

func TestGetSHA256(t *testing.T) {
	sig, err := getSHA256(filepath.Join(testdataDir, "test.conf"))
	assert.NoError(t, err)
	assert.Equal(t, "9b17fa34411e1ee1f1795b0e326f187ba7acde5ab6fc8e011ff3b0a550f9dbe2", sig)

	_, err = getSHA256("/tmp/foo/bar/fooooo")
	assert.Error(t, err)
}

func getTestdataDir() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return filepath.Join(dir, "testdata")
}
