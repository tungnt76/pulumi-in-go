package main

import (
	"context"
	"fmt"
	"os"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
	cloudflareprefixlists "github.com/tungnt76/pulumi-in-go/aws/cloudflare-prefix-lists"
	securitygroup "github.com/tungnt76/pulumi-in-go/aws/security-group"
	"github.com/tungnt76/pulumi-in-go/aws/vpc"
)

func main() {
	// to destroy our program, we can run `go run main.go destroy`
	destroy := false
	argsWithoutProg := os.Args[1:]
	if len(argsWithoutProg) > 0 {
		if argsWithoutProg[0] == "destroy" {
			destroy = true
		}
	}

	ctx := context.Background()
	stackName := "dev"
	workDir := "./" // optional and defaults to process.cwd if not specified

	s, err := auto.UpsertStackLocalSource(ctx, stackName, workDir, auto.Program(createVpcWithSG))
	if err != nil {
		fmt.Printf("Failed to create or select stack: %v\n", err)
		os.Exit(1)
	}

	w := s.Workspace()
	w.InstallPlugin(ctx, "aws", "v6.56.0")
	w.InstallPlugin(ctx, "cloudflare", "v5.40.1")

	_, err = s.Refresh(ctx)
	if err != nil {
		fmt.Printf("Failed to refresh stack: %v\n", err)
		os.Exit(1)
	}

	if destroy {
		_, err = s.Destroy(ctx)
		if err != nil {
			fmt.Printf("Failed to destroy stack: %v\n", err)
		}
		os.Exit(0)
	}

	// wire up our update to stream progress to stdout
	stdoutStreamer := optup.ProgressStreams(os.Stdout)

	_, err = s.Up(ctx, stdoutStreamer)
	if err != nil {
		fmt.Printf("Failed to update stack: %v\n\n", err)
		os.Exit(1)
	}
}

func createVpcWithSG(ctx *pulumi.Context) error {
	l, err := cloudflareprefixlists.CreateCloudflarePrefixLists(
		ctx,
		&cloudflareprefixlists.CloudflarePrefixListsArgs{
			Environment: "dev",
		})
	if err != nil {
		fmt.Printf("Failed to create Cloudflare Prefix Lists: %v\n", err)
		os.Exit(1)
	}

	vpcConfig := config.New(ctx, "vpc")
	azs := []string{}
	vpcConfig.GetObject("azs", &azs)

	vpcArgs := &vpc.VpcArgs{
		Name: vpcConfig.Get("name"),
		Cidr: vpcConfig.Get("cidr"),
		Tags: map[string]string{
			"Environment": "dev",
		},
		PrivateSubnetTags: map[string]string{
			"Environment": "dev",
		},
		PublicSubnetTags: map[string]string{
			"Environment": "dev",
		},
	}

	vpcOutput, err := vpc.CreateVpc(ctx, vpcArgs)
	if err != nil {
		fmt.Printf("Failed to create VPC: %v\n", err)
		os.Exit(1)
	}

	pulumi.All(vpcOutput.VpcId, l.Ipv4ManagedId, l.Ipv6ManagedId).ApplyT(func(args []interface{}) error {
		vpcId := args[0].(pulumi.ID)
		ipv4ManagedId := args[1].(pulumi.ID)
		ipv6ManagedId := args[2].(pulumi.ID)

		_, err := securitygroup.CreateSecurityGroup(
			ctx,
			&securitygroup.SecurityGroupArgs{
				Name:  "dev",
				VpcId: string(vpcId),
				Tags:  map[string]string{"Environment": "dev"},
				IngressRules: []*securitygroup.IngressRule{
					{
						FromPort:   80,
						ToPort:     80,
						Protocol:   "tcp",
						CidrBlocks: []string{vpcArgs.Cidr},
					},
				},
				IngressPrefixListIds: []string{string(ipv4ManagedId), string(ipv6ManagedId)},
			},
		)
		if err != nil {
			fmt.Printf("Failed to create Security Group: %v\n", err)
			os.Exit(1)
		}

		return nil
	})
	return nil
}
