package cli

import (
	"testing"

	"github.com/stretchr/testify/require"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func TestCatalogValidate_NilCatalog(t *testing.T) {
	t.Parallel()
	var catalog *Catalog

	require.NotPanics(t, func() {
		err := catalog.Validate(nil)
		require.Error(t, err)
	})
}

func TestCatalogValidate_EmptyCatalog(t *testing.T) {
	t.Parallel()
	catalog := &Catalog{
		RuleTypes: []*minderv1.RuleType{},
		Profiles:  []*minderv1.Profile{},
	}

	err := catalog.Validate(nil)
	require.Error(t, err)
}

func TestCatalogValidate_WithValidRuleReference(t *testing.T) {
	t.Parallel()
	ruleType := &minderv1.RuleType{
		Name: "test-rule",
	}

	profile := &minderv1.Profile{
		Name: "test-profile",
		Repository: []*minderv1.Profile_Rule{
			{
				Type: "test-rule",
			},
		},
	}

	catalog := &Catalog{
		RuleTypes: []*minderv1.RuleType{ruleType},
		Profiles:  []*minderv1.Profile{profile},
	}

	err := catalog.Validate(func(string, ...any) {})
	require.NoError(t, err)
}

func TestCatalogValidate_SkipsProfilesWithMissingRuleTypes(t *testing.T) {
	t.Parallel()
	ruleType := &minderv1.RuleType{
		Name: "test-rule",
	}

	validProfile := &minderv1.Profile{
		Name: "valid-profile",
		Repository: []*minderv1.Profile_Rule{
			{
				Type: "test-rule",
			},
		},
	}

	invalidProfile := &minderv1.Profile{
		Name: "invalid-profile",
		Repository: []*minderv1.Profile_Rule{
			{
				Type: "missing-rule",
			},
		},
	}

	catalog := &Catalog{
		RuleTypes: []*minderv1.RuleType{ruleType},
		Profiles:  []*minderv1.Profile{validProfile, invalidProfile},
	}

	err := catalog.Validate(func(string, ...any) {})
	require.NoError(t, err)
	require.Len(t, catalog.Profiles, 1)
	require.Equal(t, "valid-profile", catalog.Profiles[0].Name)
}
