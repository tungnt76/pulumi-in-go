package cloudflareprefixlists

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-cloudflare/sdk/v5/go/cloudflare"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type CloudflarePrefixListsArgs struct {
	Environment string
}

type CloudflarePrefixListsOutput struct {
	Ipv4ManagedId pulumi.IDOutput
	Ipv6ManagedId pulumi.IDOutput
}

func CreateCloudflarePrefixLists(ctx *pulumi.Context, args *CloudflarePrefixListsArgs) (*CloudflarePrefixListsOutput, error) {
	if args == nil {
		return nil, fmt.Errorf("args cannot be nil")
	}

	env := args.Environment

	ipRangesResult, err := cloudflare.GetIpRanges(ctx)
	if err != nil {
		return nil, err
	}

	ipv4ManagedId, err := createCloudflareIpv4List(ctx, ipRangesResult.Ipv4CidrBlocks, env)
	if err != nil {
		return nil, err
	}

	ipv6ManagedId, err := createCloudflareIpv6List(ctx, ipRangesResult.Ipv6CidrBlocks, env)
	if err != nil {
		return nil, err
	}

	return &CloudflarePrefixListsOutput{
		Ipv4ManagedId: ipv4ManagedId,
		Ipv6ManagedId: ipv6ManagedId,
	}, nil
}

func createCloudflareIpv4List(ctx *pulumi.Context, ipv4CidrBlocks []string, env string) (pulumi.IDOutput, error) {
	entries := ec2.ManagedPrefixListEntryTypeArray{}
	for i, cidr := range ipv4CidrBlocks {
		entries = append(entries, ec2.ManagedPrefixListEntryTypeArgs{
			Cidr:        pulumi.String(cidr),
			Description: pulumi.String(fmt.Sprintf("entry %d", i+1)),
		})
	}

	resource, err := ec2.NewManagedPrefixList(ctx, "cloudflare_ipv4_list", &ec2.ManagedPrefixListArgs{
		AddressFamily: pulumi.String("IPv4"),
		MaxEntries:    pulumi.Int(20),
		Name:          pulumi.String("Cloudflare IPv4 Prefix List"),
		Entries:       entries,
		Tags: pulumi.ToStringMap(map[string]string{
			"Environment": env,
		}),
	})
	if err != nil {
		return pulumi.IDOutput{}, err
	}

	return resource.ID(), nil
}

func createCloudflareIpv6List(ctx *pulumi.Context, ipv6CidrBlocks []string, env string) (pulumi.IDOutput, error) {
	entries := ec2.ManagedPrefixListEntryTypeArray{}
	for i, cidr := range ipv6CidrBlocks {
		entries = append(entries, ec2.ManagedPrefixListEntryTypeArgs{
			Cidr:        pulumi.String(cidr),
			Description: pulumi.String(fmt.Sprintf("entry %d", i+1)),
		})
	}

	ipv6List, err := ec2.NewManagedPrefixList(ctx, "cloudflare_ipv6_list", &ec2.ManagedPrefixListArgs{
		AddressFamily: pulumi.String("IPv6"),
		MaxEntries:    pulumi.Int(20),
		Name:          pulumi.String("Cloudflare IPv6 Prefix List"),
		Entries:       entries,
		Tags: pulumi.ToStringMap(map[string]string{
			"Environment": env,
		}),
	})
	if err != nil {
		return pulumi.IDOutput{}, err
	}

	return ipv6List.ID(), nil
}
