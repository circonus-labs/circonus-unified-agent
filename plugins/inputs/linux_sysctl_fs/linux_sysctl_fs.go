package linuxsysctlfs

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"strconv"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
)

// https://www.kernel.org/doc/Documentation/sysctl/fs.txt
type SysctlFS struct {
	path string
}

var sysctlFSDescription = `Provides Linux sysctl fs metrics`
var sysctlFSSampleConfig = `
  instance_id = "" # unique instance identifier (REQUIRED)
`

func (*SysctlFS) Description() string {
	return sysctlFSDescription
}
func (*SysctlFS) SampleConfig() string {
	return sysctlFSSampleConfig
}

func (sfs *SysctlFS) gatherList(file string, fields map[string]interface{}, fieldNames ...string) error {
	bs, err := os.ReadFile(sfs.path + "/" + file)
	if err != nil {
		return fmt.Errorf("readfile: %w", err)
	}

	bsplit := bytes.Split(bytes.TrimRight(bs, "\n"), []byte{'\t'})
	for i, name := range fieldNames {
		if i >= len(bsplit) {
			break
		}
		if name == "" {
			continue
		}

		v, err := strconv.ParseUint(string(bsplit[i]), 10, 64)
		if err != nil {
			return fmt.Errorf("parseuint (%s): %w", string(bsplit[i]), err)
		}
		fields[name] = v
	}

	return nil
}

func (sfs *SysctlFS) gatherOne(name string, fields map[string]interface{}) error {
	bs, err := os.ReadFile(sfs.path + "/" + name)
	if err != nil {
		return fmt.Errorf("readfile: %w", err)
	}

	v, err := strconv.ParseUint(string(bytes.TrimRight(bs, "\n")), 10, 64)
	if err != nil {
		return fmt.Errorf("parseuint (%s): %w", string(bytes.TrimRight(bs, "\n")), err)
	}

	fields[name] = v
	return nil
}

func (sfs *SysctlFS) Gather(ctx context.Context, acc cua.Accumulator) error {
	fields := map[string]interface{}{}

	for _, n := range []string{"aio-nr", "aio-max-nr", "dquot-nr", "dquot-max", "super-nr", "super-max"} {
		_ = sfs.gatherOne(n, fields)
	}

	_ = sfs.gatherList("inode-state", fields, "inode-nr", "inode-free-nr", "inode-preshrink-nr")
	_ = sfs.gatherList("dentry-state", fields, "dentry-nr", "dentry-unused-nr", "dentry-age-limit", "dentry-want-pages")
	_ = sfs.gatherList("file-nr", fields, "file-nr", "", "file-max")

	acc.AddFields("linux_sysctl_fs", fields, nil)
	return nil
}

func GetHostProc() string {
	procPath := "/proc"
	if os.Getenv("HOST_PROC") != "" {
		procPath = os.Getenv("HOST_PROC")
	}
	return procPath
}

func init() {

	inputs.Add("linux_sysctl_fs", func() cua.Input {
		return &SysctlFS{
			path: path.Join(GetHostProc(), "/sys/fs"),
		}
	})
}
