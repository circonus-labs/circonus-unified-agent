package circonus

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/circonus-labs/go-apiclient"
	apiclicfg "github.com/circonus-labs/go-apiclient/config"
)

// Check bundle config caching

// loadCheckConfig will determine if caching is enabled and attempt to load the check bundle, returns check bundle and whether successful.
func loadCheckConfig(id string) (*apiclient.CheckBundle, bool) {
	if !ch.circCfg.CacheConfigs {
		return nil, false
	}
	if id == "" || ch.circCfg.CacheDir == "" {
		return nil, false
	}

	path := ch.circCfg.CacheDir
	checkConfigFile := filepath.Join(path, strings.ReplaceAll(id, ":", "_")+".json")

	data, err := os.ReadFile(checkConfigFile)
	if err != nil {
		ch.logger.Warnf("unable to read %s: %s", checkConfigFile, err)
		return nil, false
	}

	var bundle apiclient.CheckBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		ch.logger.Warnf("parsing check config %s: %s", checkConfigFile, err)
		return nil, false
	}
	ch.logger.Infof("using cached config: %s - %s", checkConfigFile, bundle.Config[apiclicfg.SubmissionURL])

	return &bundle, true
}

// saveCheckConfig will determine if caching is enabled and attempt to save the check bundle as a json blob.
func saveCheckConfig(id string, bundle *apiclient.CheckBundle) {
	if !ch.circCfg.CacheConfigs {
		return
	}
	if id == "" || ch.circCfg.CacheDir == "" {
		return
	}
	if bundle == nil {
		return
	}

	path := ch.circCfg.CacheDir
	checkConfigFile := filepath.Join(path, strings.ReplaceAll(id, ":", "_")+".json")

	data, err := json.Marshal(bundle)
	if err != nil {
		ch.logger.Warnf("marshal check conf: %s", err)
		return
	}

	if err := os.WriteFile(checkConfigFile, data, 0644); err != nil { //nolint:gosec
		ch.logger.Warnf("save check conf %s: %s", checkConfigFile, err)
		return
	}
	ch.logger.Infof("saved check config to cache: %s", checkConfigFile)
}
