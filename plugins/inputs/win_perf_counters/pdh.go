// Copyright (c) 2010 The win Authors. All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions
// are met:
// 1. Redistributions of source code must retain the above copyright
//    notice, this list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright
//    notice, this list of conditions and the following disclaimer in the
//    documentation and/or other materials provided with the distribution.
// 3. The names of the authors may not be used to endorse or promote products
//    derived from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE AUTHORS ``AS IS'' AND ANY EXPRESS OR
// IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES
// OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED.
// IN NO EVENT SHALL THE AUTHORS BE LIABLE FOR ANY DIRECT, INDIRECT,
// INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT
// NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF
// THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
//
// This is the official list of 'win' authors for copyright purposes.
//
// Alexander Neumann <an2048@googlemail.com>
// Joseph Watson <jtwatson@linux-consulting.us>
// Kevin Pors <krpors@gmail.com>

//go:build windows
// +build windows

package winperfcounters

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Error codes
const (
	ErrnoSuccess                = 0
	ErrnoFailure                = 1
	ErrnoInvalidFunction        = 1
	EpochDifferenceMicros int64 = 11644473600000000
)

type (
	Handle uintptr
)

// PDH error codes, which can be returned by all Pdh* functions. Taken from mingw-w64 pdhmsg.h
const (
	pdhCStatusValidData = 0x00000000 // The returned data is valid.
	pdhCStatusNewData   = 0x00000001 // The return data value is valid and different from the last sample.
	// pdhCStatusNoMachine                   = 0x800007D0 // Unable to connect to the specified computer, or the computer is offline.
	// pchCStatusNoInstance                  = 0x800007D1
	pdhMoreData = 0x800007D2 // The PdhGetFormattedCounterArray* function can return this if there's 'more data to be displayed'.
	// pdhCStatusItemNotValidated            = 0x800007D3
	// pdhRetry                   = 0x800007D4
	pdhNoData                  = 0x800007D5 // The query does not currently contain any counters (for example, limited access)
	pdhCalcNegativeDenominator = 0x800007D6
	// pdhCalcNegativeTimebase               = 0x800007D7
	pdhCalcNegativeValue = 0x800007D8
	// pdhDialogCancelled                    = 0x800007D9
	// pdhEndOfLogFile = 0x800007DA
	// pdhAsyncQueryTimeout                  = 0x800007DB
	// pdhCannotSetDefaultRealtimeDatasource = 0x800007DC
	// pdhCStatusNoObject                 = 0xC0000BB8
	// pdhCStatusNoCounter                = 0xC0000BB9 // The specified counter could not be found.
	pdhCStatusInvalidData = 0xC0000BBA // The counter was successfully found, but the data returned is not valid.
	// pdhMemoryAllocationFailure         = 0xC0000BBB
	// pdhInvalidHandle                   = 0xC0000BBC
	// pdhInvalidArgument                 = 0xC0000BBD // Required argument is missing or incorrect.
	// pdhFunctionNotFound                = 0xC0000BBE
	// pdhCStatusNoCountername            = 0xC0000BBF
	// pdhCStatusBadCountername           = 0xC0000BC0 // Unable to parse the counter path. Check the format and syntax of the specified path.
	// pdhInvalidBuffer                   = 0xC0000BC1
	// pdhInsufficientBuffer              = 0xC0000BC2
	// pdhCannotConnectMachine            = 0xC0000BC3
	// pdhInvalidPath                     = 0xC0000BC4
	// pdhInvalidInstance                 = 0xC0000BC5
	pdhInvalidData = 0xC0000BC6 // specified counter does not contain valid data or a successful status code.
	// pdhNoDialogData                    = 0xC0000BC7
	// pdhCannotReadNameStrings           = 0xC0000BC8
	// pdhLogFileCreateError              = 0xC0000BC9
	// pdhLogFileOpenError                = 0xC0000BCA
	// pdhLogTypeNotFound                 = 0xC0000BCB
	// pdhNoMoreData                      = 0xC0000BCC
	// pdhEntryNotInLogFile               = 0xC0000BCD
	// pdhDataSourceIsLogFile             = 0xC0000BCE
	// pdhDataSourceIsRealTime            = 0xC0000BCF
	// pdhUnableReadLogHeader             = 0xC0000BD0
	// pdhFileNotFound                    = 0xC0000BD1
	// pdhFileAlreadyExists               = 0xC0000BD2
	// pdhNotImplemented                  = 0xC0000BD3
	// pdhStringNotFound                  = 0xC0000BD4
	// pdhUnableMapNameFiles              = 0x80000BD5
	// pdhUnknownLogFormat                = 0xC0000BD6
	// pdhUnknownLogsvcCommand         = 0xC0000BD7
	// pdhLogsvcQueryNotFound          = 0xC0000BD8
	// pdhLogsvcNotOpened              = 0xC0000BD9
	// pdhWbemError                    = 0xC0000BDA
	// pdhAccessDenied                 = 0xC0000BDB
	// pdhLogFileTooSmall              = 0xC0000BDC
	// pdhInvalidDatasource            = 0xC0000BDD
	// pdhInvalidSQLdb                 = 0xC0000BDE
	// pdhNoCounters                   = 0xC0000BDF
	// pdhSQLAllocFailed               = 0xC0000BE0
	// pdhSQLAllocconFailed            = 0xC0000BE1
	// pdhSQLExecDirectFailed          = 0xC0000BE2
	// pdhSQLFetchFailed               = 0xC0000BE3
	// pdhSQLRowcountFailed            = 0xC0000BE4
	// pdhSQLMoreResultsFailed         = 0xC0000BE5
	// pdhSQLConnectFailed             = 0xC0000BE6
	// pdhSQLBindFailed                = 0xC0000BE7
	// pdhCannotConnectWMIServer       = 0xC0000BE8
	// pdhPLACollectcionAlreadyRunning = 0xC0000BE9
	// pdhPLAErrorScheduleOverlap      = 0xC0000BEA
	// pdhPLACollectionNotFound        = 0xC0000BEB
	// pdhPLAErrorScheduleElapsed      = 0xC0000BEC
	// pdhPLAErrorNostart              = 0xC0000BED
	// pdhPLAErrorAlreadyExists        = 0xC0000BEE
	// pdhPLAErrorTypeMismatch         = 0xC0000BEF
	// pdhPLAErrorFilepath             = 0xC0000BF0
	// pdhPLAServiceError              = 0xC0000BF1
	// pdhPLAValidationError           = 0xC0000BF2
	pdhPLAValidationWarning = 0x80000BF3
	// pdhPLAErrorNameTooLong          = 0xC0000BF4
	// pdhInvalidSQLLogFormat          = 0xC0000BF5
	// pdhCounterAlreadyInQuery        = 0xC0000BF6
	// pdhBinaryLogCorrupt             = 0xC0000BF7
	// pdhLogSampleTooSmall            = 0xC0000BF8
	// pdhOSLaterVersion               = 0xC0000BF9
	// pdhOSEarlierVersion             = 0xC0000BFA
	// pdhIncorrectAppendTime          = 0xC0000BFB
	// pdhUnmatchedAppendCounter       = 0xC0000BFC
	// pdhSQLAlterDetailFailed         = 0xC0000BFD
	// pdhQueryPerfDataTimeout         = 0xC0000BFE
)

// Formatting options for GetFormattedCounterValue().
const (
	// pdhFmtRaw          = 0x00000010
	// pdhFmtAnsi         = 0x00000020
	// pdhFmtUnicode      = 0x00000040
	// pdhFmtLong         = 0x00000100 // Return data as a long int.
	pdhFmtDouble = 0x00000200 // Return data as a double precision floating point real.
	// pdhFmtLarge        = 0x00000400 // Return data as a 64 bit integer.
	// pdhFmtNoscale      = 0x00001000 // can be OR-ed: Do not apply the counter's default scaling factor.
	// pdhFmt1000         = 0x00002000 // can be OR-ed: multiply the actual value by 1,000.
	// pdhFmtNodata       = 0x00004000 // can be OR-ed: unknown what this is for, MSDN says nothing.
	pdhFmtNocap100 = 0x00008000 // can be OR-ed: do not cap values > 100.
	// perfDetailCostly   = 0x00010000
	// perfDetailStandard = 0x0000FFFF
)

type (
	PdhHQuery   Handle // query handle
	PdhHCounter Handle // counter handle
)

var (
	// Library
	libpdhDll *syscall.DLL

	// Functions
	pdhAddCounterW               *syscall.Proc
	pdhAddEnglishCounterW        *syscall.Proc
	pdhCloseQuery                *syscall.Proc
	pdhCollectQueryData          *syscall.Proc
	pdhCollectQueryDataWithTime  *syscall.Proc
	pdhGetFormattedCounterValue  *syscall.Proc
	pdhGetFormattedCounterArrayW *syscall.Proc
	pdhOpenQuery                 *syscall.Proc
	pdhValidatePathW             *syscall.Proc
	pdhExpandWildCardPathW       *syscall.Proc
	pdhGetCounterInfoW           *syscall.Proc
)

func init() {
	// Library
	libpdhDll = syscall.MustLoadDLL("pdh.dll")

	// Functions
	pdhAddCounterW = libpdhDll.MustFindProc("PdhAddCounterW")
	pdhAddEnglishCounterW, _ = libpdhDll.FindProc("PdhAddEnglishCounterW") // XXX: only supported on versions > Vista.
	pdhCloseQuery = libpdhDll.MustFindProc("PdhCloseQuery")
	pdhCollectQueryData = libpdhDll.MustFindProc("PdhCollectQueryData")
	pdhCollectQueryDataWithTime, _ = libpdhDll.FindProc("PdhCollectQueryDataWithTime")
	pdhGetFormattedCounterValue = libpdhDll.MustFindProc("PdhGetFormattedCounterValue")
	pdhGetFormattedCounterArrayW = libpdhDll.MustFindProc("PdhGetFormattedCounterArrayW")
	pdhOpenQuery = libpdhDll.MustFindProc("PdhOpenQuery")
	pdhValidatePathW = libpdhDll.MustFindProc("PdhValidatePathW")
	pdhExpandWildCardPathW = libpdhDll.MustFindProc("PdhExpandWildCardPathW")
	pdhGetCounterInfoW = libpdhDll.MustFindProc("PdhGetCounterInfoW")
}

// PdhAddCounter adds the specified counter to the query. This is the internationalized version. Preferably, use the
// function PdhAddEnglishCounter instead. hQuery is the query handle, which has been fetched by PdhOpenQuery.
// szFullCounterPath is a full, internationalized counter path (this will differ per Windows language version).
// dwUserData is a 'user-defined value', which becomes part of the counter information. To retrieve this value
// later, call PdhGetCounterInfo() and access dwQueryUserData of the PDH_COUNTER_INFO structure.
//
// Examples of szFullCounterPath (in an English version of Windows):
//
//	\\Processor(_Total)\\% Idle Time
//	\\Processor(_Total)\\% Processor Time
//	\\LogicalDisk(C:)\% Free Space
//
// To view all (internationalized...) counters on a system, there are three non-programmatic ways: perfmon utility,
// the typeperf command, and the the registry editor. perfmon.exe is perhaps the easiest way, because it's basically a
// full implementation of the pdh.dll API, except with a GUI and all that. The registry setting also provides an
// interface to the available counters, and can be found at the following key:
//
// 	HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Perflib\CurrentLanguage
//
// This registry key contains several values as follows:
//
//	1
//	1847
//	2
//	System
//	4
//	Memory
//	6
//	% Processor Time
//	... many, many more
//
// Somehow, these numeric values can be used as szFullCounterPath too:
//
//	\2\6 will correspond to \\System\% Processor Time
//
// The typeperf command may also be pretty easy. To find all performance counters, simply execute:
//
//	typeperf -qx
func PdhAddCounter(hQuery PdhHQuery, szFullCounterPath string, dwUserData uintptr, phCounter *PdhHCounter) uint32 {
	ptxt, _ := syscall.UTF16PtrFromString(szFullCounterPath)
	ret, _, _ := pdhAddCounterW.Call(
		uintptr(hQuery),
		uintptr(unsafe.Pointer(ptxt)),
		dwUserData,
		uintptr(unsafe.Pointer(phCounter)))

	return uint32(ret)
}

// PdhAddEnglishCounterSupported returns true if PdhAddEnglishCounterW Win API function was found in pdh.dll.
// PdhAddEnglishCounterW function is not supported on pre-Windows Vista systems

func PdhAddEnglishCounterSupported() bool {
	return pdhAddEnglishCounterW != nil
}

// PdhAddEnglishCounter adds the specified language-neutral counter to the query. See the PdhAddCounter function. This function only exists on
// Windows versions higher than Vista.
func PdhAddEnglishCounter(hQuery PdhHQuery, szFullCounterPath string, dwUserData uintptr, phCounter *PdhHCounter) uint32 {
	if pdhAddEnglishCounterW == nil {
		return ErrnoInvalidFunction
	}

	ptxt, _ := syscall.UTF16PtrFromString(szFullCounterPath)
	ret, _, _ := pdhAddEnglishCounterW.Call(
		uintptr(hQuery),
		uintptr(unsafe.Pointer(ptxt)),
		dwUserData,
		uintptr(unsafe.Pointer(phCounter)))

	return uint32(ret)
}

// PdhCloseQuery closes all counters contained in the specified query, closes all handles related to the query,
// and frees all memory associated with the query.
func PdhCloseQuery(hQuery PdhHQuery) uint32 {
	ret, _, _ := pdhCloseQuery.Call(uintptr(hQuery))

	return uint32(ret)
}

// Collects the current raw data value for all counters in the specified query and updates the status
// code of each counter. With some counters, this function needs to be repeatedly called before the value
// of the counter can be extracted with PdhGetFormattedCounterValue(). For example, the following code
// requires at least two calls:
//
// 	var handle win.PDH_HQUERY
// 	var counterHandle win.PDH_HCOUNTER
// 	ret := win.PdhOpenQuery(0, 0, &handle)
//	ret = win.PdhAddEnglishCounter(handle, "\\Processor(_Total)\\% Idle Time", 0, &counterHandle)
//	var derp win.PDH_FMT_COUNTERVALUE_DOUBLE
//
//	ret = win.PdhCollectQueryData(handle)
//	fmt.Printf("Collect return code is %x\n", ret) // return code will be PDH_CSTATUS_INVALID_DATA
//	ret = win.PdhGetFormattedCounterValueDouble(counterHandle, 0, &derp)
//
//	ret = win.PdhCollectQueryData(handle)
//	fmt.Printf("Collect return code is %x\n", ret) // return code will be ERROR_SUCCESS
//	ret = win.PdhGetFormattedCounterValueDouble(counterHandle, 0, &derp)
//
// The PdhCollectQueryData will return an error in the first call because it needs two values for
// displaying the correct data for the processor idle time. The second call will have a 0 return code.
func PdhCollectQueryData(hQuery PdhHQuery) uint32 {
	ret, _, _ := pdhCollectQueryData.Call(uintptr(hQuery))

	return uint32(ret)
}

// PdhCollectQueryDataWithTime queries data from perfmon, retrieving the device/windows timestamp from the node it was collected on.
// Converts the filetime structure to a GO time class and returns the native time.
//
func PdhCollectQueryDataWithTime(hQuery PdhHQuery) (uint32, time.Time) {
	var localFileTime FILETIME
	ret, _, _ := pdhCollectQueryDataWithTime.Call(uintptr(hQuery), uintptr(unsafe.Pointer(&localFileTime)))

	if ret == ErrnoSuccess {
		var utcFileTime FILETIME
		ret, _, _ := krnLocalFileTimeToFileTime.Call(
			uintptr(unsafe.Pointer(&localFileTime)),
			uintptr(unsafe.Pointer(&utcFileTime)))

		if ret == 0 {
			return uint32(ErrnoFailure), time.Now()
		}

		// First convert 100-ns intervals to microseconds, then adjust for the
		// epoch difference
		var totalMicroSeconds int64
		totalMicroSeconds = ((int64(utcFileTime.dwHighDateTime) << 32) | int64(utcFileTime.dwLowDateTime)) / 10
		totalMicroSeconds -= EpochDifferenceMicros

		retTime := time.Unix(0, totalMicroSeconds*1000)

		return uint32(ErrnoSuccess), retTime
	}

	return uint32(ret), time.Now()
}

// PdhGetFormattedCounterValueDouble formats the given hCounter using a 'double'. The result is set into the specialized union struct pValue.
// This function does not directly translate to a Windows counterpart due to union specialization tricks.
func PdhGetFormattedCounterValueDouble(hCounter PdhHCounter, lpdwType *uint32, pValue *PdhFmtCountervalueDouble) uint32 {
	ret, _, _ := pdhGetFormattedCounterValue.Call(
		uintptr(hCounter),
		uintptr(pdhFmtDouble|pdhFmtNocap100),
		uintptr(unsafe.Pointer(lpdwType)),
		uintptr(unsafe.Pointer(pValue)))

	return uint32(ret)
}

// PdhGetFormattedCounterArrayDouble returns an array of formatted counter values. Use this function when you want to format the counter values of a
// counter that contains a wildcard character for the instance name. The itemBuffer must a slice of type PDH_FMT_COUNTERVALUE_ITEM_DOUBLE.
// An example of how this function can be used:
//
//	okPath := "\\Process(*)\\% Processor Time" // notice the wildcard * character
//
//	// omitted all necessary stuff ...
//
//	var bufSize uint32
//	var bufCount uint32
//	var size uint32 = uint32(unsafe.Sizeof(win.PDH_FMT_COUNTERVALUE_ITEM_DOUBLE{}))
//	var emptyBuf [1]win.PDH_FMT_COUNTERVALUE_ITEM_DOUBLE // need at least 1 addressable null ptr.
//
//	for {
//		// collect
//		ret := win.PdhCollectQueryData(queryHandle)
//		if ret == win.ERROR_SUCCESS {
//			ret = win.PdhGetFormattedCounterArrayDouble(counterHandle, &bufSize, &bufCount, &emptyBuf[0]) // uses null ptr here according to MSDN.
//			if ret == win.PDH_MORE_DATA {
//				filledBuf := make([]win.PDH_FMT_COUNTERVALUE_ITEM_DOUBLE, bufCount*size)
//				ret = win.PdhGetFormattedCounterArrayDouble(counterHandle, &bufSize, &bufCount, &filledBuf[0])
//				for i := 0; i < int(bufCount); i++ {
//					c := filledBuf[i]
//					var s string = win.UTF16PtrToString(c.SzName)
//					fmt.Printf("Index %d -> %s, value %v\n", i, s, c.FmtValue.DoubleValue)
//				}
//
//				filledBuf = nil
//				// Need to at least set bufSize to zero, because if not, the function will not
//				// return PDH_MORE_DATA and will not set the bufSize.
//				bufCount = 0
//				bufSize = 0
//			}
//
//			time.Sleep(2000 * time.Millisecond)
//		}
//	}
func PdhGetFormattedCounterArrayDouble(hCounter PdhHCounter, lpdwBufferSize *uint32, lpdwBufferCount *uint32, itemBuffer *byte) uint32 {
	ret, _, _ := pdhGetFormattedCounterArrayW.Call(
		uintptr(hCounter),
		uintptr(pdhFmtDouble|pdhFmtNocap100),
		uintptr(unsafe.Pointer(lpdwBufferSize)),
		uintptr(unsafe.Pointer(lpdwBufferCount)),
		uintptr(unsafe.Pointer(itemBuffer)))

	return uint32(ret)
}

// PdhOpenQuery creates a new query that is used to manage the collection of performance data.
// szDataSource is a null terminated string that specifies the name of the log file from which to
// retrieve the performance data. If 0, performance data is collected from a real-time data source.
// dwUserData is a user-defined value to associate with this query. To retrieve the user data later,
// call PdhGetCounterInfo and access dwQueryUserData of the PDH_COUNTER_INFO structure. phQuery is
// the handle to the query, and must be used in subsequent calls. This function returns a PDH_
// constant error code, or ERROR_SUCCESS if the call succeeded.
func PdhOpenQuery(szDataSource uintptr, dwUserData uintptr, phQuery *PdhHQuery) uint32 {
	ret, _, _ := pdhOpenQuery.Call(
		szDataSource,
		dwUserData,
		uintptr(unsafe.Pointer(phQuery)))

	return uint32(ret)
}

// PdhExpandWildCardPath examines the specified computer or log file and returns those counter paths that match the given counter path which contains wildcard characters.
// The general counter path format is as follows:

// \\computer\object(parent/instance#index)\counter

// The parent, instance, index, and counter components of the counter path may contain either a valid name or a wildcard character. The computer, parent, instance,
// and index components are not necessary for all counters.

// The following is a list of the possible formats:

// \\computer\object(parent/instance#index)\counter
// \\computer\object(parent/instance)\counter
// \\computer\object(instance#index)\counter
// \\computer\object(instance)\counter
// \\computer\object\counter
// \object(parent/instance#index)\counter
// \object(parent/instance)\counter
// \object(instance#index)\counter
// \object(instance)\counter
// \object\counter
// Use an asterisk (*) as the wildcard character, for example, \object(*)\counter.

// If a wildcard character is specified in the parent name, all instances of the specified object that match the specified instance and counter fields will be returned.
// For example, \object(*/instance)\counter.

// If a wildcard character is specified in the instance name, all instances of the specified object and parent object will be returned if all instance names
// corresponding to the specified index match the wildcard character. For example, \object(parent/*)\counter. If the object does not contain an instance, an error occurs.

// If a wildcard character is specified in the counter name, all counters of the specified object are returned.

// Partial counter path string matches (for example, "pro*") are supported.
func PdhExpandWildCardPath(szWildCardPath string, mszExpandedPathList *uint16, pcchPathListLength *uint32) uint32 {
	ptxt, _ := syscall.UTF16PtrFromString(szWildCardPath)
	flags := uint32(0) // expand instances and counters
	ret, _, _ := pdhExpandWildCardPathW.Call(
		uintptr(unsafe.Pointer(nil)), // search counters on local computer
		uintptr(unsafe.Pointer(ptxt)),
		uintptr(unsafe.Pointer(mszExpandedPathList)),
		uintptr(unsafe.Pointer(pcchPathListLength)),
		uintptr(unsafe.Pointer(&flags)))

	return uint32(ret)
}

// PdhValidatePath validates a path. Will return ERROR_SUCCESS when ok, or PDH_CSTATUS_BAD_COUNTERNAME when the path is
// erroneous.
func PdhValidatePath(path string) uint32 {
	ptxt, _ := syscall.UTF16PtrFromString(path)
	ret, _, _ := pdhValidatePathW.Call(uintptr(unsafe.Pointer(ptxt)))

	return uint32(ret)
}

func PdhFormatError(msgID uint32) string {
	var flags uint32 = windows.FORMAT_MESSAGE_FROM_HMODULE | windows.FORMAT_MESSAGE_ARGUMENT_ARRAY | windows.FORMAT_MESSAGE_IGNORE_INSERTS
	buf := make([]uint16, 300)
	_, err := windows.FormatMessage(flags, uintptr(libpdhDll.Handle), msgID, 0, buf, nil)
	if err == nil {
		return UTF16PtrToString(&buf[0])
	}
	return fmt.Sprintf("(pdhErr=%d) %s", msgID, err.Error())
}

// Retrieves information about a counter, such as data size, counter type, path, and user-supplied data values
// hCounter [in]
// Handle of the counter from which you want to retrieve information. The PdhAddCounter function returns this handle.

// bRetrieveExplainText [in]
// Determines whether explain text is retrieved. If you set this parameter to TRUE, the explain text for the counter is retrieved. If you set this parameter to FALSE, the field in the returned buffer is NULL.

// pdwBufferSize [in, out]
// Size of the lpBuffer buffer, in bytes. If zero on input, the function returns PDH_MORE_DATA and sets this parameter to the required buffer size. If the buffer is larger than the required size, the function sets this parameter to the actual size of the buffer that was used. If the specified size on input is greater than zero but less than the required size, you should not rely on the returned size to reallocate the buffer.

// lpBuffer [out]
// Caller-allocated buffer that receives a PDH_COUNTER_INFO structure. The structure is variable-length, because the string data is appended to the end of the fixed-format portion of the structure. This is done so that all data is returned in a single buffer allocated by the caller. Set to NULL if pdwBufferSize is zero.
func PdhGetCounterInfo(hCounter PdhHCounter, bRetrieveExplainText int, pdwBufferSize *uint32, lpBuffer *byte) uint32 {
	ret, _, _ := pdhGetCounterInfoW.Call(
		uintptr(hCounter),
		uintptr(bRetrieveExplainText),
		uintptr(unsafe.Pointer(pdwBufferSize)),
		uintptr(unsafe.Pointer(lpBuffer)))

	return uint32(ret)
}
