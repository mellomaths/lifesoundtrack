package config

import (
	"testing"
)

func TestParseMetadataFeatureFlag(t *testing.T) {
	k := t.Name() + "_LST"
	t.Setenv(k, "false")
	if parseMetadataFeatureFlag(k) {
		t.Fatal("false should disable")
	}
	t.Setenv(k, "0")
	if parseMetadataFeatureFlag(k) {
		t.Fatal("0 should disable")
	}
	t.Setenv(k, "Off")
	if parseMetadataFeatureFlag(k) {
		t.Fatal("off should disable")
	}
	t.Setenv(k, "true")
	if !parseMetadataFeatureFlag(k) {
		t.Fatal("true should enable")
	}
	t.Setenv(k, "1")
	if !parseMetadataFeatureFlag(k) {
		t.Fatal("1 should enable")
	}
	t.Setenv(k, "")
	if !parseMetadataFeatureFlag(k) {
		t.Fatal("empty value should be enabled (opt-out default)")
	}
}
