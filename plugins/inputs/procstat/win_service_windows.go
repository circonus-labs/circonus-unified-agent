//go:build windows
// +build windows

package procstat

import (
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc/mgr"
)

func getService(name string) (*mgr.Service, error) {
	m, err := mgr.Connect()
	if err != nil {
		return nil, fmt.Errorf("win_service mgr conn: %w", err)
	}
	defer func() { _ = m.Disconnect() }()

	srv, err := m.OpenService(name)
	if err != nil {
		return nil, fmt.Errorf("win_service open svc: %w", err)
	}

	return srv, nil
}

func queryPidWithWinServiceName(winServiceName string) (uint32, error) {
	srv, err := getService(winServiceName)
	if err != nil {
		return 0, err
	}

	var p *windows.SERVICE_STATUS_PROCESS
	var bytesNeeded uint32
	var buf []byte

	if err := windows.QueryServiceStatusEx(srv.Handle, windows.SC_STATUS_PROCESS_INFO, nil, 0, &bytesNeeded); !errors.Is(err, windows.ERROR_INSUFFICIENT_BUFFER) {
		return 0, fmt.Errorf("win_service qry svc status: %w", err)
	}

	buf = make([]byte, bytesNeeded)
	p = (*windows.SERVICE_STATUS_PROCESS)(unsafe.Pointer(&buf[0]))
	if err := windows.QueryServiceStatusEx(srv.Handle, windows.SC_STATUS_PROCESS_INFO, &buf[0], uint32(len(buf)), &bytesNeeded); err != nil {
		return 0, fmt.Errorf("win_service qry svc status: %w", err)
	}

	return p.ProcessId, nil
}
