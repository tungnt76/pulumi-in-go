package targetgroup

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/lb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type TargetGroupArgs struct {
	Name                string
	Port                int
	Protocol            string
	TargetType          string
	VpcId               string
	ProtocolVersion     string
	HealthCheckPath     string
	HealthCheckProtocol string
}

type TargetGroupOutput struct {
	TargetGroupArn pulumi.StringOutput
	TargetGroupId  pulumi.IDOutput
}

func CreateTargetGroup(ctx *pulumi.Context, args *TargetGroupArgs) (*TargetGroupOutput, error) {
	tg, err := lb.NewTargetGroup(ctx, args.Name, &lb.TargetGroupArgs{
		Name:            pulumi.String(args.Name),
		Port:            pulumi.IntPtr(args.Port),
		Protocol:        pulumi.String(args.Protocol),
		TargetType:      pulumi.String(args.TargetType),
		VpcId:           pulumi.String(args.VpcId),
		ProtocolVersion: pulumi.String(args.ProtocolVersion),
		HealthCheck: &lb.TargetGroupHealthCheckArgs{
			Path:     pulumi.String(args.HealthCheckPath),
			Protocol: pulumi.String(args.HealthCheckProtocol),
		},
		ProxyProtocolV2: pulumi.Bool(false),
	})
	if err != nil {
		return nil, err
	}

	return &TargetGroupOutput{
		TargetGroupArn: tg.Arn,
		TargetGroupId:  tg.ID(),
	}, nil
}
