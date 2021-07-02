package release

const (
	// NAME is the name of this application.
	NAME = "circonus-unified-agent"
	// ENVPREFIX is the environment variable prefix.
	ENVPREFIX = "CUA"
)

// defined during build (e.g. goreleaser, see .goreleaser.yml).
var (
	// branch of relase in git repo.
	branch = "undef"
	// commit of relase in git repo.
	commit = "undef"
	// buildDate of release.
	buildDate = "undef"
	// buildTag of release.
	buildTag = "none"
	// version of the release.
	version = "Dev"
)

// Info contains release information.
type Info struct {
	Name      string
	Version   string
	Branch    string
	Commit    string
	BuildDate string
	BuildTag  string
	EnvPrefix string
}

func GetInfo() *Info {
	return &Info{
		Name:      NAME,
		Version:   version,
		Branch:    branch,
		Commit:    commit,
		BuildDate: buildDate,
		BuildTag:  buildTag,
		EnvPrefix: ENVPREFIX,
	}
}
