package file

import (
	"bytes"
	"io"
	"os"
	"testing"
	"time"

	"github.com/circonus-labs/circonus-unified-agent/internal"
	"github.com/circonus-labs/circonus-unified-agent/plugins/serializers"
	"github.com/circonus-labs/circonus-unified-agent/testutil"
	"github.com/stretchr/testify/assert"
)

const (
	expNewFile   = `{"value|ST[input_metric_group:test1,tag1:value1]": {"_value": 1, "_type": "L", "_ts": 1257894000000}}` + "\n"
	expExistFile = `{"cpu|ST[cpu:cpu0]": {"_value": 100, "_type": "L", "_ts": 1455312810012459582}}` + "\n" +
		`{"value|ST[input_metric_group:test1,tag1:value1]": {"_value": 1, "_type": "n", "_ts": 1257894000000}}` + "\n"
)

func TestFileExistingFile(t *testing.T) {
	fh := createFile()
	defer os.Remove(fh.Name())
	s, _ := serializers.NewCirconusSerializer(time.Millisecond)
	f := File{
		Files:      []string{fh.Name()},
		serializer: s,
	}

	err := f.Connect()
	assert.NoError(t, err)

	_, err = f.Write(testutil.MockMetrics())
	assert.NoError(t, err)

	validateFile(fh.Name(), expExistFile, t)

	err = f.Close()
	assert.NoError(t, err)
}

func TestFileNewFile(t *testing.T) {
	s, _ := serializers.NewCirconusSerializer(time.Millisecond)
	fh := tmpFile()
	defer os.Remove(fh)
	f := File{
		Files:      []string{fh},
		serializer: s,
	}

	err := f.Connect()
	assert.NoError(t, err)

	_, err = f.Write(testutil.MockMetrics())
	assert.NoError(t, err)

	validateFile(fh, expNewFile, t)

	err = f.Close()
	assert.NoError(t, err)
}

func TestFileExistingFiles(t *testing.T) {
	fh1 := createFile()
	defer os.Remove(fh1.Name())
	fh2 := createFile()
	defer os.Remove(fh2.Name())
	fh3 := createFile()
	defer os.Remove(fh3.Name())

	s, _ := serializers.NewCirconusSerializer(time.Millisecond)
	f := File{
		Files:      []string{fh1.Name(), fh2.Name(), fh3.Name()},
		serializer: s,
	}

	err := f.Connect()
	assert.NoError(t, err)

	_, err = f.Write(testutil.MockMetrics())
	assert.NoError(t, err)

	validateFile(fh1.Name(), expExistFile, t)
	validateFile(fh2.Name(), expExistFile, t)
	validateFile(fh3.Name(), expExistFile, t)

	err = f.Close()
	assert.NoError(t, err)
}

func TestFileNewFiles(t *testing.T) {
	s, _ := serializers.NewCirconusSerializer(time.Millisecond)
	fh1 := tmpFile()
	defer os.Remove(fh1)
	fh2 := tmpFile()
	defer os.Remove(fh2)
	fh3 := tmpFile()
	defer os.Remove(fh3)
	f := File{
		Files:      []string{fh1, fh2, fh3},
		serializer: s,
	}

	err := f.Connect()
	assert.NoError(t, err)

	_, err = f.Write(testutil.MockMetrics())
	assert.NoError(t, err)

	validateFile(fh1, expNewFile, t)
	validateFile(fh2, expNewFile, t)
	validateFile(fh3, expNewFile, t)

	err = f.Close()
	assert.NoError(t, err)
}

func TestFileBoth(t *testing.T) {
	fh1 := createFile()
	defer os.Remove(fh1.Name())
	fh2 := tmpFile()
	defer os.Remove(fh2)

	s, _ := serializers.NewCirconusSerializer(time.Millisecond)
	f := File{
		Files:      []string{fh1.Name(), fh2},
		serializer: s,
	}

	err := f.Connect()
	assert.NoError(t, err)

	_, err = f.Write(testutil.MockMetrics())
	assert.NoError(t, err)

	validateFile(fh1.Name(), expExistFile, t)
	validateFile(fh2, expNewFile, t)

	err = f.Close()
	assert.NoError(t, err)
}

func TestFileStdout(t *testing.T) {
	// keep backup of the real stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	s, _ := serializers.NewCirconusSerializer(time.Millisecond)
	f := File{
		Files:      []string{"stdout"},
		serializer: s,
	}

	err := f.Connect()
	assert.NoError(t, err)

	_, err = f.Write(testutil.MockMetrics())
	assert.NoError(t, err)

	err = f.Close()
	assert.NoError(t, err)

	outC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outC <- buf.String()
	}()

	// back to normal state
	w.Close()
	// restoring the real stdout
	os.Stdout = old
	out := <-outC

	assert.Equal(t, expNewFile, out)
}

func createFile() *os.File {
	f, err := os.CreateTemp("", "")
	if err != nil {
		panic(err)
	}
	_, _ = f.WriteString(`{"cpu|ST[cpu:cpu0]": {"_value": 100, "_type": "L", "_ts": 1455312810012459582}}` + "\n")
	return f
}

func tmpFile() string {
	d, err := os.MkdirTemp("", "")
	if err != nil {
		panic(err)
	}
	return d + internal.RandomString(10)
}

func validateFile(fname, expS string, t *testing.T) {
	buf, err := os.ReadFile(fname)
	if err != nil {
		panic(err)
	}
	assert.Equal(t, expS, string(buf))
}
