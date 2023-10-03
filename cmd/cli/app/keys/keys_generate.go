//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// NOTE: This file is for stubbing out client code for proof of concept
// purposes. It will / should be removed in the future.
// Until then, it is not covered by unit tests and should not be used
// It does make a good example of how to use the generated client code
// for others to use as a reference.

package keys

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

var genKeys_listCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate keys within a mediator control plane",
	Long: `The medic keys generate  subcommand lets you create keys within a
mediator control plane for an specific project.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {

		project_id := viper.GetString("project-id")
		out := util.GetConfigValue("output", "output", cmd, "").(string)
		pass := util.GetConfigValue("passphrase", "passphrase", cmd, "").(string)
		var passphrase []byte

		if pass == "" {
			var err error
			passphrase, err = util.GetPassFromTerm(true)
			if err != nil {
				util.ExitNicelyOnError(err, "error getting password")
			}
			fmt.Println()
		} else {
			passphrase = []byte(pass)
		}

		conn, err := util.GrpcForCommand(cmd)
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := pb.NewKeyServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		keyResp, err := client.CreateKeyPair(ctx, &pb.CreateKeyPairRequest{
			Passphrase: base64.RawStdEncoding.EncodeToString(passphrase),
			ProjectId:  project_id,
		})
		if err != nil {
			util.ExitNicelyOnError(err, "Error calling create keys")
		}

		decodedPublicKey, err := base64.RawStdEncoding.DecodeString(keyResp.PublicKey)
		if err != nil {
			util.ExitNicelyOnError(err, "Error decoding public key:")
		}

		if out != "" {
			err = util.WriteToFile(out, decodedPublicKey, 0644)
			if err != nil {
				util.ExitNicelyOnError(err, "Error writing public key to file")
			}
		}

		// write to tablewriter for output

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Status", "Key Indentifier"})
		table.Append([]string{"Success", keyResp.KeyIdentifier})
		table.Render()

		return nil
	},
}

func init() {
	KeysCmd.AddCommand(genKeys_listCmd)
	genKeys_listCmd.Flags().StringP("project-id", "g", "", "project id to list roles for")
	genKeys_listCmd.Flags().StringP("output", "o", "", "Output public key to file")
	genKeys_listCmd.Flags().StringP("passphrase", "p", "", "Passphrase to use for key generation")
}
