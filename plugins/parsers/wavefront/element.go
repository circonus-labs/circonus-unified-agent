package wavefront

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

var (
	ErrEOF              = errors.New("EOF")
	ErrInvalidTimestamp = errors.New("Invalid timestamp")
)

// Interface for parsing line elements.
type ElementParser interface {
	parse(p *PointParser, pt *Point) error
}

type NameParser struct{}
type ValueParser struct{}
type TimestampParser struct {
	optional bool
}
type WhiteSpaceParser struct {
	nextOptional bool
}
type TagParser struct{}
type LoopedParser struct {
	wrappedParser ElementParser
	wsParser      *WhiteSpaceParser
}
type LiteralParser struct {
	// literal string
}

func (ep *NameParser) parse(p *PointParser, pt *Point) error {
	// Valid characters are: a-z, A-Z, 0-9, hyphen ("-"), underscore ("_"), dot (".").
	// Forward slash ("/") and comma (",") are allowed if metricName is enclosed in double quotes.
	// Delta (U+2206) is allowed as the first character of the
	// metricName
	name, err := parseLiteral(p)

	if err != nil {
		return err
	}
	pt.Name = name
	return nil
}

func (ep *ValueParser) parse(p *PointParser, pt *Point) error {
	tok, lit := p.scan()
	if tok == EOF {
		return fmt.Errorf("found %q, expected number", lit)
	}

	p.writeBuf.Reset()
	if tok == MINUSSIGN {
		p.writeBuf.WriteString(lit)
		tok, lit = p.scan()
	}

	for tok != EOF && (tok == LETTER || tok == NUMBER || tok == DOT || tok == MINUSSIGN) {
		p.writeBuf.WriteString(lit)
		tok, lit = p.scan()
	}
	p.unscan()

	pt.Value = p.writeBuf.String()
	_, err := strconv.ParseFloat(pt.Value, 64)
	if err != nil {
		return fmt.Errorf("invalid metric value %s", pt.Value)
	}
	return nil
}

func (ep *TimestampParser) parse(p *PointParser, pt *Point) error {
	tok, lit := p.scan()
	if tok == EOF {
		if ep.optional {
			p.unscanTokens(2)
			return setTimestamp(pt, 0, 1)
		}
		return fmt.Errorf("found %q, expected number", lit)
	}

	if tok != NUMBER {
		if ep.optional {
			p.unscanTokens(2)
			return setTimestamp(pt, 0, 1)
		}
		return ErrInvalidTimestamp
	}

	p.writeBuf.Reset()
	for tok != EOF && tok == NUMBER {
		p.writeBuf.WriteString(lit)
		tok, lit = p.scan()
	}
	p.unscan()

	tsStr := p.writeBuf.String()
	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return fmt.Errorf("parse int (%s): %w", tsStr, err)
	}
	return setTimestamp(pt, ts, len(tsStr))
}

func setTimestamp(pt *Point, ts int64, numDigits int) error {

	switch {
	case numDigits == 19:
		// nanoseconds
		ts /= 1e9
	case numDigits == 16:
		// microseconds
		ts /= 1e6
	case numDigits == 13:
		// milliseconds
		ts /= 1e3
	case numDigits != 10:
		// must be in seconds, return error if not 0
		if ts == 0 {
			ts = getCurrentTime()
		} else {
			return ErrInvalidTimestamp
		}
	}
	pt.Timestamp = ts
	return nil
}

func (ep *LoopedParser) parse(p *PointParser, pt *Point) error {
	for {
		err := ep.wrappedParser.parse(p, pt)
		if err != nil {
			return fmt.Errorf("parser: %w", err)
		}
		err = ep.wsParser.parse(p, pt)
		if errors.Is(err, ErrEOF) {
			break
		}
	}
	return nil
}

func (ep *TagParser) parse(p *PointParser, pt *Point) error {
	k, err := parseLiteral(p)
	if err != nil {
		if k == "" {
			return nil
		}
		return err
	}

	next, lit := p.scan()
	if next != EQUALS {
		return fmt.Errorf("found %q, expected equals", lit)
	}

	v, err := parseLiteral(p)
	if err != nil {
		return err
	}
	if len(pt.Tags) == 0 {
		pt.Tags = make(map[string]string)
	}
	pt.Tags[k] = v
	return nil
}

func (ep *WhiteSpaceParser) parse(p *PointParser, pt *Point) error {
	tok := WS
	for tok != EOF && tok == WS {
		tok, _ = p.scan()
	}

	if tok == EOF {
		if !ep.nextOptional {
			return ErrEOF
		}
		return nil
	}
	p.unscan()
	return nil
}

// unused
// func (ep *LiteralParser) parse(p *PointParser) error {
// 	l, err := parseLiteral(p)
// 	if err != nil {
// 		return err
// 	}

// 	if l != ep.literal {
// 		return fmt.Errorf("found %s, expected %s", l, ep.literal)
// 	}
// 	return nil
// }

func parseQuotedLiteral(p *PointParser) (string, error) {
	p.writeBuf.Reset()

	escaped := false
	tok, lit := p.scan()
	for tok != EOF && (tok != QUOTES || (tok == QUOTES && escaped)) {
		// let everything through
		escaped = tok == BACKSLASH
		p.writeBuf.WriteString(lit)
		tok, lit = p.scan()
	}
	if tok == EOF {
		return "", fmt.Errorf("found %q, expected quotes", lit)
	}
	return p.writeBuf.String(), nil
}

func parseLiteral(p *PointParser) (string, error) {
	tok, lit := p.scan()
	if tok == EOF {
		return "", fmt.Errorf("found %q, expected literal", lit)
	}

	if tok == QUOTES {
		return parseQuotedLiteral(p)
	}

	p.writeBuf.Reset()
	for tok != EOF && tok > literalBeg && tok < literalEnd {
		p.writeBuf.WriteString(lit)
		tok, lit = p.scan()
		if tok == DELTA {
			return "", errors.New("found delta inside metric name")
		}
	}
	if tok == QUOTES {
		return "", errors.New("found quote inside unquoted literal")
	}
	p.unscan()
	return p.writeBuf.String(), nil
}

func getCurrentTime() int64 {
	return time.Now().UnixNano() / 1e9
}
