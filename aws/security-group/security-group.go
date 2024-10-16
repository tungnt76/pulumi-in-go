package securitygroup

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type SecurityGroupArgs struct {
	Name                 string
	VpcId                string
	Tags                 map[string]string
	IngressRules         []*IngressRule
	IngressPrefixListIds []string
	EgressRules          []*EgressRule
	EgressPrefixListIds  []string
}

type IngressRule struct {
	FromPort   int
	ToPort     int
	Protocol   string
	CidrBlocks []string
}

type EgressRule struct {
	FromPort   int
	ToPort     int
	Protocol   string
	CidrBlocks []string
}

type SecurityGroup struct {
	SecurityGroupID  string
	SecurityGroupArn string
}

func CreateSecurityGroup(ctx *pulumi.Context, args *SecurityGroupArgs) (*SecurityGroup, error) {
	if args == nil {
		return nil, fmt.Errorf("args cannot be nil")
	}

	name := args.Name
	vpcId := args.VpcId
	tags := args.Tags
	ingressRules := args.IngressRules
	egressRules := args.EgressRules

	ingress := ec2.SecurityGroupIngressArray{}
	for _, rule := range ingressRules {
		ingress = append(ingress, ec2.SecurityGroupIngressArgs{
			FromPort:      pulumi.Int(rule.FromPort),
			ToPort:        pulumi.Int(rule.ToPort),
			Protocol:      pulumi.String(rule.Protocol),
			CidrBlocks:    pulumi.ToStringArray(rule.CidrBlocks),
			PrefixListIds: pulumi.ToStringArray(args.IngressPrefixListIds),
		})
	}

	egress := ec2.SecurityGroupEgressArray{}
	for _, rule := range egressRules {
		egress = append(egress, ec2.SecurityGroupEgressArgs{
			FromPort:      pulumi.Int(rule.FromPort),
			ToPort:        pulumi.Int(rule.ToPort),
			Protocol:      pulumi.String(rule.Protocol),
			CidrBlocks:    pulumi.ToStringArray(rule.CidrBlocks),
			PrefixListIds: pulumi.ToStringArray(args.IngressPrefixListIds),
		})
	}

	sg, err := ec2.NewSecurityGroup(ctx, name, &ec2.SecurityGroupArgs{
		Name:  pulumi.StringPtr(name),
		VpcId: pulumi.StringPtr(vpcId),
		Tags: pulumi.ToStringMap(
			merge(map[string]string{
				"Name": name,
			}, tags),
		),
		Ingress: ingress,
		Egress:  egress,
	})
	if err != nil {
		return nil, err
	}

	return &SecurityGroup{
		SecurityGroupID:  sg.ID().ElementType().String(),
		SecurityGroupArn: sg.Arn.ElementType().String(),
	}, nil
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
