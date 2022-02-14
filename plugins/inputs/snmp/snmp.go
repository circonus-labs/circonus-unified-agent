package snmp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/circonus-labs/circonus-unified-agent/cua"
	"github.com/circonus-labs/circonus-unified-agent/internal"
	circmgr "github.com/circonus-labs/circonus-unified-agent/internal/circonus"
	"github.com/circonus-labs/circonus-unified-agent/internal/snmp"
	"github.com/circonus-labs/circonus-unified-agent/plugins/inputs"
	"github.com/circonus-labs/go-trapmetrics"
	"github.com/gosnmp/gosnmp"
)

const description = `Retrieves SNMP values from remote agents`
const sampleConfig = `
  ## Agent addresses to retrieve values from.
  ##   example: agents = ["udp://127.0.0.1:161"]
  ##            agents = ["tcp://127.0.0.1:161"]
  agents = ["udp://127.0.0.1:161"]

  ## Timeout for each request.
  # timeout = "5s"

  ## SNMP version; can be 1, 2, or 3.
  # version = 2

  ## Agent host tag; the tag used to reference the source host
  # agent_host_tag = "agent_host"

  ## SNMP community string.
  # community = "public"

  ## Number of retries to attempt.
  # retries = 3

  ## The GETBULK max-repetitions parameter.
  # max_repetitions = 10

  ## SNMPv3 authentication and encryption options.
  ##
  ## Security Name.
  # sec_name = "myuser"
  ## Authentication protocol; one of "MD5", "SHA", or "".
  # auth_protocol = "MD5"
  ## Authentication password.
  # auth_password = "pass"
  ## Security Level; one of "noAuthNoPriv", "authNoPriv", or "authPriv".
  # sec_level = "authNoPriv"
  ## Context Name.
  # context_name = ""
  ## Privacy protocol used for encrypted messages; one of "DES", "AES" or "".
  # priv_protocol = ""
  ## Privacy password used for encrypted messages.
  # priv_password = ""

  ## Add fields and tables defining the variables you wish to collect.  This
  ## example collects the system uptime and interface variables.  Reference the
  ## full plugin documentation for configuration details.
`

// execCommand is so tests can mock out exec.Command usage.
var execCommand = exec.Command

// execCmd executes the specified command, returning the STDOUT content.
// If command exits with error status, the output is captured into the returned error.
func execCmd(arg0 string, args ...string) ([]byte, error) {
	// if wlog.LogLevel() == wlog.DEBUG {
	// 	quoted := make([]string, 0, len(args))
	// 	for _, arg := range args {
	// 		quoted = append(quoted, fmt.Sprintf("%q", arg))
	// 	}
	// 	log.Printf("D! [inputs.snmp] executing %q %s", arg0, strings.Join(quoted, " "))
	// }

	out, err := execCommand(arg0, args...).Output()
	if err != nil {
		var exiterr *exec.ExitError
		if errors.As(err, &exiterr) {
			return nil, fmt.Errorf("%s: %w", bytes.TrimRight(exiterr.Stderr, "\r\n"), exiterr)
		}
		return nil, fmt.Errorf("exec cmd: %w", err)
	}
	return out, nil
}

// Snmp holds the configuration for the plugin.
type Snmp struct {
	Log               cua.Logger
	metricDestination *trapmetrics.TrapMetrics // direct metrics mode - send directly to circonus (bypassing output)
	DebugAPI          *bool                    `toml:"debug_api"`     // direct metrics mode - send directly to circonus (bypassing output)
	TraceMetrics      *string                  `toml:"trace_metrics"` // direct metrics mode - send directly to circonus (bypassing output)
	FlushDelay        string                   `toml:"flush_delay"`   // direct metrics mode - send directly to circonus (bypassing output)
	Broker            string                   `toml:"broker"`        // direct metrics mode - send directly to circonus (bypassing output)
	Name              string                   // Name & Fields are the elements of a Table.
	AgentHostTag      string                   `toml:"agent_host_tag"` // The tag used to name the agent host
	InstanceID        string                   `toml:"instance_id"`    // direct metrics mode - send directly to circonus (bypassing output)
	Tables            []Table                  `toml:"table"`
	Fields            []Field                  `toml:"field"` // Name & Fields are the elements of a Table. agent chokes if we try to embed a Table. So instead we have to embed the fields of a Table, and construct a Table during runtime.
	connectionCache   []snmpConnection
	Agents            []string `toml:"agents"`
	Tags              map[string]string
	snmp.ClientConfig
	flushDelay     time.Duration // direct metrics mode - send directly to circonus (bypassing output)
	FlushPoolSize  uint          `toml:"flush_pool_size"`
	FlushQueueSize uint          `toml:"flush_queue_size"`
	DirectMetrics  bool          `toml:"direct_metrics"` // direct metrics mode - send directly to circonus (bypassing output)
	initialized    bool
}

func (s *Snmp) init() error {
	if s.initialized {
		return nil
	}

	if s.DirectMetrics {
		opts := &circmgr.MetricDestConfig{
			MetricMeta: circmgr.MetricMeta{
				PluginID:   "snmp",
				InstanceID: s.InstanceID,
			},
			Broker:       s.Broker,
			DebugAPI:     s.DebugAPI,
			TraceMetrics: s.TraceMetrics,
		}
		dest, err := circmgr.NewMetricDestination(opts, s.Log)
		if err != nil {
			return fmt.Errorf("new metric destination: %w", err)
		}

		s.metricDestination = dest
		// s.Log.Info("using Direct Metrics mode")

		if s.FlushDelay != "" {
			fd, err := time.ParseDuration(s.FlushDelay)
			if err != nil {
				return fmt.Errorf("parsing flush_delay (%s): %w", s.FlushDelay, err)
			}
			s.flushDelay = fd
		}
		initFlusherPool(s.Log, s.FlushPoolSize, s.FlushQueueSize)
	}

	s.connectionCache = make([]snmpConnection, len(s.Agents))

	for i := range s.Tables {
		if err := s.Tables[i].Init(); err != nil {
			return fmt.Errorf("initializing table %s: %w", s.Tables[i].Name, err)
		}
	}

	for i := range s.Fields {
		if err := s.Fields[i].init(); err != nil {
			return fmt.Errorf("initializing field %s: %w", s.Fields[i].Name, err)
		}
	}

	if len(s.AgentHostTag) == 0 {
		s.AgentHostTag = "agent_host"
	}

	s.initialized = true
	return nil
}

// Table holds the configuration for a SNMP table.
type Table struct {
	Name        string   // Name will be the name of the measurement.
	Oid         string   // OID for automatic field population. If provided, init() will populate Fields with all the table columns of the given OID.
	InheritTags []string // Which tags to inherit from the top-level config.
	Fields      []Field  `toml:"field"` // Fields is the tags and values to look up.
	IndexAsTag  bool     // Adds each row's table index as a tag.
	initialized bool
}

// Init() builds & initializes the nested fields.
func (t *Table) Init() error {
	if t.initialized {
		return nil
	}

	if err := t.initBuild(); err != nil {
		return err
	}

	// initialize all the nested fields
	for i := range t.Fields {
		if err := t.Fields[i].init(); err != nil {
			return fmt.Errorf("initializing field %s: %w", t.Fields[i].Name, err)
		}
	}

	t.initialized = true
	return nil
}

// initBuild initializes the table if it has an OID configured. If so, the
// net-snmp tools will be used to look up the OID and auto-populate the table's
// fields.
func (t *Table) initBuild() error {
	if t.Oid == "" {
		return nil
	}

	_, _, oidText, fields, err := snmpTable(t.Oid)
	if err != nil {
		return err
	}

	if t.Name == "" {
		t.Name = oidText
	}

	knownOIDs := map[string]bool{}
	for _, f := range t.Fields {
		knownOIDs[f.Oid] = true
	}
	for _, f := range fields {
		if !knownOIDs[f.Oid] {
			t.Fields = append(t.Fields, f)
		}
	}

	return nil
}

// Field holds the configuration for a Field to look up.
type Field struct {
	// Name will be the name of the field.
	Name string
	// OID is prefix for this field. The plugin will perform a walk through all
	// OIDs with this as their parent. For each value found, the plugin will strip
	// off the OID prefix, and use the remainder as the index. For multiple fields
	// to show up in the same row, they must share the same index.
	Oid string
	// OidIndexSuffix is the trailing sub-identifier on a table record OID that will be stripped off to get the record's index.
	OidIndexSuffix string
	// Conversion controls any type conversion that is done on the value.
	//  "float"/"float(0)" will convert the value into a float.
	//  "float(X)" will convert the value into a float, and then move the decimal before Xth right-most digit.
	//  "int" will conver the value into an integer.
	//  "hwaddr" will convert a 6-byte string to a MAC address.
	//  "ipaddr" will convert the value to an IPv4 or IPv6 address.
	//  "" or "string" byte slice will be returned as string if it contains only printable runes
	//                 otherwise it will encoded as hex
	//                 if it is not a byte slice, it will be returned as-is
	Conversion string
	// OidIndexLength specifies the length of the index in OID path segments. It can be used to remove sub-identifiers that vary in content or length.
	OidIndexLength int
	// IsTag controls whether this OID is output as a tag or a value.
	IsTag bool
	// TextMetric controls whether this metric (if SYNTAX INTEGER) is sent as both an int and a textual representation
	TextMetric bool
	// Translate tells if the value of the field should be snmptranslated
	Translate   bool
	initialized bool
}

// init() converts OID names to numbers, and sets the .Name attribute if unset.
func (f *Field) init() error {
	if f.initialized {
		return nil
	}

	stc := TranslateOID(f.Oid)
	if stc.err != nil {
		return fmt.Errorf("translating: %w", stc.err)
	}

	f.Oid = stc.oidNum

	if f.Name == "" {
		f.Name = stc.oidText
	}

	if f.Conversion == "" {
		f.Conversion = stc.conversion
	}

	// if tag and able to parse a textual convention from oid translation
	// for SYNTAX INTEGER fields e.g. ifType
	if (f.IsTag || f.TextMetric) && len(stc.valMap) > 0 {
		if f.Conversion == "" {
			f.Conversion = "lookup"
		}
	}

	f.initialized = true
	return nil
}

// RTable is the resulting table built from a Table.
type RTable struct {
	// Name is the name of the field, copied from Table.Name.
	Name string
	// Time is the time the table was built.
	Time time.Time
	// Rows are the rows that were found, one row for each table OID index found.
	Rows []RTableRow
}

// RTableRow is the resulting row containing all the OID values which shared
// the same index.
type RTableRow struct {
	// Tags are all the Field values which had IsTag=true.
	Tags map[string]string
	// Fields are all the Field values which had IsTag=false.
	Fields map[string]interface{}
}

type walkError struct {
	err error
	msg string
}

func (e *walkError) Error() string {
	return e.msg
}

func (e *walkError) Unwrap() error {
	return e.err
}

func init() {
	inputs.Add("snmp", func() cua.Input {
		return &Snmp{
			Name: "snmp",
			ClientConfig: snmp.ClientConfig{
				Retries:        3,
				MaxRepetitions: 10,
				Timeout:        internal.Duration{Duration: 5 * time.Second},
				Version:        2,
				Community:      "public",
			},
		}
	})
}

// SampleConfig returns the default configuration of the input.
func (s *Snmp) SampleConfig() string {
	return sampleConfig
}

// Description returns a one-sentence description on the input.
func (s *Snmp) Description() string {
	return description
}

func isDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// Gather retrieves all the configured fields and tables.
// Any error encountered does not halt the process. The errors are accumulated
// and returned at the end.
func (s *Snmp) Gather(ctx context.Context, acc cua.Accumulator) error {
	gstart := time.Now()

	if err := s.init(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	topDMTags := make(map[string]string)
	for i, agent := range s.Agents {
		wg.Add(1)
		go func(i int, agent string) {
			defer wg.Done()
			gs, err := s.getConnection(i)

			// test for not re-using connections
			defer func(agent string) {
				err := gs.Close()
				if err != nil {
					s.Log.Errorf("closing snmp conn: %s (%s)", err, agent)
				}
			}(agent)

			if err != nil {
				acc.AddError(fmt.Errorf("agent %s: %w", agent, err))
				return
			}

			if isDone(ctx) {
				return
			}

			// First is the top-level fields. We treat the fields as table prefixes with an empty index.
			t := Table{
				Name:   s.Name,
				Fields: s.Fields,
			}
			topTags := map[string]string{}

			if err := s.gatherTable(acc, gs, t, topTags, false); err != nil {
				acc.AddError(fmt.Errorf("agent %s: %w", agent, err))
			}
			topDMTags = topTags

			if isDone(ctx) {
				return
			}

			// Now is the real tables.
			for _, t := range s.Tables {
				if err := s.gatherTable(acc, gs, t, topTags, true); err != nil {
					acc.AddError(fmt.Errorf("agent %s: gathering table %s: %w", agent, t.Name, err))
				}
				if isDone(ctx) {
					return
				}
			}
		}(i, agent)
	}
	wg.Wait()

	stats := map[string]interface{}{"dur_snmp_get": time.Since(gstart).Seconds()}
	stags := map[string]string{"units": "seconds"}
	dmtags := trapmetrics.Tags{trapmetrics.Tag{Category: "units", Value: "seconds"}}
	for k, v := range topDMTags {
		dmtags = append(dmtags, trapmetrics.Tag{Category: k, Value: v})
		stags[k] = v
	}

	if s.DirectMetrics && s.metricDestination != nil {
		_ = s.metricDestination.GaugeSet("dur_snmp_get", dmtags, time.Since(gstart).Seconds(), nil)
		flusherPool.traps <- trap{
			name: s.InstanceID,
			ctx:  ctx,
			dest: s.metricDestination,
			tags: dmtags,
		}
		// if s.flushDelay > time.Duration(0) {
		// 	spent := time.Since(gstart)
		// 	if spent < s.flushDelay {
		// 		delay := s.flushDelay - spent
		// 		fd := internal.RandomDuration(delay)
		// 		s.Log.Debugf("flush delay: %s", fd)
		// 		select {
		// 		case <-ctx.Done():
		// 		case <-time.After(fd):
		// 		}
		// 	} else {
		// 		s.Log.Debugf("flush delay: 0 - snmp get (%s) took longer than %s", spent, s.flushDelay)
		// 	}
		// }
		// fstart := time.Now()
		// if _, err := s.metricDestination.Flush(ctx); err != nil {
		// 	s.Log.Warn(err)
		// }
		// _ = s.metricDestination.GaugeSet("dur_last_submit", dmtags, time.Since(fstart).Seconds(), nil)
	}

	if s.DirectMetrics && s.metricDestination != nil {
		_ = s.metricDestination.GaugeSet("dur_last_gather", dmtags, time.Since(gstart).Seconds(), nil)
	} else {
		stats["dur_gather"] = time.Since(gstart).Seconds()
		acc.AddFields("snmp", stats, stags, time.Now())
	}

	return nil
}

func (s *Snmp) gatherTable(acc cua.Accumulator, gs snmpConnection, t Table, topTags map[string]string, walk bool) error {
	rt, err := t.Build(gs, walk)
	if err != nil {
		return err
	}

	for _, tr := range rt.Rows {
		if !walk {
			// top-level table. Add tags to topTags.
			for k, v := range tr.Tags {
				topTags[k] = v
			}
		} else {
			// real table. Inherit any specified tags.
			for _, k := range t.InheritTags {
				if v, ok := topTags[k]; ok {
					tr.Tags[k] = v
				}
			}
		}
		if _, ok := tr.Tags[s.AgentHostTag]; !ok {
			tr.Tags[s.AgentHostTag] = gs.Host()
		}

		if s.DirectMetrics && s.metricDestination != nil {
			for metricName, val := range tr.Fields {
				if err := circmgr.AddMetricToDest(s.metricDestination, "snmp_dm", rt.Name, metricName, tr.Tags, s.Tags, val, rt.Time); err != nil {
					s.Log.Warnf("adding %s: %s", metricName, err)
				}
			}
		} else {
			acc.AddFields(rt.Name, tr.Fields, tr.Tags, rt.Time)
		}
	}

	return nil
}

// Build retrieves all the fields specified in the table and constructs the RTable.
func (t Table) Build(gs snmpConnection, walk bool) (*RTable, error) {
	rows := map[string]RTableRow{}

	tagCount := 0
	for _, f := range t.Fields {
		f := f

		if f.IsTag {
			tagCount++
		}

		if len(f.Oid) == 0 {
			return nil, fmt.Errorf("cannot have empty OID on field %s", f.Name)
		}
		var oid string
		if f.Oid[0] == '.' {
			oid = f.Oid
		} else {
			// make sure OID has "." because the BulkWalkAll results do, and the prefix needs to match
			oid = "." + f.Oid
		}

		// ifv contains a mapping of table OID index to field value
		ifv := map[string]interface{}{}

		if !walk {
			// This is used when fetching non-table fields. Fields configured a the top
			// scope of the plugin.
			// We fetch the fields directly, and add them to ifv as if the index were an
			// empty string. This results in all the non-table fields sharing the same
			// index, and being added on the same row.
			if pkt, err := gs.Get([]string{oid}); err != nil {
				return nil, fmt.Errorf("performing get on field %s (oid:%s): %w", f.Name, oid, err)
			} else if pkt != nil && len(pkt.Variables) > 0 && pkt.Variables[0].Type != gosnmp.NoSuchObject && pkt.Variables[0].Type != gosnmp.NoSuchInstance {
				ent := pkt.Variables[0]
				fv, err := fieldConvert(f.Conversion, ent)
				if err != nil {
					return nil, fmt.Errorf("converting %q (OID %s) for field %s: %w", ent.Value, ent.Name, f.Name, err)
				}
				ifv[""] = fv
			}
		} else {
			err := gs.Walk(oid, func(ent gosnmp.SnmpPDU) error {
				if len(ent.Name) <= len(oid) || ent.Name[:len(oid)+1] != oid+"." {
					return &walkError{} // break the walk
				}

				idx := ent.Name[len(oid):]
				if f.OidIndexSuffix != "" {
					if !strings.HasSuffix(idx, f.OidIndexSuffix) {
						// this entry doesn't match our OidIndexSuffix. skip it
						return nil
					}
					idx = idx[:len(idx)-len(f.OidIndexSuffix)]
				}
				if f.OidIndexLength != 0 {
					i := f.OidIndexLength + 1 // leading separator
					idx = strings.Map(func(r rune) rune {
						if r == '.' {
							i--
						}
						if i < 1 {
							return -1
						}
						return r
					}, idx)
				}

				// snmptranslate table field value here
				if f.Translate {
					if entOid, ok := ent.Value.(string); ok {
						stc := TranslateOID(entOid)
						if stc.err == nil {
							// If no error translating, the original value for ent.Value should be replaced
							ent.Value = stc.oidText
						}
					}
				}

				if f.TextMetric && f.Conversion == "lookup" {
					fv, err := fieldConvert("int", ent)
					if err != nil {
						return &walkError{
							msg: fmt.Sprintf("converting %q (OID %s) for field %s", ent.Value, ent.Name, f.Name),
							err: err,
						}
					}
					ifv[idx] = fv

					fv, err = fieldConvert(f.Conversion, ent)
					if err != nil {
						return &walkError{
							msg: fmt.Sprintf("converting %q (OID %s) for field %s", ent.Value, ent.Name, f.Name),
							err: err,
						}
					}
					ifv[idx+"_desc"] = fv

				} else {
					fv, err := fieldConvert(f.Conversion, ent)
					if err != nil {
						return &walkError{
							msg: fmt.Sprintf("converting %q (OID %s) for field %s", ent.Value, ent.Name, f.Name),
							err: err,
						}
					}
					ifv[idx] = fv
				}

				return nil
			})
			if err != nil {
				// Our callback always wraps errors in a walkError.
				// If this error isn't a walkError, we know it's not
				// from the callback
				var werr *walkError
				if !errors.As(err, &werr) {
					return nil, fmt.Errorf("performing bulk walk for field %s: %w", f.Name, err)
				}
			}
		}

		for idx, v := range ifv {
			rtr, ok := rows[idx]
			if !ok {
				rtr = RTableRow{}
				rtr.Tags = map[string]string{}
				rtr.Fields = map[string]interface{}{}
				rows[idx] = rtr
			}
			if t.IndexAsTag && idx != "" {
				if idx[0] == '.' {
					idx = idx[1:]
				}
				rtr.Tags["index"] = idx
			}
			// don't add an empty string
			if vs, ok := v.(string); !ok || vs != "" {
				if f.IsTag {
					if ok {
						rtr.Tags[f.Name] = vs
					} else {
						rtr.Tags[f.Name] = fmt.Sprintf("%v", v)
					}
				} else {
					mname := f.Name
					if strings.HasSuffix(idx, "_desc") {
						rtr.Fields[mname+"_desc"] = v
					} else {
						rtr.Fields[mname] = v
					}
				}
			}
		}
	}

	rt := RTable{
		Name: t.Name,
		Time: time.Now(), // TODO record time at start
		Rows: make([]RTableRow, 0, len(rows)),
	}
	for _, r := range rows {
		rt.Rows = append(rt.Rows, r)
	}
	return &rt, nil
}

// snmpConnection is an interface which wraps a *gosnmp.GoSNMP object.
// We interact through an interface so we can mock it out in tests.
type snmpConnection interface {
	Host() string
	// BulkWalkAll(string) ([]gosnmp.SnmpPDU, error)
	Walk(string, gosnmp.WalkFunc) error
	Get(oids []string) (*gosnmp.SnmpPacket, error)
	Close() error
}

// getConnection creates a snmpConnection (*gosnmp.GoSNMP) object and caches the
// result using `agentIndex` as the cache key.  This is done to allow multiple
// connections to a single address.  It is an error to use a connection in
// more than one goroutine.
func (s *Snmp) getConnection(idx int) (snmpConnection, error) {
	if gs := s.connectionCache[idx]; gs != nil {
		return gs, nil
	}

	agent := s.Agents[idx]

	var err error
	var gs snmp.GosnmpWrapper
	gs, err = snmp.NewWrapper(s.ClientConfig)
	if err != nil {
		return nil, fmt.Errorf("new wrapper: %w", err)
	}
	_ = gs.SetAgent(agent)
	if err != nil {
		return nil, fmt.Errorf("set agent: %w", err)
	}

	// cache connection for reuse
	// s.connectionCache[idx] = gs

	if err := gs.Connect(); err != nil {
		return nil, fmt.Errorf("setting up connection: %w", err)
	}

	return gs, nil
}

// fieldConvert converts from any type according to the conv specification
//  "float"/"float(0)" will convert the value into a float.
//  "float(X)" will convert the value into a float, and then move the decimal before Xth right-most digit.
//  "int" will convert the value into an integer.
//  "hwaddr" will convert the value into a MAC address.
//  "ipaddr" will convert the value into into an IP address.
//  "" or "string" will convert a byte slice into a string (if all runes are printable, otherwise it will return a hex string)
//                 if the value is not a byte slice, it is returned as-is
func fieldConvert(conv string, sv gosnmp.SnmpPDU) (interface{}, error) {
	v := sv.Value

	if conv == "" || conv == "string" {
		if bs, ok := v.([]byte); ok {
			for _, b := range bs {
				if !unicode.IsPrint(rune(b)) {
					return hex.EncodeToString(bs), nil
				}
			}
			return string(bs), nil
		}
		return v, nil
	}

	if conv == "lookup" {
		stc := TranslateOID(sv.Name)
		key := fmt.Sprintf("%d", v)
		if stc.valMap != nil {
			if val, ok := stc.valMap[key]; ok {
				return val, nil
			}
			return key, nil
		}
		return key, nil
	}

	var d int
	if _, err := fmt.Sscanf(conv, "float(%d)", &d); err == nil || conv == "float" {
		switch vt := v.(type) {
		case float32:
			v = float64(vt) / math.Pow10(d)
		case float64:
			v = vt / math.Pow10(d)
		case int:
			v = float64(vt) / math.Pow10(d)
		case int8:
			v = float64(vt) / math.Pow10(d)
		case int16:
			v = float64(vt) / math.Pow10(d)
		case int32:
			v = float64(vt) / math.Pow10(d)
		case int64:
			v = float64(vt) / math.Pow10(d)
		case uint:
			v = float64(vt) / math.Pow10(d)
		case uint8:
			v = float64(vt) / math.Pow10(d)
		case uint16:
			v = float64(vt) / math.Pow10(d)
		case uint32:
			v = float64(vt) / math.Pow10(d)
		case uint64:
			v = float64(vt) / math.Pow10(d)
		case []byte:
			vf, _ := strconv.ParseFloat(string(vt), 64)
			v = vf / math.Pow10(d)
		case string:
			vf, _ := strconv.ParseFloat(vt, 64)
			v = vf / math.Pow10(d)
		}
		return v, nil
	}

	if conv == "int" {
		switch vt := v.(type) {
		case float32:
			v = int64(vt)
		case float64:
			v = int64(vt)
		case int:
			v = int64(vt)
		case int8:
			v = int64(vt)
		case int16:
			v = int64(vt)
		case int32:
			v = int64(vt)
		case int64:
			v = vt
		case uint:
			v = int64(vt)
		case uint8:
			v = int64(vt)
		case uint16:
			v = int64(vt)
		case uint32:
			v = int64(vt)
		case uint64:
			v = int64(vt)
		case []byte:
			v, _ = strconv.ParseInt(string(vt), 10, 64)
		case string:
			v, _ = strconv.ParseInt(vt, 10, 64)
		}
		return v, nil
	}

	if conv == "hwaddr" {
		switch vt := v.(type) {
		case string:
			v = net.HardwareAddr(vt).String()
		case []byte:
			v = net.HardwareAddr(vt).String()
		default:
			return nil, fmt.Errorf("invalid type (%T) for hwaddr conversion", v)
		}
		return v, nil
	}

	if conv == "ipaddr" {
		var ipbs []byte

		switch vt := v.(type) {
		case string:
			ipbs = []byte(vt)
		case []byte:
			ipbs = vt
		default:
			return nil, fmt.Errorf("invalid type (%T) for ipaddr conversion", v)
		}

		switch len(ipbs) {
		case 4, 16:
			v = net.IP(ipbs).String()
		default:
			return nil, fmt.Errorf("invalid length (%d) for ipaddr conversion", len(ipbs))
		}

		return v, nil
	}

	return nil, fmt.Errorf("invalid conversion type '%s'", conv)
}

type snmpTableCache struct {
	err     error
	mibName string
	oidNum  string
	oidText string
	fields  []Field
}

var snmpTableCaches map[string]snmpTableCache
var snmpTableCachesLock sync.Mutex

// snmpTable resolves the given OID as a table, providing information about the
// table and fields within.
func snmpTable(oid string) (mibName string, oidNum string, oidText string, fields []Field, err error) {
	snmpTableCachesLock.Lock()
	if snmpTableCaches == nil {
		snmpTableCaches = map[string]snmpTableCache{}
	}

	var stc snmpTableCache
	var ok bool
	if stc, ok = snmpTableCaches[oid]; !ok {
		stc.mibName, stc.oidNum, stc.oidText, stc.fields, stc.err = snmpTableCall(oid)
		snmpTableCaches[oid] = stc
	}

	snmpTableCachesLock.Unlock()
	return stc.mibName, stc.oidNum, stc.oidText, stc.fields, stc.err
}

func snmpTableCall(oid string) (mibName string, oidNum string, oidText string, fields []Field, err error) {
	stc := TranslateOID(oid)
	if stc.err != nil {
		return "", "", "", nil, fmt.Errorf("translating: %w", stc.err)
	}

	mibName = stc.mibName
	oidNum = stc.oidNum
	oidText = stc.oidText

	mibPrefix := mibName + "::"
	oidFullName := mibPrefix + stc.oidText

	// first attempt to get the table's tags
	tagOids := map[string]struct{}{}
	// We have to guess that the "entry" oid is `oid+".1"`. snmptable and snmptranslate don't seem to have a way to provide the info.
	if out, err := execCmd("snmptranslate", "-Td", oidFullName+".1"); err == nil {
		scanner := bufio.NewScanner(bytes.NewBuffer(out))
		for scanner.Scan() {
			line := scanner.Text()

			if !strings.HasPrefix(line, "  INDEX") {
				continue
			}

			i := strings.Index(line, "{ ")
			if i == -1 { // parse error
				continue
			}
			line = line[i+2:]
			i = strings.Index(line, " }")
			if i == -1 { // parse error
				continue
			}
			line = line[:i]
			for _, col := range strings.Split(line, ", ") {
				tagOids[mibPrefix+col] = struct{}{}
			}
		}
	}

	// this won't actually try to run a query. The `-Ch` will just cause it to dump headers.
	out, err := execCmd("snmptable", "-Ch", "-Cl", "-c", "public", "127.0.0.1", oidFullName)
	if err != nil {
		return "", "", "", nil, fmt.Errorf("getting table columns: %w", err)
	}
	scanner := bufio.NewScanner(bytes.NewBuffer(out))
	scanner.Scan()
	cols := scanner.Text()
	if len(cols) == 0 {
		return "", "", "", nil, fmt.Errorf("could not find any columns in table")
	}
	for _, col := range strings.Split(cols, " ") {
		if len(col) == 0 {
			continue
		}
		_, isTag := tagOids[mibPrefix+col]
		fields = append(fields, Field{Name: col, Oid: mibPrefix + col, IsTag: isTag})
	}

	return mibName, oidNum, oidText, fields, err
}

type TranslateItem struct {
	err        error
	valMap     map[string]string
	mibName    string
	oidNum     string
	oidText    string
	conversion string
}

var snmpTranslateCachesLock sync.Mutex
var snmpTranslateCache map[string]TranslateItem

// TranslateOID resolves the given OID.
func TranslateOID(oid string) TranslateItem {
	snmpTranslateCachesLock.Lock()
	if snmpTranslateCache == nil {
		snmpTranslateCache = map[string]TranslateItem{}
	}

	var stc TranslateItem
	var ok bool
	if stc, ok = snmpTranslateCache[oid]; !ok {
		// This will result in only one call to snmptranslate running at a time.
		// We could speed it up by putting a lock in snmpTranslateCache and then
		// returning it immediately, and multiple callers would then release the
		// snmpTranslateCachesLock and instead wait on the individual
		// snmpTranslation.Lock to release. But I don't know that the extra complexity
		// is worth it. Especially when it would slam the system pretty hard if lots
		// of lookups are being performed.

		stcp := snmpTranslateCall(oid)
		snmpTranslateCache[oid] = *stcp
		stc = *stcp
	}

	snmpTranslateCachesLock.Unlock()

	return stc
}

func TranslateForce(oid string, mibName string, oidNum string, oidText string, conversion string) {
	snmpTranslateCachesLock.Lock()
	defer snmpTranslateCachesLock.Unlock()
	if snmpTranslateCache == nil {
		snmpTranslateCache = map[string]TranslateItem{}
	}

	var stc TranslateItem
	stc.mibName = mibName
	stc.oidNum = oidNum
	stc.oidText = oidText
	stc.conversion = conversion
	stc.err = nil
	snmpTranslateCache[oid] = stc
}

func TranslateClear() {
	snmpTranslateCachesLock.Lock()
	defer snmpTranslateCachesLock.Unlock()
	snmpTranslateCache = map[string]TranslateItem{}
}

func snmpTranslateCall(oid string) *TranslateItem {
	stc := &TranslateItem{
		oidNum:  oid,
		oidText: oid,
	}

	var err error
	var out []byte
	if strings.ContainsAny(oid, ":abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		out, err = execCmd("snmptranslate", "-Td", "-Ob", oid)
	} else {
		out, err = execCmd("snmptranslate", "-Td", "-Ob", "-m", "all", oid)
		var exiterr *exec.Error
		if errors.As(err, &exiterr) && errors.Is(exiterr.Err, exec.ErrNotFound) {
			// Silently discard error if snmptranslate not found and we have a numeric OID.
			// Meaning we can get by without the lookup.
			return stc
		}
	}
	if err != nil {
		return &TranslateItem{err: err}
	}

	scanner := bufio.NewScanner(bytes.NewBuffer(out))
	ok := scanner.Scan()
	if !ok && scanner.Err() != nil {
		return &TranslateItem{err: fmt.Errorf("getting OID text: %w", scanner.Err())}
	}

	oidText := scanner.Text()

	i := strings.Index(oidText, "::")
	if i == -1 {
		// was not found in MIB.
		if bytes.Contains(out, []byte("[TRUNCATED]")) {
			return stc
		}
		// not truncated, but not fully found. We still need to parse out numeric OID, so keep going
		oidText = oid
		stc.oidText = oid
	} else {
		stc.mibName = oidText[:i]
		stc.oidText = oidText[i+2:]
	}

	stc.oidNum = ""

	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "  -- TEXTUAL CONVENTION "):
			tc := strings.TrimPrefix(line, "  -- TEXTUAL CONVENTION ")
			switch tc {
			case "MacAddress", "PhysAddress":
				stc.conversion = "hwaddr"
			case "InetAddressIPv4", "InetAddressIPv6", "InetAddress", "IPSIpAddress":
				stc.conversion = "ipaddr"
			}
		case strings.HasPrefix(line, "  SYNTAX	INTEGER {"):
			items := strings.TrimPrefix(line, "  SYNTAX	INTEGER {")
			items = strings.TrimSuffix(items, "}")
			stc.valMap = make(map[string]string)
			for _, item := range strings.Split(items, ",") {
				item = strings.TrimSpace(item)
				if len(item) == 0 {
					continue
				}
				lp := strings.Index(item, "(")
				rp := strings.Index(item, ")")
				if lp != -1 && rp != -1 {
					// e.g. ethernetCsmacd(6) key=6, value=ethernetCsmacd
					itemValue := item[:lp]
					itemKey := item[lp+1 : rp]
					stc.valMap[itemKey] = itemValue
				}
			}
		case strings.HasPrefix(line, "::= { "):
			objs := strings.TrimPrefix(line, "::= { ")
			objs = strings.TrimSuffix(objs, " }")

			for _, obj := range strings.Split(objs, " ") {
				if len(obj) == 0 {
					continue
				}
				i := strings.Index(obj, "(")
				j := strings.Index(obj, ")")
				if i != -1 && j != -1 {
					elem := obj[i+1 : j]
					stc.oidNum += "." + elem
				} else {
					stc.oidNum += "." + obj
				}
			}
		}
	}

	return stc
}
