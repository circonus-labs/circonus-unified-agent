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
func loadCheckConfig(id string) *apiclient.CheckBundle {
	if !ch.circCfg.CacheConfigs {
		return nil
	}
	if id == "" || ch.circCfg.CacheDir == "" {
		return nil
	}

	path := ch.circCfg.CacheDir
	checkConfigFile := filepath.Join(path, strings.ReplaceAll(id, ":", "_")+".json")

	data, err := os.ReadFile(checkConfigFile)
	if err != nil {
		if !os.IsNotExist(err) {
			ch.logger.Warnf("unable to read %s: %s", checkConfigFile, err)
		}
		return nil
	}

	var bundle apiclient.CheckBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		ch.logger.Warnf("parsing check config %s: %s", checkConfigFile, err)
		return nil
	}
	ch.logger.Infof("using cached config: %s - %s", checkConfigFile, bundle.Config[apiclicfg.SubmissionURL])

	return &bundle
}

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
}
