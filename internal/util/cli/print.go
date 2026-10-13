// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"

	"github.com/mindersec/minder/internal/util"
)

// PrintProto formats a protobuf message as JSON or YAML and writes it to cmd.OutOrStdout().
func PrintProto(cmd *cobra.Command, format string, msg proto.Message) error {
	switch strings.ToLower(format) {
	case "json":
		out, err := util.GetJsonFromProto(msg)
		if err != nil {
			return MessageAndError("Error getting json from proto", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), out)
		return nil
	case "yaml":
		out, err := util.GetYamlFromProto(msg)
		if err != nil {
			return MessageAndError("Error getting yaml from proto", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), out)
		return nil
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// PrintMultiProtoYAML formats multiple protobuf messages as YAML documents separated by "---"
// and writes them to cmd.OutOrStdout().
func PrintMultiProtoYAML[T proto.Message](cmd *cobra.Command, msgs []T) error {
	for _, msg := range msgs {
		out, err := util.GetYamlFromProto(msg)
		if err != nil {
			return MessageAndError("Error getting yaml from proto", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "---")
		fmt.Fprintln(cmd.OutOrStdout(), out)
	}
	return nil
}
