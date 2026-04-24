// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package profile

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit an existing profile",
	Long:  `The profile edit subcommand lets you fetch an existing profile, edit it in your $EDITOR, and apply the updates.`,
	RunE:  cli.GRPCClientWrapRunE(editCommand),
}

func editCommand(ctx context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
	client := minderv1.NewProfileServiceClient(conn)
	project := viper.GetString("project")
	id := viper.GetString("id")
	name := viper.GetString("name")

	if id == "" && name == "" {
		return cli.MessageAndError("Error editing profile", fmt.Errorf("id or name required"))
	}
	cmd.SilenceUsage = true

	prof, err := getProfile(ctx, client, project, id, name)
	if err != nil {
		return err
	}

	// hardcoded type and version since ParseResource requires it
	if prof.Type == "" {
		prof.Type = "profile"
	}
	if prof.Version == "" {
		prof.Version = "v1"
	}

	yamlString, err := util.GetYamlFromProto(prof)
	if err != nil {
		return cli.MessageAndError("Error marshaling profile to YAML", err)
	}

	tmpFile, err := os.CreateTemp("", "tmp-minder-profile-*.yaml")
	if err != nil {
		return cli.MessageAndError("Error creating temporary file", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(yamlString); err != nil {
		return cli.MessageAndError("Error writing to temporary file", err)
	}

	// we must close the file descriptor before handing it to the editor
	// many terminal editor perform atomic saves which changes the file's inode
	// keeping the old FD open would read the orphaned file.
	if err := tmpFile.Close(); err != nil {
		return cli.MessageAndError("Error closing temporary file", err)
	}

	if err := handleEditor(tmpFile.Name()); err != nil {
		return err
	}

	updatedBytes, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return cli.MessageAndError("Error reading updated temporary file", err)
	}

	if string(updatedBytes) == yamlString {
		cmd.Println("No changes made to the profile. Aborting update.")
		return nil
	}

	return updateProfile(ctx, client, prof, updatedBytes, cmd)
}

func getProfile(ctx context.Context, client minderv1.ProfileServiceClient, project, id, name string) (*minderv1.Profile, error) {
	if id != "" {
		p, err := client.GetProfileById(ctx, &minderv1.GetProfileByIdRequest{
			Context: &minderv1.Context{Project: &project},
			Id:      id,
		})
		if err != nil {
			return nil, cli.MessageAndError("Error getting profile by ID", err)
		}
		return p.GetProfile(), nil
	}

	p, err := client.GetProfileByName(ctx, &minderv1.GetProfileByNameRequest{
		Context: &minderv1.Context{Project: &project},
		Name:    name,
	})
	if err != nil {
		return nil, cli.MessageAndError("Error getting profile by name", err)
	}
	return p.GetProfile(), nil
}

func handleEditor(fileName string) error {
	editorCmd := cmp.Or(os.Getenv("VISUAL"), os.Getenv("EDITOR"))

	if editorCmd == "" {
		commonEditors := []string{"nano", "vim", "nvim", "vi", "emacs"}
		for _, e := range commonEditors {
			if _, err := exec.LookPath(e); err == nil {
				editorCmd = e
				break
			}
		}
	}

	if editorCmd == "" {
		msg := "no editor found in $PATH. Please set $EDITOR to your preferred editor"
		return cli.MessageAndError(msg, fmt.Errorf("no editor found"))
	}

	// #nosec G204
	// #nosec G702
	execCmd := exec.Command(editorCmd, fileName)
	execCmd.Stdin, execCmd.Stdout, execCmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	if err := execCmd.Run(); err != nil {
		return cli.MessageAndError(fmt.Sprintf("Editor execution failed (%s)", editorCmd), err)
	}
	return nil
}

func updateProfile(
	_ context.Context,
	client minderv1.ProfileServiceClient,
	oldProf *minderv1.Profile,
	updatedBytes []byte,
	cmd *cobra.Command,
) error {
	var updatedProfile minderv1.Profile
	if err := minderv1.ParseResource(bytes.NewReader(updatedBytes), &updatedProfile); err != nil {
		return cli.MessageAndError("Error parsing updated profile YAML", err)
	}

	updatedProfile.Id = proto.String(oldProf.GetId())
	updatedProfile.Context = oldProf.GetContext()
	updatedProfile.Type = oldProf.GetType()
	if updatedProfile.Type == "" {
		updatedProfile.Type = "profile"
	}
	updatedProfile.Version = oldProf.GetVersion()
	if updatedProfile.Version == "" {
		updatedProfile.Version = "v1"
	}

	updateCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.UpdateProfile(updateCtx, &minderv1.UpdateProfileRequest{
		Profile: &updatedProfile,
	})
	if err != nil {
		return cli.MessageAndError("Error updating profile", err)
	}

	cmd.Println("Successfully updated profile named:", resp.GetProfile().GetName())
	return nil
}

func init() {
	ProfileCmd.AddCommand(editCmd)
	editCmd.Flags().StringP("id", "i", "", "ID of the profile to edit")
	editCmd.Flags().StringP("name", "n", "", "Name of the profile to edit")
	editCmd.MarkFlagsMutuallyExclusive("id", "name")
}
