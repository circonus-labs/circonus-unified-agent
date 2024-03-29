// Go API over pdh syscalls
//go:build windows
// +build windows

package winperfcounters

import (
	"errors"
	"syscall"
	"time"
	"unsafe"
)

// PerformanceQuery is abstraction for PDH_FMT_COUNTERVALUE_ITEM_DOUBLE
type CounterValue struct {
	InstanceName string
	Value        float64
}

// PerformanceQuery provides wrappers around Windows performance counters API for easy usage in GO
type PerformanceQuery interface {
	Open() error
	Close() error
	AddCounterToQuery(counterPath string) (PdhHCounter, error)
	AddEnglishCounterToQuery(counterPath string) (PdhHCounter, error)
	GetCounterPath(counterHandle PdhHCounter) (string, error)
	ExpandWildCardPath(counterPath string) ([]string, error)
	GetFormattedCounterValueDouble(hCounter PdhHCounter) (float64, error)
	GetFormattedCounterArrayDouble(hCounter PdhHCounter) ([]CounterValue, error)
	CollectData() error
	CollectDataWithTime() (time.Time, error)
	IsVistaOrNewer() bool
}

// PdhError represents error returned from Performance Counters API
type PdhError struct {
	ErrorCode uint32
	errorText string
}

func (m *PdhError) Error() string {
	return m.errorText
}

func NewPdhError(code uint32) error {
	return &PdhError{
		ErrorCode: code,
		errorText: PdhFormatError(code),
	}
}

// PerformanceQueryImpl is implementation of PerformanceQuery interface, which calls phd.dll functions
type PerformanceQueryImpl struct {
	query PdhHQuery
}

// Open creates a new counterPath that is used to manage the collection of performance data.
// It returns counterPath handle used for subsequent calls for adding counters and querying data
func (m *PerformanceQueryImpl) Open() error {
	if m.query != 0 {
		err := m.Close()
		if err != nil {
			return err
		}
	}
	var handle PdhHQuery

	if ret := PdhOpenQuery(0, 0, &handle); ret != ErrnoSuccess {
		return NewPdhError(ret)
	}
	m.query = handle
	return nil
}

// Close closes the counterPath, releases associated counter handles and frees resources
func (m *PerformanceQueryImpl) Close() error {
	if m.query == 0 {
		return errors.New("uninitialized query")
	}

	if ret := PdhCloseQuery(m.query); ret != ErrnoSuccess {
		return NewPdhError(ret)
	}
	m.query = 0
	return nil
}

func (m *PerformanceQueryImpl) AddCounterToQuery(counterPath string) (PdhHCounter, error) {
	var counterHandle PdhHCounter
	if m.query == 0 {
		return 0, errors.New("uninitialized query")
	}

	if ret := PdhAddCounter(m.query, counterPath, 0, &counterHandle); ret != ErrnoSuccess {
		return 0, NewPdhError(ret)
	}
	return counterHandle, nil
}

func (m *PerformanceQueryImpl) AddEnglishCounterToQuery(counterPath string) (PdhHCounter, error) {
	var counterHandle PdhHCounter
	if m.query == 0 {
		return 0, errors.New("uninitialized query")
	}
	if ret := PdhAddEnglishCounter(m.query, counterPath, 0, &counterHandle); ret != ErrnoSuccess {
		return 0, NewPdhError(ret)
	}
	return counterHandle, nil
}

// GetCounterPath return counter information for given handle
func (m *PerformanceQueryImpl) GetCounterPath(counterHandle PdhHCounter) (string, error) {
	var bufSize uint32
	var buff []byte
	var ret uint32
	if ret = PdhGetCounterInfo(counterHandle, 0, &bufSize, nil); ret == pdhMoreData {
		buff = make([]byte, bufSize)
		bufSize = uint32(len(buff))
		if ret = PdhGetCounterInfo(counterHandle, 0, &bufSize, &buff[0]); ret == ErrnoSuccess {
			ci := (*PdhCounterInfo)(unsafe.Pointer(&buff[0]))
			return UTF16PtrToString(ci.SzFullPath), nil
		}
	}
	return "", NewPdhError(ret)
}

// ExpandWildCardPath  examines local computer and returns those counter paths that match the given counter path which contains wildcard characters.
func (m *PerformanceQueryImpl) ExpandWildCardPath(counterPath string) ([]string, error) {
	var bufSize uint32
	var buff []uint16
	var ret uint32

	if ret = PdhExpandWildCardPath(counterPath, nil, &bufSize); ret == pdhMoreData {
		buff = make([]uint16, bufSize)
		bufSize = uint32(len(buff))
		ret = PdhExpandWildCardPath(counterPath, &buff[0], &bufSize)
		if ret == ErrnoSuccess {
			list := UTF16ToStringArray(buff)
			return list, nil
		}
	}
	return nil, NewPdhError(ret)
}

// GetFormattedCounterValueDouble computes a displayable value for the specified counter
func (m *PerformanceQueryImpl) GetFormattedCounterValueDouble(hCounter PdhHCounter) (float64, error) {
	var counterType uint32
	var value PdhFmtCountervalueDouble
	var ret uint32

	if ret = PdhGetFormattedCounterValueDouble(hCounter, &counterType, &value); ret == ErrnoSuccess {
		if value.CStatus == pdhCStatusValidData || value.CStatus == pdhCStatusNewData {
			return value.DoubleValue, nil
		}
		return 0, NewPdhError(value.CStatus)
	}
	return 0, NewPdhError(ret)
}

func (m *PerformanceQueryImpl) GetFormattedCounterArrayDouble(hCounter PdhHCounter) ([]CounterValue, error) {
	var buffSize uint32
	var itemCount uint32
	var ret uint32

	if ret = PdhGetFormattedCounterArrayDouble(hCounter, &buffSize, &itemCount, nil); ret == pdhMoreData {
		buff := make([]byte, buffSize)

		if ret = PdhGetFormattedCounterArrayDouble(hCounter, &buffSize, &itemCount, &buff[0]); ret == ErrnoSuccess {
			items := (*[1 << 20]PdhFmtCountervalueItemDouble)(unsafe.Pointer(&buff[0]))[:itemCount]
			values := make([]CounterValue, 0, itemCount)
			for _, item := range items {
				if item.FmtValue.CStatus == pdhCStatusValidData || item.FmtValue.CStatus == pdhCStatusNewData {
					val := CounterValue{UTF16PtrToString(item.SzName), item.FmtValue.DoubleValue}
					values = append(values, val)
				}
			}
			return values, nil
		}
	}
	return nil, NewPdhError(ret)
}

func (m *PerformanceQueryImpl) CollectData() error {
	var ret uint32
	if m.query == 0 {
		return errors.New("uninitialized query")
	}

	if ret = PdhCollectQueryData(m.query); ret != ErrnoSuccess {
		return NewPdhError(ret)
	}
	return nil
}

func (m *PerformanceQueryImpl) CollectDataWithTime() (time.Time, error) {
	if m.query == 0 {
		return time.Now(), errors.New("uninitialized query")
	}
	ret, mtime := PdhCollectQueryDataWithTime(m.query)
	if ret != ErrnoSuccess {
		return time.Now(), NewPdhError(ret)
	}
	return mtime, nil
}

func (m *PerformanceQueryImpl) IsVistaOrNewer() bool {
	return PdhAddEnglishCounterSupported()
}

// UTF16PtrToString converts Windows API LPTSTR (pointer to string) to go string
func UTF16PtrToString(s *uint16) string {
	if s == nil {
		return ""
	}
	return syscall.UTF16ToString((*[1 << 29]uint16)(unsafe.Pointer(s))[0:])
}

// UTF16ToStringArray converts list of Windows API NULL terminated strings  to go string array
func UTF16ToStringArray(buf []uint16) []string {
	var strings []string
	nextLineStart := 0
	stringLine := UTF16PtrToString(&buf[0])
	for stringLine != "" {
		strings = append(strings, stringLine)
		nextLineStart += len([]rune(stringLine)) + 1
		remainingBuf := buf[nextLineStart:]
		stringLine = UTF16PtrToString(&remainingBuf[0])
	}
	return strings
}
