package csv

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	_ "time/tzdata" // needed to bundle timezone info into the binary for Windows

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/internal"
	"github.com/circonus-labs/circonus-unified-agent/metric"
)

type TimeFunc func() time.Time

type Config struct {
	DefaultTags       map[string]string
	TimeFunc          func() time.Time
	Timezone          string   `toml:"csv_timezone"`
	Comment           string   `toml:"csv_comment"`
	Delimiter         string   `toml:"csv_delimiter"`
	MeasurementColumn string   `toml:"csv_measurement_column"`
	MetricName        string   `toml:"metric_name"`
	TimestampColumn   string   `toml:"csv_timestamp_column"`
	TimestampFormat   string   `toml:"csv_timestamp_format"`
	ColumnTypes       []string `toml:"csv_column_types"`
	ColumnNames       []string `toml:"csv_column_names"`
	TagColumns        []string `toml:"csv_tag_columns"`
	HeaderRowCount    int      `toml:"csv_header_row_count"`
	SkipRows          int      `toml:"csv_skip_rows"`
	SkipColumns       int      `toml:"csv_skip_columns"`
	TrimSpace         bool     `toml:"csv_trim_space"`
	gotColumnNames    bool
}

// Parser is a CSV parser, you should use NewParser to create a new instance.
type Parser struct {
	*Config
}

func NewParser(c *Config) (*Parser, error) {
	if c.HeaderRowCount == 0 && len(c.ColumnNames) == 0 {
		return nil, fmt.Errorf("`csv_header_row_count` must be defined if `csv_column_names` is not specified")
	}

	if c.Delimiter != "" {
		runeStr := []rune(c.Delimiter)
		if len(runeStr) > 1 {
			return nil, fmt.Errorf("csv_delimiter must be a single character, got: %s", c.Delimiter)
		}
	}

	if c.Comment != "" {
		runeStr := []rune(c.Comment)
		if len(runeStr) > 1 {
			return nil, fmt.Errorf("csv_delimiter must be a single character, got: %s", c.Comment)
		}
	}

	if len(c.ColumnNames) > 0 && len(c.ColumnTypes) > 0 && len(c.ColumnNames) != len(c.ColumnTypes) {
		return nil, fmt.Errorf("csv_column_names field count doesn't match with csv_column_types")
	}

	c.gotColumnNames = len(c.ColumnNames) > 0

	if c.TimeFunc == nil {
		c.TimeFunc = time.Now
	}

	return &Parser{Config: c}, nil
}

func (p *Parser) SetTimeFunc(fn TimeFunc) {
	p.TimeFunc = fn
}

func (p *Parser) compile(r io.Reader) *csv.Reader {
	csvReader := csv.NewReader(r)
	// ensures that the reader reads records of different lengths without an error
	csvReader.FieldsPerRecord = -1
	if p.Delimiter != "" {
		csvReader.Comma = []rune(p.Delimiter)[0]
	}
	if p.Comment != "" {
		csvReader.Comment = []rune(p.Comment)[0]
	}
	csvReader.TrimLeadingSpace = p.TrimSpace
	return csvReader
}

func (p *Parser) Parse(buf []byte) ([]cua.Metric, error) {
	r := bytes.NewReader(buf)
	csvReader := p.compile(r)
	// skip first rows
	for i := 0; i < p.SkipRows; i++ {
		_, err := csvReader.Read()
		if err != nil {
			return nil, fmt.Errorf("csv read: %w", err)
		}
	}
	// if there is a header and we did not get DataColumns
	// set DataColumns to names extracted from the header
	if !p.gotColumnNames {
		headerNames := make([]string, 0)
		for i := 0; i < p.HeaderRowCount; i++ {
			header, err := csvReader.Read()
			if err != nil {
				return nil, fmt.Errorf("csv read: %w", err)
			}
			// concatenate header names
			for i := range header {
				name := header[i]
				if p.TrimSpace {
					name = strings.Trim(name, " ")
				}
				if len(headerNames) <= i {
					headerNames = append(headerNames, name)
				} else {
					headerNames[i] += name
				}
			}
		}
		p.ColumnNames = headerNames[p.SkipColumns:]
	} else {
		// if columns are named, just skip header rows
		for i := 0; i < p.HeaderRowCount; i++ {
			_, err := csvReader.Read()
			if err != nil {
				return nil, fmt.Errorf("csv read: %w", err)
			}
		}
	}

	table, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("csv readall: %w", err)
	}

	metrics := make([]cua.Metric, 0)
	for _, record := range table {
		m, err := p.parseRecord(record)
		if err != nil {
			return metrics, err
		}
		metrics = append(metrics, m)
	}
	return metrics, nil
}

// ParseLine does not use any information in header and assumes DataColumns is set
// it will also not skip any rows
func (p *Parser) ParseLine(line string) (cua.Metric, error) {
	r := bytes.NewReader([]byte(line))
	csvReader := p.compile(r)

	// if there is nothing in DataColumns, ParseLine will fail
	if len(p.ColumnNames) == 0 {
		return nil, fmt.Errorf("[parsers.csv] data columns must be specified")
	}

	record, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("csv read: %w", err)
	}
	m, err := p.parseRecord(record)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (p *Parser) parseRecord(record []string) (cua.Metric, error) {
	recordFields := make(map[string]interface{})
	tags := make(map[string]string)

	// skip columns in record
	record = record[p.SkipColumns:]
outer:
	for i, fieldName := range p.ColumnNames {
		if i < len(record) {
			value := record[i]
			if p.TrimSpace {
				value = strings.Trim(value, " ")
			}

			for _, tagName := range p.TagColumns {
				if tagName == fieldName {
					tags[tagName] = value
					continue outer
				}
			}

			// If the field name is the timestamp column, then keep field name as is.
			if fieldName == p.TimestampColumn {
				recordFields[fieldName] = value
				continue
			}

			// Try explicit conversion only when column types is defined.
			if len(p.ColumnTypes) > 0 {
				// Throw error if current column count exceeds defined types.
				if i >= len(p.ColumnTypes) {
					return nil, fmt.Errorf("column type: column count exceeded")
				}

				var val interface{}
				var err error

				switch p.ColumnTypes[i] {
				case "int":
					val, err = strconv.ParseInt(value, 10, 64)
					if err != nil {
						return nil, fmt.Errorf("column type: parse int error %w", err)
					}
				case "float":
					val, err = strconv.ParseFloat(value, 64)
					if err != nil {
						return nil, fmt.Errorf("column type: parse float error %w", err)
					}
				case "bool":
					val, err = strconv.ParseBool(value)
					if err != nil {
						return nil, fmt.Errorf("column type: parse bool error %w", err)
					}
				default:
					val = value
				}

				recordFields[fieldName] = val
				continue
			}

			// attempt type conversions
			if iValue, err := strconv.ParseInt(value, 10, 64); err == nil {
				recordFields[fieldName] = iValue
			} else if fValue, err := strconv.ParseFloat(value, 64); err == nil {
				recordFields[fieldName] = fValue
			} else if bValue, err := strconv.ParseBool(value); err == nil {
				recordFields[fieldName] = bValue
			} else {
				recordFields[fieldName] = value
			}
		}
	}

	// add default tags
	for k, v := range p.DefaultTags {
		tags[k] = v
	}

	// will default to plugin name
	measurementName := p.MetricName
	if p.MeasurementColumn != "" {
		if recordFields[p.MeasurementColumn] != nil && recordFields[p.MeasurementColumn] != "" {
			measurementName = fmt.Sprintf("%v", recordFields[p.MeasurementColumn])
		}
	}

	metricTime, err := parseTimestamp(p.TimeFunc, recordFields, p.TimestampColumn, p.TimestampFormat, p.Timezone)
	if err != nil {
		return nil, err
	}

	// Exclude `TimestampColumn` and `MeasurementColumn`
	delete(recordFields, p.TimestampColumn)
	delete(recordFields, p.MeasurementColumn)

	m, err := metric.New(measurementName, tags, recordFields, metricTime)
	if err != nil {
		return nil, fmt.Errorf("new metric: %w", err)
	}
	return m, nil
}

// ParseTimestamp return a timestamp, if there is no timestamp on the csv it
// will be the current timestamp, else it will try to parse the time according
// to the format.
func parseTimestamp(timeFunc func() time.Time, recordFields map[string]interface{},
	timestampColumn, timestampFormat string, timezone string,
) (time.Time, error) {
	if timestampColumn != "" {
		if recordFields[timestampColumn] == nil {
			return time.Time{}, fmt.Errorf("timestamp column: %v could not be found", timestampColumn)
		}

		switch timestampFormat {
		case "":
			return time.Time{}, fmt.Errorf("timestamp format must be specified")
		default:
			metricTime, err := internal.ParseTimestamp(timestampFormat, recordFields[timestampColumn], timezone)
			if err != nil {
				return time.Time{}, fmt.Errorf("parse timestamp: %w", err)
			}
			return metricTime, nil
		}
	}

	return timeFunc(), nil
}

// SetDefaultTags set the DefaultTags
func (p *Parser) SetDefaultTags(tags map[string]string) {
	p.DefaultTags = tags
}
