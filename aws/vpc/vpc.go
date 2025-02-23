package vpc

import (
	"fmt"
	"log"
	"net"

	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	_newBits = 8

	// single nat gateway
	_enableNatGateway = true
)

type VpcArgs struct {
	Name              string
	Cidr              string
	Tags              map[string]string
	PrivateSubnetTags map[string]string
	PublicSubnetTags  map[string]string
}

type VpcOutput struct {
	VpcId  pulumi.IDOutput
	VpcArn pulumi.StringOutput
}

func CreateVpc(ctx *pulumi.Context, args *VpcArgs) (*VpcOutput, error) {
	if args == nil {
		return nil, fmt.Errorf("args cannot be nil")
	}

	name := args.Name
	vcpCidr := args.Cidr
	tags := args.Tags
	privateSubnetTags := args.PrivateSubnetTags
	publicSubnetTags := args.PublicSubnetTags

	azs, err := aws.GetAvailabilityZones(ctx, &aws.GetAvailabilityZonesArgs{
		State: pulumi.StringRef("available"),
	})
	if err != nil {
		return nil, err
	}

	vpc, err := ec2.NewVpc(
		ctx,
		fmt.Sprintf("%s-vpc", name),
		&ec2.VpcArgs{
			CidrBlock:          pulumi.String(vcpCidr),
			EnableDnsHostnames: pulumi.Bool(true),
			EnableDnsSupport:   pulumi.Bool(true),
			Tags: pulumi.ToStringMap(
				merge(map[string]string{
					"Name": fmt.Sprintf("%s-vpc", name),
				}, tags),
			),
		})
	if err != nil {
		log.Println("new vpc error", err)
		return nil, err
	}

	igw, err := ec2.NewInternetGateway(
		ctx,
		fmt.Sprintf("%s-igw", name),
		&ec2.InternetGatewayArgs{
			VpcId: vpc.ID(),
			Tags: pulumi.ToStringMap(
				merge(map[string]string{
					"Name": fmt.Sprintf("%s-igw", name),
				}, tags),
			),
		})
	if err != nil {
		log.Println("new internet gateway error", err)
		return nil, err
	}

	privateCidrSubnets, publicCidrSubnets, err := cidrSubnet(vcpCidr, azs.Names)
	fmt.Println(privateCidrSubnets, publicCidrSubnets)
	if err != nil {
		log.Println("cidr subnet error", err)
		return nil, err
	}

	privateSubnets := []*ec2.Subnet{}
	for index, cidrBlock := range privateCidrSubnets {
		subnet, err := ec2.NewSubnet(
			ctx,
			fmt.Sprintf("%s-private-%d", name, index+1),
			&ec2.SubnetArgs{
				VpcId:            vpc.ID(),
				CidrBlock:        pulumi.String(cidrBlock),
				AvailabilityZone: pulumi.String(azs.Names[index]),
				Tags: pulumi.ToStringMap(
					merge(map[string]string{
						"Name": fmt.Sprintf("%s-private-%d", name, index+1),
					}, privateSubnetTags),
				),
			})
		if err != nil {
			log.Println("new private subnet error", err)
			return nil, err
		}
		privateSubnets = append(privateSubnets, subnet)
	}

	publicSubnets := []*ec2.Subnet{}
	for index, cidrBlock := range publicCidrSubnets {
		subnet, err := ec2.NewSubnet(
			ctx,
			fmt.Sprintf("%s-public-%d", name, index+1),
			&ec2.SubnetArgs{
				VpcId:            vpc.ID(),
				CidrBlock:        pulumi.String(cidrBlock),
				AvailabilityZone: pulumi.String(azs.Names[index]),
				Tags: pulumi.ToStringMap(
					merge(map[string]string{
						"Name": fmt.Sprintf("%s-public-%d", name, index+1),
					}, publicSubnetTags),
				),
			})
		if err != nil {
			log.Println("new public subnet error", err)
			return nil, err
		}
		publicSubnets = append(publicSubnets, subnet)

		// route public subnet to gateway
		routeTable, err := ec2.NewRouteTable(
			ctx,
			fmt.Sprintf("%s-public-rt-%d", name, index+1),
			&ec2.RouteTableArgs{
				VpcId: vpc.ID(),
				Routes: ec2.RouteTableRouteArray{
					&ec2.RouteTableRouteArgs{
						CidrBlock: pulumi.String("0.0.0.0/0"),
						GatewayId: igw.ID(),
					},
				},
				Tags: pulumi.ToStringMap(
					merge(map[string]string{
						"Name": fmt.Sprintf("%s-public-rt-%d", name, index+1),
					}, tags),
				),
			},
		)
		if err != nil {
			log.Println("new route table error", err)
			return nil, err
		}

		_, err = ec2.NewRouteTableAssociation(
			ctx,
			fmt.Sprintf("%s-public-rt-asc-%d", name, index+1),
			&ec2.RouteTableAssociationArgs{
				RouteTableId: routeTable.ID(),
				SubnetId:     subnet.ID(),
			},
		)
		if err != nil {
			log.Println("new route table association error", err)
			return nil, err
		}
	}

	if _enableNatGateway {
		// create eip for nat gateway
		eip, err := ec2.NewEip(
			ctx,
			fmt.Sprintf("%s-eip", name),
			&ec2.EipArgs{
				Vpc: pulumi.Bool(true),
				Tags: pulumi.ToStringMap(
					merge(map[string]string{
						"Name": fmt.Sprintf("%s-eip", name),
					}, tags),
				),
			},
		)
		if err != nil {
			log.Println("new eip error", err)
			return nil, err
		}

		natGw, err := ec2.NewNatGateway(
			ctx,
			fmt.Sprintf("%s-ngw", name),
			&ec2.NatGatewayArgs{
				AllocationId: eip.ID(),
				SubnetId:     publicSubnets[0].ID(),
				Tags: pulumi.ToStringMap(
					merge(map[string]string{
						"Name": fmt.Sprintf("%s-ngw", name),
					}, tags),
				),
			},
		)
		if err != nil {
			log.Println("new nat gateway error", err)
			return nil, err
		}

		for index, subnet := range privateSubnets {
			// route private subnet to nat gateway
			routeTable, err := ec2.NewRouteTable(
				ctx,
				fmt.Sprintf("%s-private-rt-%d", name, index+1),
				&ec2.RouteTableArgs{
					VpcId: vpc.ID(),
					Routes: ec2.RouteTableRouteArray{
						&ec2.RouteTableRouteArgs{
							CidrBlock:    pulumi.String("0.0.0.0/0"),
							NatGatewayId: natGw.ID(),
						},
					},
					Tags: pulumi.ToStringMap(
						merge(map[string]string{
							"Name": fmt.Sprintf("%s-private-rt-%d", name, index+1),
						}, tags),
					),
				},
			)
			if err != nil {
				log.Println("new route table error", err)
				return nil, err
			}

			_, err = ec2.NewRouteTableAssociation(
				ctx,
				fmt.Sprintf("%s-ngw-rt-asc-%d", name, index+1),
				&ec2.RouteTableAssociationArgs{
					RouteTableId: routeTable.ID(),
					SubnetId:     subnet.ID(),
				},
			)
			if err != nil {
				log.Println("new route table association error", err)
				return nil, err
			}
		}
	}

	return &VpcOutput{
		VpcId:  vpc.ID(),
		VpcArn: vpc.Arn,
	}, nil
}

func cidrSubnet(vcpCidr string, azs []string) (privateCidrSubnets, publicSCidrSubnets []string, err error) {
	_, base, err := net.ParseCIDR(vcpCidr)
	if err != nil {
		return nil, nil, err
	}

	subnets := []string{}
	numAz := len(azs)
	for i := 1; i <= numAz*2; i++ {
		n, err := cidr.Subnet(base, _newBits, i)
		if err != nil {
			return nil, nil, err
		}

		subnets = append(subnets, n.String())
	}

	privateCidrSubnets = subnets[:numAz]
	publicSCidrSubnets = subnets[numAz:]

	return
}

func merge[M ~map[K]V, K comparable, V any](m1 M, m2 M) M {
	new := map[K]V{}
	for k, v := range m1 {
		new[k] = v
	}
	for k, v := range m2 {
		new[k] = v
	}

	return new
}
