//go:build windows
// +build windows

//revive:disable-next-line:var-naming
// Package win_eventlog Input plugin to collect Windows Event Log messages
package wineventlog

import "syscall"

// Event log error codes.
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms681382(v=vs.85).aspx
const (
	//revive:disable:var-naming
	ErrnoInsufficientBuffer syscall.Errno = 122
	ErrnoNoMoreItems        syscall.Errno = 259
	ErrnoInvalidOperation   syscall.Errno = 4317
	//revive:enable:var-naming
)

// EvtSubscribeFlag defines the possible values that specify when to start subscribing to events.
type EvtSubscribeFlag uint32

// EVT_SUBSCRIBE_FLAGS enumeration
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa385588(v=vs.85).aspx
const (
	EvtSubscribeToFutureEvents EvtSubscribeFlag = 1
)

// EvtRenderFlag uint32
type EvtRenderFlag uint32

// EVT_RENDER_FLAGS enumeration
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa385563(v=vs.85).aspx
const (
	//revive:disable:var-naming
	// Render the event as an XML string. For details on the contents of the
	// XML string, see the Event schema.
	EvtRenderEventXML EvtRenderFlag = 1
	//revive:enable:var-naming
)
