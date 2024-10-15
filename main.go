package main

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
	"github.com/tungnt76/pulumi-in-go/aws/vpc"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		vpcConfig := config.New(ctx, "vpc")

		var (
			name     string
			baseCidr string
			azs      []string
			tags     map[string]string
		)
		name = vpcConfig.Require("name")
		baseCidr = vpcConfig.Require("base_cidr")
		vpcConfig.RequireObject("azs", &azs)
		vpcConfig.GetObject("tags", &tags)

		fmt.Println(name, baseCidr, azs, tags)

		vpc, privateSubnets, publicSubnets, err := vpc.Create(ctx, name, baseCidr, azs, map[string]string{})
		if err != nil {
			return err
		}

		fmt.Printf("VPC: %v\n", vpc)
		fmt.Printf("Private Subnets: %v\n", privateSubnets)
		fmt.Printf("Public Subnets: %v\n", publicSubnets)

		return nil
	})
}
