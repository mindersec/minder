// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package diff provides the diff rule data ingest engine
package diff

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
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

func TestPyPiParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		description          string
		content              string
		expectedCount        int
		expectedDependencies []*pb.Dependency
	}{
		{
			description: "Single addition, exact version",
			content: `
 Flask
+requests==2.19.0`,
			expectedCount: 1,
			expectedDependencies: []*pb.Dependency{
				{
					Ecosystem: pb.DepEcosystem_DEP_ECOSYSTEM_PYPI,
					Name:      "requests",
					Version:   "2.19.0",
				},
			},
		},
		{
			description: "Single addition, exact version, comment",
			content: `
 Flask
+# this version has a CVE
+requests==2.19.0`,
			expectedCount: 1,
			expectedDependencies: []*pb.Dependency{
				{
					Ecosystem: pb.DepEcosystem_DEP_ECOSYSTEM_PYPI,
					Name:      "requests",
					Version:   "2.19.0",
				},
			},
		},
		{
			description: "Single addition, exact version, whitespace",
			content: `
 Flask
+ 
+requests==2.19.0`,
			expectedCount: 1,
			expectedDependencies: []*pb.Dependency{
				{
					Ecosystem: pb.DepEcosystem_DEP_ECOSYSTEM_PYPI,
					Name:      "requests",
					Version:   "2.19.0",
				},
			},
		},
		{
			description: "Single addition, greater or equal version",
			content: `
 Flask
+requests>=2.19.0`,
			expectedCount: 1,
			expectedDependencies: []*pb.Dependency{
				{
					Ecosystem: pb.DepEcosystem_DEP_ECOSYSTEM_PYPI,
					Name:      "requests",
					Version:   "2.19.0",
				},
			},
		},
		{
			description: "Single addition, greater or equal version",
			content: `
 Flask
+requests>=2.19.0`,
			expectedCount: 1,
			expectedDependencies: []*pb.Dependency{
				{
					Ecosystem: pb.DepEcosystem_DEP_ECOSYSTEM_PYPI,
					Name:      "requests",
					Version:   "2.19.0",
				},
			},
		},
		{
			description: "Single addition, exact version, comment",
			content: `
 Flask
+requests==2.19.0 # this version has a CVE`,
			expectedCount: 1,
			expectedDependencies: []*pb.Dependency{
				{
					Ecosystem: pb.DepEcosystem_DEP_ECOSYSTEM_PYPI,
					Name:      "requests",
					Version:   "2.19.0",
				},
			},
		},
		{
			description: "Single addition, wildcard version",
			content: `
 Flask
+requests==2.*`,
			expectedCount: 1,
			expectedDependencies: []*pb.Dependency{
				{
					Ecosystem: pb.DepEcosystem_DEP_ECOSYSTEM_PYPI,
					Name:      "requests",
					Version:   "2",
				},
			},
		},
		{
			description: "Single addition, no version",
			content: `
 Flask
+requests`,
			expectedCount: 1,
			expectedDependencies: []*pb.Dependency{
				{
					Ecosystem: pb.DepEcosystem_DEP_ECOSYSTEM_PYPI,
					Name:      "requests",
					Version:   "",
				},
			},
		},
		{
			description: "Single addition, lower or equal version",
			content: `
 Flask
+requests<=2.19.0`,
			expectedCount: 1,
			expectedDependencies: []*pb.Dependency{
				{
					Ecosystem: pb.DepEcosystem_DEP_ECOSYSTEM_PYPI,
					Name:      "requests",
					Version:   "2.19.0",
				},
			},
		},
		{
			description: "Single addition, version range",
			content: `
 Flask
+requests<3,>=2.0`,
			expectedCount: 1,
			expectedDependencies: []*pb.Dependency{
				{
					Ecosystem: pb.DepEcosystem_DEP_ECOSYSTEM_PYPI,
					Name:      "requests",
					Version:   "2.0",
				},
			},
		},
		{
			description: "Single addition, version range reversed",
			content: `
 Flask
+requests>=2.0,<3`,
			expectedCount: 1,
			expectedDependencies: []*pb.Dependency{
				{
					Ecosystem: pb.DepEcosystem_DEP_ECOSYSTEM_PYPI,
					Name:      "requests",
					Version:   "2.0",
				},
			},
		},
		{
			description: "Multiple additions",
			content: `
 Flask
+requests>=2.0,<3
+pandas<0.25.0,>=0.24.0
+numpy==1.16.0`,
			expectedCount: 3,
			expectedDependencies: []*pb.Dependency{
				{
					Ecosystem: pb.DepEcosystem_DEP_ECOSYSTEM_PYPI,
					Name:      "requests",
					Version:   "2.0",
				},
				{
					Ecosystem: pb.DepEcosystem_DEP_ECOSYSTEM_PYPI,
					Name:      "pandas",
					Version:   "0.24.0",
				},
				{
					Ecosystem: pb.DepEcosystem_DEP_ECOSYSTEM_PYPI,
					Name:      "numpy",
					Version:   "1.16.0",
				},
			},
		},
		{
			description: "No additions",
			content: `
 Flask
# just a comment
`,
			expectedCount:        0,
			expectedDependencies: []*pb.Dependency{},
		},
		{
			description: "Single addition, uppercase",
			content: `
 Flask
+ Django==3.2.21`,
			expectedCount: 1,
			expectedDependencies: []*pb.Dependency{
				{
					Ecosystem: pb.DepEcosystem_DEP_ECOSYSTEM_PYPI,
					Name:      "django",
					Version:   "3.2.21",
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.description, func(t *testing.T) {
			t.Parallel()
			got, err := requirementsParse(tt.content)
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

func TestNpmParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		description          string
		content              string
		expectedCount        int
		expectedDependencies []*pb.Dependency
	}{
		{
			description: "New dependency addition",
			content: `
       "version": "1.0.0",
       "license": "ISC",
       "dependencies": {
+        "chalk": "^5.3.0",
         "lodash": "^3.7.0"
       }
     },
+    "node_modules/chalk": {
+      "version": "5.3.0",
+      "resolved": "https://registry.npmjs.org/chalk/-/chalk-5.3.0.tgz",
+      "integrity": "sha512-dLitG79d+GV1Nb/VYcCDFivJeK1hiukt9QjRNVOsUtTy1rR1YJsmpGGTZ3qJos+uw7WmWF4wUwBd9jxjocFC2w==",
+      "engines": {
+        "node": "^12.17.0 || ^14.13 || >=16.0.0"
+      },
+      "funding": {
+        "url": "https://github.com/chalk/chalk?sponsor=1"
+      }
+    },
     "node_modules/lodash": {
       "version": "3.10.1",
       "resolved": "https://registry.npmjs.org/lodash/-/lodash-3.10.1.tgz",
`,
			expectedCount: 1,
			expectedDependencies: []*pb.Dependency{
				{
					Ecosystem: pb.DepEcosystem_DEP_ECOSYSTEM_NPM,
					Name:      "chalk",
					Version:   "5.3.0",
				},
			},
		},
		{
			description: "Exising dependency version update",
			content: `
       }
     },
     "node_modules/lodash": {
-      "version": "4.17.16",
-      "resolved": "https://registry.npmjs.org/lodash/-/lodash-4.17.16.tgz",
-      "integrity": "sha512-mzxOTaU4AsJhnIujhngm+OnA6JX4fTI8D5H26wwGd+BJ57bW70oyRwTqo6EFJm1jTZ7hCo7yVzH1vB8TMFd2ww=="
+      "version": "4.17.21",
+      "resolved": "https://registry.npmjs.org/lodash/-/lodash-4.17.21.tgz",
+      "integrity": "sha512-v2kDEe57lecTulaDIuNTPy3Ry4gLGJ6Z1O3vE1krgXZNrsQ+LFTGHVxVjcXPs17LhbZVGedAJv8XZ1tvj5FvSg=="
     }
   }
 }
`,
			expectedCount: 1,
			expectedDependencies: []*pb.Dependency{
				{
					Ecosystem: pb.DepEcosystem_DEP_ECOSYSTEM_NPM,
					Name:      "lodash",
					Version:   "4.17.21",
				},
			},
		},
		{
			description: "Entirely new package-lock.json",
			content: `
+{
+  "name": "public-repo",
+  "version": "1.0.0",
+  "lockfileVersion": 3,
+  "requires": true,
+  "packages": {
+    "": {
+      "name": "public-repo",
+      "version": "1.0.0",
+      "license": "ISC",
+      "dependencies": {
+        "lodash": "^4.17.21"
+      }
+    },
+    "node_modules/lodash": {
+      "version": "4.17.21",
+      "resolved": "https://registry.npmjs.org/lodash/-/lodash-4.17.21.tgz",
+      "integrity": "sha512-v2kDEe57lecTulaDIuNTPy3Ry4gLGJ6Z1O3vE1krgXZNrsQ+LFTGHVxVjcXPs17LhbZVGedAJv8XZ1tvj5FvSg=="
+    }
+  }
+}
`,
			expectedCount: 1,
			expectedDependencies: []*pb.Dependency{
				{
					Ecosystem: pb.DepEcosystem_DEP_ECOSYSTEM_NPM,
					Name:      "lodash",
					Version:   "4.17.21",
				},
			},
		},
		{
			description: "Multiple new dependencies",
			content: `
       "version": "1.0.0",
       "license": "ISC",
       "dependencies": {
+        "@types/node": "^20.9.0",
         "lodash": "^4.17.16"
       }
     },
+    "node_modules/@types/node": {
+      "version": "20.9.0",
+      "resolved": "https://registry.npmjs.org/@types/node/-/node-20.9.0.tgz",
+      "integrity": "sha512-nekiGu2NDb1BcVofVcEKMIwzlx4NjHlcjhoxxKBNLtz15Y1z7MYf549DFvkHSId02Ax6kGwWntIBPC3l/JZcmw==",
+      "dependencies": {
+        "undici-types": "~5.26.4"
+      }
+    },
     "node_modules/lodash": {
       "version": "4.17.16",
       "resolved": "https://registry.npmjs.org/lodash/-/lodash-4.17.16.tgz",
       "integrity": "sha512-mzxOTaU4AsJhnIujhngm+OnA6JX4fTI8D5H26wwGd+BJ57bW70oyRwTqo6EFJm1jTZ7hCo7yVzH1vB8TMFd2ww=="
+    },
+    "node_modules/undici-types": {
+      "version": "5.26.5",
+      "resolved": "https://registry.npmjs.org/undici-types/-/undici-types-5.26.5.tgz",
+      "integrity": "sha512-JlCMO+ehdEIKqlFxk6IfVoAUVmgz7cU7zD/h9XZ0qzeosSHmUJVOzSQvvYSYWXkFXC+IfLKSIffhv0sVZup6pA=="
     }
   }
 }
`,
			expectedCount: 2,
			expectedDependencies: []*pb.Dependency{
				{
					Ecosystem: pb.DepEcosystem_DEP_ECOSYSTEM_NPM,
					Name:      "@types/node",
					Version:   "20.9.0",
				},
				{
					Ecosystem: pb.DepEcosystem_DEP_ECOSYSTEM_NPM,
					Name:      "undici-types",
					Version:   "5.26.5",
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.description, func(t *testing.T) {
			t.Parallel()
			got, err := npmParse(tt.content)
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
