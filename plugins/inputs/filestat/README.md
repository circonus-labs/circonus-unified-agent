# filestat Input Plugin

The filestat plugin gathers metrics about file existence, size, and other stats.

### Configuration

```toml
# Read stats about given file(s)
[[inputs.filestat]]
  instance_id = "" # unique instance identifier (REQUIRED)

  ## Files to gather stats about.
  ## These accept standard unix glob matching rules, but with the addition of
  ## ** as a "super asterisk". See https://github.com/gobwas/glob.
  files = ["/etc/circonus-unified-agent/circonus-unified-agent.conf", "/var/log/**.log"]

  ## If true, read the entire file and calculate an md5 checksum.
  md5 = false
```

### Measurements & Fields

- filestat
    - exists (int, 0 | 1)
    - size_bytes (int, bytes)
    - modification_time (int, unix time nanoseconds)
    - md5 (optional, string)

### Tags

- All measurements have the following tags:
    - file (the path the to file, as specified in the config)

### Example Output

```
$ circonus-unified-agent --config /etc/circonus-unified-agent/circonus-unified-agent.conf --input-filter filestat --test
* Plugin: filestat, Collection 1
> filestat,file=/tmp/foo/bar,host=tyrion exists=0i 1507218518192154351
> filestat,file=/Users/sparrc/ws/circonus-unified-agent.conf,host=tyrion exists=1i,size=47894i,modification_time=1507152973123456789i  1507218518192154351
```
