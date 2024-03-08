package bundles

import v1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"

type Metadata struct {
	BundleName string
	Version    string
	Namespace  string
	Profiles   []string
	RuleTypes  []string
}

type Bundle interface {
	GetMetadata() Metadata
	GetProfile(profileName string) (*v1.Profile, error)
	GetRuleType(ruleTypeName string) (*v1.RuleType, error)
}
