package repo

import (
	"context"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"github.com/stacklok/mediator/pkg/util"
	"os"
	"strings"
)

// repo_registerCmd represents the register command to register a repo with the
// mediator control plane
var repo_registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a repo with the mediator control plane",
	Long:  `Repo register is used to register a repo with the mediator control plane`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {

		grpc_host := util.GetConfigValue("grpc_server.host", "grpc-host", cmd, "").(string)
		grpc_port := util.GetConfigValue("grpc_server.port", "grpc-port", cmd, 0).(int)

		groupID := viper.GetInt32("group-id")
		limit := viper.GetInt32("limit")
		offset := viper.GetInt32("offset")

		conn, err := util.GetGrpcConnection(grpc_host, grpc_port)
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := pb.NewRepositoryServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		listResp, err := client.ListRepositories(ctx, &pb.ListRepositoriesRequest{
			GroupId: groupID,
			Limit:   int32(limit),
			Offset:  int32(offset),
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting repo of repos: %s\n", err)
			os.Exit(1)
		}

		var allSelectedRepos []string

		repoNames := make([]string, len(listResp.Results))
		for i, repo := range listResp.Results {
			repoNames[i] = fmt.Sprintf("%s/%s", repo.Owner, repo.Name)
		}

		var selectedRepos []string
		prompt := &survey.MultiSelect{
			Message:  "Choose a repository:",
			Options:  repoNames,
			PageSize: 20, // PageSize determins how many options are shown at once, restricted by limit flag
		}

		err = survey.AskOne(prompt, &selectedRepos)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting repo selection: %s\n", err)
			os.Exit(1)
		}
		allSelectedRepos = append(allSelectedRepos, selectedRepos...)
		repoProtos := make([]*pb.Repositories, len(allSelectedRepos))

		// Convert the selected repos into a slice of Repositories protobufs
		for i, repo := range allSelectedRepos {
			splitRepo := strings.Split(repo, "/")
			if len(splitRepo) != 2 {
				fmt.Fprintf(os.Stderr, "Unexpected repository name format: %s\n", repo)
				os.Exit(1)
			}
			repoProtos[i] = &pb.Repositories{
				Owner: splitRepo[0],
				Name:  splitRepo[1],
			}
		}

		// Construct the RegisterRepositoryRequest
		request := &pb.RegisterRepositoryRequest{
			Repositories: repoProtos,
			Events:       nil, // Nil results in all events being registered
			GroupId:      groupID,
		}

		fmt.Printf("Registering repositories: %v\n", allSelectedRepos)

		registerResp, err := client.RegisterRepository(context.Background(), request)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error registering repositories: %s\n", err)
			os.Exit(1)
		}

		fmt.Printf("Registered repositories successfully. Response: %v\n", registerResp.Results)

	},
}

func init() {
	RepoCmd.AddCommand(repo_registerCmd)
	var reposFlag string
	repo_registerCmd.Flags().Int32P("group-id", "g", 0, "ID of the group for repo registration")
	repo_registerCmd.Flags().Int32P("limit", "l", 20, "Number of repos to display per page")
	repo_registerCmd.Flags().Int32P("offset", "o", 0, "Offset of the repos to display")
	repo_registerCmd.Flags().StringVar(&reposFlag, "repo", "", "List of key-value pairs")
}
