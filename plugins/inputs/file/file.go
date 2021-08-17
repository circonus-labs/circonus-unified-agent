package file

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/internal/globpath"
	"github.com/circonus-labs/circonus-unified-agent/plugins/common/encoding"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
	"github.com/circonus-labs/circonus-unified-agent/plugins/parsers"
	"github.com/dimchansky/utfbom"
)

type File struct {
	parser            parsers.Parser
	decoder           *encoding.Decoder
	FileTag           string   `toml:"file_tag"`
	CharacterEncoding string   `toml:"character_encoding"`
	Files             []string `toml:"files"`
	filenames         []string
}

const sampleConfig = `
  ## Files to parse each interval.  Accept standard unix glob matching rules,
  ## as well as ** to match recursive files and directories.
  files = ["/tmp/metrics.out"]

  ## Name a tag containing the name of the file the data was parsed from.  Leave empty
  ## to disable.
  # file_tag = ""

  ## Character encoding to use when interpreting the file contents.  Invalid
  ## characters are replaced using the unicode replacement character.  When set
  ## to the empty string the data is not decoded to text.
  ##   ex: character_encoding = "utf-8"
  ##       character_encoding = "utf-16le"
  ##       character_encoding = "utf-16be"
  ##       character_encoding = ""
  # character_encoding = ""

  ## The dataformat to be read from files
  ## Each data format has its own unique set of configuration options, read
  ## more about them here:
  ## https://github.com/circonus-labs/circonus-unified-agent/blob/master/docs/DATA_FORMATS_INPUT.md
  data_format = "influx"
`

// SampleConfig returns the default configuration of the Input
func (f *File) SampleConfig() string {
	return sampleConfig
}

func (f *File) Description() string {
	return "Parse a complete file each interval"
}

func (f *File) Init() error {
	var err error
	f.decoder, err = encoding.NewDecoder(f.CharacterEncoding)
	return fmt.Errorf("new decoder: %w", err)
}

func (f *File) Gather(ctx context.Context, acc cua.Accumulator) error {
	err := f.refreshFilePaths()
	if err != nil {
		return err
	}
	for _, k := range f.filenames {
		metrics, err := f.readMetric(k)
		if err != nil {
			return err
		}

		for _, m := range metrics {
			if f.FileTag != "" {
				m.AddTag(f.FileTag, filepath.Base(k))
			}
			acc.AddMetric(m)
		}
	}
	return nil
}

func (f *File) SetParser(p parsers.Parser) {
	f.parser = p
}

func (f *File) refreshFilePaths() error {
	var allFiles []string //nolint:prealloc
	for _, file := range f.Files {
		g, err := globpath.Compile(file)
		if err != nil {
			return fmt.Errorf("could not compile glob %v: %w", file, err)
		}
		files := g.Match()
		if len(files) == 0 {
			return fmt.Errorf("could not find file: %v", file)
		}
		allFiles = append(allFiles, files...)
	}

	f.filenames = allFiles
	return nil
}

func (f *File) readMetric(filename string) ([]cua.Metric, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open (%s): %w", filename, err)
	}
	defer file.Close()

	r, _ := utfbom.Skip(f.decoder.Reader(file))
	fileContents, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("E! Error file: %s could not be read, %w", filename, err)
	}
	return f.parser.Parse(fileContents)
}

func init() {
	inputs.Add("file", func() cua.Input {
		return &File{}
	})
}
