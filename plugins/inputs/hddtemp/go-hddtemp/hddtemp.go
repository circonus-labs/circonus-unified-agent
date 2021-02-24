package hddtemp

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
)

type Disk struct {
	DeviceName  string
	Model       string
	Temperature int32
	Unit        string
	Status      string
}

type HDDTemp struct {
}

func New() *HDDTemp {
	return &HDDTemp{}
}

func (h *HDDTemp) Fetch(address string) ([]Disk, error) {
	var (
		err    error
		conn   net.Conn
		buffer bytes.Buffer
		disks  []Disk
	)

	if conn, err = net.Dial("tcp", address); err != nil {
		return nil, fmt.Errorf("dial tcp (%s): %w", address, err)
	}

	if _, err = io.Copy(&buffer, conn); err != nil {
		return nil, fmt.Errorf("io copy: %w", err)
	}

	fields := strings.Split(buffer.String(), "|")

	for index := 0; index < len(fields)/5; index++ {
		status := ""
		offset := index * 5
		device := fields[offset+1]
		device = device[strings.LastIndex(device, "/")+1:]

		temperatureField := fields[offset+3]
		temperature, err := strconv.ParseInt(temperatureField, 10, 32)

		if err != nil {
			temperature = 0
			status = temperatureField
		}

		disks = append(disks, Disk{
			DeviceName:  device,
			Model:       fields[offset+2],
			Temperature: int32(temperature),
			Unit:        fields[offset+4],
			Status:      status,
		})
	}

	return disks, nil
}
