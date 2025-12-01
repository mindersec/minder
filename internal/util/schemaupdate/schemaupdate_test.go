// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package schemaupdate_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/mindersec/minder/internal/util/schemaupdate"
)

func TestValidateSchemaUpdate(t *testing.T) {
	t.Parallel()

	type args struct {
		oldRuleSchemaDef string
		newRuleSchemaDef string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "empty schemas should not error",
			args: args{
				oldRuleSchemaDef: "{}",
				newRuleSchemaDef: "{}",
			},
		},
		{
			name: "empty new schema should not error",
			args: args{
				oldRuleSchemaDef: `{
					"type": "object",
					"properties": {
						"foo": {
							"type": "string"
						}
					}
				}`,
				newRuleSchemaDef: "{}",
			},
		},
		{
			name: "empty old schema should error if new schema has required fields",
			args: args{
				oldRuleSchemaDef: "{}",
				newRuleSchemaDef: `{
					"type": "object",
					"properties": {
						"foo": {
							"type": "string"
						}
					},
					"required": ["foo"]
				}`,
			},
			wantErr: true,
		},
		{
			name: "empty old schema should not error if new schema has no required fields",
			args: args{
				oldRuleSchemaDef: "{}",
				newRuleSchemaDef: `{
					"type": "object",
					"properties": {
						"foo": {
							"type": "string"
						}
					}
				}`,
			},
		},
		{
			name: "old schema should error if new schema has required fields",
			args: args{
				oldRuleSchemaDef: `{
					"type": "object",
					"properties": {
						"foo": {
							"type": "string"
						}
					}
				}`,
				newRuleSchemaDef: `{
					"type": "object",
					"properties": {
						"foo": {
							"type": "string"
						}
					},
					"required": ["foo"]
				}`,
			},
			wantErr: true,
		},
		{
			name: "removing required fields is allowed",
			args: args{
				oldRuleSchemaDef: `{
					"type": "object",
					"properties": {
						"foo": {
							"type": "string"
						}
					},
					"required": ["foo"]
				}`,
				newRuleSchemaDef: `{
					"type": "object",
					"properties": {
						"foo": {
							"type": "string"
						}
					}
				}`,
			},
		},
		{
			name: "old schema should error if new schema deletes fields",
			args: args{
				oldRuleSchemaDef: `{
					"type": "object",
					"properties": {
						"foo": {
							"type": "string"
						},
						"bar": {
							"type": "string"
						}
					}
				}`,
				newRuleSchemaDef: `{
					"type": "object",
					"properties": {
						"foo": {
							"type": "string"
						}
					}
				}`,
			},
			wantErr: true,
		},
		{
			name: "old schema should error if new schema changes type of field",
			args: args{
				oldRuleSchemaDef: `{
					"type": "object",
					"properties": {
						"foo": {
							"type": "string"
						}
					}
				}`,
				newRuleSchemaDef: `{
					"type": "bool"
				}`,
			},
			wantErr: true,
		},
		{
			name: "update should succeed if new schema is a superset of old schema",
			args: args{
				oldRuleSchemaDef: `{
					"type": "object",
					"properties": {
						"foo": {
							"type": "string"
						}
					}
				}`,
				newRuleSchemaDef: `{
					"type": "object",
					"properties": {
						"foo": {
							"type": "string"
						},
						"bar": {
							"type": "string"
						}
					}
				}`,
			},
		},
		{
			name: "Changing the items type of an array should error",
			args: args{
				oldRuleSchemaDef: `{
					"type": "array",
					"items": {
						"type": "string"
					}
				}`,
				newRuleSchemaDef: `{
					"type": "array",
					"items": {
						"type": "bool"
					}
				}`,
			},
			wantErr: true,
		},
		{
			name: "changing the description is allowed",
			args: args{
				oldRuleSchemaDef: `{
					"type": "object",
					"properties": {
						"foo": {
							"type": "string",
							"description": "foo desc original"
						},
						"bar": {
							"type": "string"
						}
					}
				}`,
				newRuleSchemaDef: `{
					"type": "object",
					"properties": {
						"foo": {
							"type": "string",
							"description": "foo desc modified"
						},
						"bar": {
							"type": "string"
						}
					}
				}`,
			},
			wantErr: false,
		},
		{
			name: "changing a property named description is not allowed",
			args: args{
				oldRuleSchemaDef: `{
					"type": "object",
					"properties": {
						"description": {
							"type": "string"
						},
						"bar": {
							"type": "string"
						}
					}
				}`,
				newRuleSchemaDef: `{
					"type": "object",
					"properties": {
						"description": {
							"type": "int"
						},
						"bar": {
							"type": "string"
						}
					}
				}`,
			},
			wantErr: true,
		},
		{
			name: "changing the default is allowed",
			args: args{
				oldRuleSchemaDef: `{
					"type": "object",
					"properties": {
						"foo": {
							"type": "string",
							"default": "f-o-o"
						},
						"bar": {
							"type": "string"
						}
					}
				}`,
				newRuleSchemaDef: `{
					"type": "object",
					"properties": {
						"foo": {
							"type": "string",
							"default": "o-o-f"
						},
						"bar": {
							"type": "string"
						}
					}
				}`,
			},
			wantErr: false,
		},
		{
			name: "changing a property named default is not allowed",
			args: args{
				oldRuleSchemaDef: `{
					"type": "object",
					"properties": {
						"default": {
							"type": "string"
						},
						"bar": {
							"type": "string"
						}
					}
				}`,
				newRuleSchemaDef: `{
					"type": "object",
					"properties": {
						"default": {
							"type": "int"
						},
						"bar": {
							"type": "string"
						}
					}
				}`,
			},
			wantErr: true,
		},
		{
			name: "Removing the properties map is not allowed",
			args: args{
				oldRuleSchemaDef: `{
					"type": "object",
					"properties": {
						"foo": {
							"type": "string"
						}
					}
				}`,
				newRuleSchemaDef: `{
					"type": "object"
				}`,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			oldRuleSchema := &structpb.Struct{}
			newRuleSchema := &structpb.Struct{}
			require.NoError(t, protojson.Unmarshal([]byte(tt.args.oldRuleSchemaDef), oldRuleSchema),
				"expected no error parsing old rule schema")
			require.NoError(t, protojson.Unmarshal([]byte(tt.args.newRuleSchemaDef), newRuleSchema),
				"expected no error parsing new rule schema")

			err := schemaupdate.ValidateSchemaUpdate(oldRuleSchema, newRuleSchema)
			if tt.wantErr {
				require.Error(t, err, "expected error validating schema update")
			} else {
				require.NoError(t, err, "expected no error validating schema update")
			}
		})
	}
}
