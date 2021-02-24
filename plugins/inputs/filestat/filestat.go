package filestat

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/internal/globpath"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
)

const sampleConfig = `
  ## Files to gather stats about.
  ## These accept standard unix glob matching rules, but with the addition of
  ## ** as a "super asterisk". ie:
  ##   "/var/log/**.log"  -> recursively find all .log files in /var/log
  ##   "/var/log/*/*.log" -> find all .log files with a parent dir in /var/log
  ##   "/var/log/apache.log" -> just tail the apache log file
  ##
  ## See https://github.com/gobwas/glob for more examples
  ##
  files = ["/var/log/**.log"]

  ## If true, read the entire file and calculate an sha256 checksum.
  sha256 = false
`

type FileStat struct {
	SHA256 bool
	Files  []string

	Log cua.Logger

	// maps full file paths to globmatch obj
	globs map[string]*globpath.GlobPath
}

func NewFileStat() *FileStat {
	return &FileStat{
		globs: make(map[string]*globpath.GlobPath),
	}
}

func (*FileStat) Description() string {
	return "Read stats about given file(s)"
}

func (*FileStat) SampleConfig() string { return sampleConfig }

func (f *FileStat) Gather(acc cua.Accumulator) error {
	var err error

	for _, filepath := range f.Files {
		// Get the compiled glob object for this filepath
		g, ok := f.globs[filepath]
		if !ok {
			if g, err = globpath.Compile(filepath); err != nil {
				acc.AddError(err)
				continue
			}
			f.globs[filepath] = g
		}

		files := g.Match()
		if len(files) == 0 {
			acc.AddFields("filestat",
				map[string]interface{}{
					"exists": int64(0),
				},
				map[string]string{
					"file": filepath,
				})
			continue
		}

		for _, fileName := range files {
			tags := map[string]string{
				"file": fileName,
			}
			fields := map[string]interface{}{
				"exists": int64(1),
			}
			fileInfo, err := os.Stat(fileName)
			if os.IsNotExist(err) {
				fields["exists"] = int64(0)
			}

			if fileInfo == nil {
				f.Log.Errorf("Unable to get info for file %q, possible permissions issue",
					fileName)
			} else {
				fields["size_bytes"] = fileInfo.Size()
				fields["modification_time"] = fileInfo.ModTime().UnixNano()
			}

			if f.SHA256 {
				sig, err := getSHA256(fileName)
				if err != nil {
					acc.AddError(err)
				} else {
					fields["sha256_sum"] = sig
				}
			}

			acc.AddFields("filestat", fields, tags)
		}
	}

	return nil
}

// // Read given file and calculate an md5 hash.
// func getMd5(file string) (string, error) {
// 	of, err := os.Open(file)
// 	if err != nil {
// 		return "", err
// 	}
// 	defer of.Close()

// 	hash := md5.New()
// 	_, err = io.Copy(hash, of)
// 	if err != nil {
// 		// fatal error
// 		return "", err
// 	}
// 	return fmt.Sprintf("%x", hash.Sum(nil)), nil
// }

// Read given file and calculate an sha256 hash.
func getSHA256(file string) (string, error) {
	of, err := os.Open(file)
	if err != nil {
		return "", fmt.Errorf("open: %w", err)
	}
	defer of.Close()

	hash := sha256.New()
	_, err = io.Copy(hash, of)
	if err != nil {
		// fatal error
		return "", fmt.Errorf("copy: %w", err)
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func init() {
	inputs.Add("filestat", func() cua.Input {
		return NewFileStat()
	})
}
