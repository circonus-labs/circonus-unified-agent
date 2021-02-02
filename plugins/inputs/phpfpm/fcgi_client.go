package phpfpm

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
)

// Create an fcgi client
func newFcgiClient(h string, args ...interface{}) (*conn, error) {
	var con net.Conn
	if len(args) != 1 {
		return nil, fmt.Errorf("fcgi: not enough params")
	}

	var err error
	switch arg := args[0].(type) {
	case int:
		addr := h + ":" + strconv.FormatInt(int64(arg), 10)
		con, err = net.Dial("tcp", addr)
	case string:
		laddr := net.UnixAddr{Name: arg, Net: h}
		con, err = net.DialUnix(h, nil, &laddr)
	default:
		err = fmt.Errorf("fcgi: we only accept int (port) or string (socket) params")
	}
	fcgi := &conn{
		rwc: con,
	}

	return fcgi, err
}

func (client *conn) Request(
	env map[string]string,
	requestData string,
) (retout []byte, reterr []byte, err error) {
	defer client.rwc.Close()
	var reqID uint16 = 1

	err = client.writeBeginRequest(reqID, uint16(roleResponder), 0)
	if err != nil {
		return
	}

	err = client.writePairs(typeParams, reqID, env)
	if err != nil {
		return
	}

	if len(requestData) > 0 {
		if err = client.writeRecord(typeStdin, reqID, []byte(requestData)); err != nil {
			return
		}
	}

	rec := &record{}
	var err1 error

	// receive until EOF or FCGI_END_REQUEST
READ_LOOP:
	for {
		err1 = rec.read(client.rwc)
		if err1 != nil && strings.Contains(err1.Error(), "use of closed network connection") {
			if !errors.Is(err1, io.EOF) {
				err = err1
			}
			break
		}

		switch {
		case rec.h.Type == typeStdout:
			retout = append(retout, rec.content()...)
		case rec.h.Type == typeStderr:
			reterr = append(reterr, rec.content()...)
		case rec.h.Type == typeEndRequest:
			fallthrough
		default:
			break READ_LOOP
		}
	}

	return
}
