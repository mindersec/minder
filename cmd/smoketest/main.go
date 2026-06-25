package main

import (
	"fmt"
	"strings"

	"github.com/mindersec/minder/internal/util"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func section(title string) {
	fmt.Printf("\n%s\n%s\n", title, strings.Repeat("─", len(title)))
}

func strPtr(s string) *string { return &s }

func main() {
	project := "my-project"

	section("./bin/minder profile get --output yaml")
	p := &pb.Profile{
		Version:   "v1",
		Type:      "profile",
		Name:      "secret-scanning",
		Context:   &pb.Context{Project: &project},
		Alert:     strPtr("on"),
		Remediate: strPtr("off"),
	}
	out, _ := util.GetOrderedYamlFromProto(p)
	fmt.Print(out)

	section("./bin/minder ruletype get --output yaml")
	rt := &pb.RuleType{
		Version:     "v1",
		Type:        "rule-type",
		Name:        "secret_scanning",
		Context:     &pb.Context{Project: &project},
		Description: "Verifies secret scanning is enabled.",
		Guidance:    "Enable secret scanning in repo settings.",
	}
	out, _ = util.GetOrderedYamlFromProto(rt)
	fmt.Print(out)

	section("./bin/minder datasource get --output yaml")
	ds := &pb.DataSource{
		Version: "v1",
		Type:    "data-source",
		Name:    "github-api",
		Context: &pb.ContextV2{ProjectId: project},
	}
	out, _ = util.GetOrderedYamlFromProto(ds)
	fmt.Print(out)
}
