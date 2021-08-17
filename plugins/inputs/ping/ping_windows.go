//go:build windows
// +build windows

package ping

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/circonus-labs/circonus-unified-agent/cua"
)

func (p *Ping) pingToURL(ctx context.Context, u string, acc cua.Accumulator) {
	tags := map[string]string{"url": u}
	fields := map[string]interface{}{"result_code": 0}

	args := p.args(u)
	totalTimeout := 60.0
	if len(p.Arguments) == 0 {
		totalTimeout = p.timeout() * float64(p.Count)
	}

	out, err := p.pingHost(p.Binary, totalTimeout, args...)
	// ping host return exitcode != 0 also when there was no response from host
	// but command was execute successfully
	var pendingError error
	if err != nil {
		// Combine go err + stderr output
		pendingError = errors.New(strings.TrimSpace(out) + ", " + err.Error())
	}
	trans, recReply, receivePacket, avg, min, max, rtts, err := processPingOutput(out)
	if err != nil {
		// fatal error
		if pendingError != nil {
			acc.AddError(fmt.Errorf("%s: %w", u, pendingError))
		} else {
			acc.AddError(fmt.Errorf("%s: %w", u, err))
		}

		fields["result_code"] = 2
		fields["errors"] = 100.0
		p.addStats(acc, fields, tags, nil, nil)
		// acc.AddFields("ping", fields, tags)
		return
	}
	// Calculate packet loss percentage
	lossReply := float64(trans-recReply) / float64(trans) * 100.0
	lossPackets := float64(trans-receivePacket) / float64(trans) * 100.0

	fields["packets_transmitted"] = trans
	fields["reply_received"] = recReply
	fields["packets_received"] = receivePacket
	fields["percent_packet_loss"] = lossPackets
	fields["percent_reply_loss"] = lossReply
	if avg >= 0 {
		fields["average_response_ms"] = float64(avg)
	}
	if min >= 0 {
		fields["minimum_response_ms"] = float64(min)
	}
	if max >= 0 {
		fields["maximum_response_ms"] = float64(max)
	}

	p.addStats(acc, fields, tags, nil, &rtts)
	// acc.AddFields("ping", fields, tags)
}

// args returns the arguments for the 'ping' executable
func (p *Ping) args(url string) []string {
	if len(p.Arguments) > 0 {
		return p.Arguments
	}

	args := []string{"-n", strconv.Itoa(p.Count)}

	if p.Timeout > 0 {
		args = append(args, "-w", strconv.FormatFloat(p.Timeout*1000, 'f', 0, 64))
	}

	args = append(args, url)

	return args
}

// processPingOutput takes in a string output from the ping command
// based on linux implementation but using regex ( multilanguage support )
// It returns (<transmitted packets>, <received reply>, <received packet>, <average response>, <min response>, <max response>, []rtts)
func processPingOutput(out string) (int, int, int, int, int, int, []float64, error) {
	// So find a line contain 3 numbers except reply lines
	var stats, aproxs []string = nil, nil
	rtts := make([]float64, 0)
	err := errors.New("Fatal error processing ping output")
	stat := regexp.MustCompile(`=\W*(\d+)\D*=\W*(\d+)\D*=\W*(\d+)`)
	aprox := regexp.MustCompile(`=\W*(\d+)\D*ms\D*=\W*(\d+)\D*ms\D*=\W*(\d+)\D*ms`)
	tttLine := regexp.MustCompile(`TTL=\d+`)
	timeLine := regexp.MustCompile(`time=(\d+[\.\d]*)`)
	lines := strings.Split(out, "\n")
	var receivedReply int = 0
	for _, line := range lines {
		if tttLine.MatchString(line) {
			receivedReply++
			rttMatch := timeLine.FindStringSubmatch(line)
			if len(rttMatch) < 2 {
				continue
			}
			rtt, err := strconv.ParseFloat(rttMatch[1], 32)
			if err == nil {
				rtts = append(rtts, rtt)
			}
		} else {
			if stats == nil {
				stats = stat.FindStringSubmatch(line)
			}
			if stats != nil && aproxs == nil {
				aproxs = aprox.FindStringSubmatch(line)
			}
		}
	}

	// stats data should contain 4 members: entireExpression + ( Send, Receive, Lost )
	if len(stats) != 4 {
		return 0, 0, 0, -1, -1, -1, rtts, err
	}
	trans, err := strconv.Atoi(stats[1])
	if err != nil {
		return 0, 0, 0, -1, -1, -1, rtts, err
	}
	receivedPacket, err := strconv.Atoi(stats[2])
	if err != nil {
		return 0, 0, 0, -1, -1, -1, rtts, err
	}

	// aproxs data should contain 4 members: entireExpression + ( min, max, avg )
	if len(aproxs) != 4 {
		return trans, receivedReply, receivedPacket, -1, -1, -1, rtts, err
	}
	min, err := strconv.Atoi(aproxs[1])
	if err != nil {
		return trans, receivedReply, receivedPacket, -1, -1, -1, rtts, err
	}
	max, err := strconv.Atoi(aproxs[2])
	if err != nil {
		return trans, receivedReply, receivedPacket, -1, -1, -1, rtts, err
	}
	avg, err := strconv.Atoi(aproxs[3])
	if err != nil {
		return 0, 0, 0, -1, -1, -1, rtts, err
	}

	return trans, receivedReply, receivedPacket, avg, min, max, rtts, err
}

func (p *Ping) timeout() float64 {
	// According to MSDN, default ping timeout for windows is 4 second
	// Add also one second interval

	if p.Timeout > 0 {
		return p.Timeout + 1
	}
	return 4 + 1
}
