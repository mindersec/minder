package diff

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

func TestGoParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		description          string
		content              string
		expectedCount        int
		expectedDependencies []*pb.Dependency
	}{
		{
			description: "Single addition",
			content: `
+cloud.google.com/go/compute v1.23.0 h1:tP41Zoavr8ptEqaW6j+LQOnyBBhO7OkOMAGrgLopTwY=
+cloud.google.com/go/compute v1.23.0/go.mod h1:4tCnrn48xsqlwSAiLf1HXMQk8CONslYbdiEZc9FEIbM=
cloud.google.com/go/compute/metadata v0.2.3 h1:mg4jlk7mCAj6xXp9UJ4fjI9VUI5rubuGBW5aJ7UnBMY=
cloud.google.com/go/compute/metadata v0.2.3/go.mod h1:VAV5nSsACxMJvgaAuX6Pk2AawlZn8kiOGuCv6gTkwuA=`,
			expectedCount: 1,
			expectedDependencies: []*pb.Dependency{
				{
					Ecosystem: pb.DepEcosystem_DEP_ECOSYSTEM_GO,
					Name:      "cloud.google.com/go/compute",
					Version:   "v1.23.0",
				},
			},
		},
		{
			description: "Single removal",
			content: `
-cloud.google.com/go/compute v1.23.0 h1:tP41Zoavr8ptEqaW6j+LQOnyBBhO7OkOMAGrgLopTwY=
-cloud.google.com/go/compute v1.23.0/go.mod h1:4tCnrn48xsqlwSAiLf1HXMQk8CONslYbdiEZc9FEIbM=
cloud.google.com/go/compute/metadata v0.2.3 h1:mg4jlk7mCAj6xXp9UJ4fjI9VUI5rubuGBW5aJ7UnBMY=
cloud.google.com/go/compute/metadata v0.2.3/go.mod h1:VAV5nSsACxMJvgaAuX6Pk2AawlZn8kiOGuCv6gTkwuA=`,
			expectedCount:        0,
			expectedDependencies: nil,
		},
		{
			description: "Mixed additions and removals",
			content: `
-cloud.google.com/go/compute v1.23.0 h1:tP41Zoavr8ptEqaW6j+LQOnyBBhO7OkOMAGrgLopTwY=
-cloud.google.com/go/compute v1.23.0/go.mod h1:4tCnrn48xsqlwSAiLf1HXMQk8CONslYbdiEZc9FEIbM=
+cloud.google.com/go/compute/metadata v0.2.3 h1:mg4jlk7mCAj6xXp9UJ4fjI9VUI5rubuGBW5aJ7UnBMY=
+cloud.google.com/go/compute/metadata v0.2.3/go.mod h1:VAV5nSsACxMJvgaAuX6Pk2AawlZn8kiOGuCv6gTkwuA=
+dario.cat/mergo v1.0.0 h1:AGCNq9Evsj31mOgNPcLyXc+4PNABt905YmuqPYYpBWk=
+dario.cat/mergo v1.0.0/go.mod h1:uNxQE+84aUszobStD9th8a29P2fMDhsBdgRYvZOxGmk=`,
			expectedCount: 2,
			expectedDependencies: []*pb.Dependency{
				{
					Ecosystem: pb.DepEcosystem_DEP_ECOSYSTEM_GO,
					Name:      "cloud.google.com/go/compute/metadata",
					Version:   "v0.2.3",
				},
				{
					Ecosystem: pb.DepEcosystem_DEP_ECOSYSTEM_GO,
					Name:      "dario.cat/mergo",
					Version:   "v1.0.0",
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.description, func(t *testing.T) {
			t.Parallel()
			got, err := goParse(tt.content)
			if err != nil {
				t.Fatalf("goParse() returned error: %v", err)
			}

			assert.Equal(t, tt.expectedCount, len(got), "mismatched dependency count")

			for i, expectedDep := range tt.expectedDependencies {
				if !proto.Equal(expectedDep, got[i]) {
					t.Errorf("mismatch at index %d: expected %v, got %v", i, expectedDep, got[i])
				}
			}
		})
	}
}
