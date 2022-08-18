package circhttpjson

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCHJ_hasStreamtags(t *testing.T) {
	testDir := "testdata"
	tests := []struct {
		name    string
		file    string
		wantErr bool
	}{
		{"invalid json -- empty", filepath.Join(testDir, "invalid1.json"), true},
		{"invalid json -- no metrics", filepath.Join(testDir, "invalid2.json"), true},
		// the following format is valid for the broker, it is not valid for this plugin
		{"invalid json -- non-streamtag", filepath.Join(testDir, "invalid3.json"), true},
		{"invalid json -- non-streamtag idb sample", filepath.Join(testDir, "untagged-stats.json"), true},
		{"valid", filepath.Join(testDir, "valid1.json"), false},
		{"valid w/ts", filepath.Join(testDir, "valid2.json"), false},
		{"valid -- idb sample", filepath.Join(testDir, "tagged-stats.json"), false},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			chj := &CHJ{}
			data, err := os.ReadFile(tt.file)
			if err != nil {
				t.Fatalf("reading %s", err)
			}
			if err := chj.hasStreamtags(data); (err != nil) != tt.wantErr {
				t.Errorf("CHJ.hasStreamtags() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
