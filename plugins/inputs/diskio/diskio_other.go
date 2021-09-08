//go:build !linux
// +build !linux

package diskio

type diskInfoCache struct{} //nolint:unused

func (s *DiskIO) diskInfo(devName string) (map[string]string, error) {
	return nil, nil
}
