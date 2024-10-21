package alb

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/lb"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/route53"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/tungnt76/pulumi-in-go/cloudflare/elb/alb/acm"
)

type ALBArgs struct {
	Name          string
	CloudZoneName string
	Environment   string
	Domain        string

	VpcId            string
	Internal         bool
	SecurityGroupIDs []string
	TargetGroupArn   string
	Listener         *Listener
	FixedResponse    *FixedResponse

	Route53HostedZone string
	ExtraDomains      []string
	Proxied           bool
}

type Listener struct {
	Port       int
	Protocol   string
	AlpnPolicy string
}

type FixedResponse struct {
	ContentType string
	MessageBody string
	StatusCode  string
}

type ALBOutput struct{}

func CreateALB(ctx *pulumi.Context, args *ALBArgs) (*ALBOutput, error) {
	if args.Listener == nil {
		args.Listener = &Listener{
			Port:       443,
			Protocol:   "HTTPS",
			AlpnPolicy: "HTTP2Preferred",
		}
	}

	if args.FixedResponse == nil {
		args.FixedResponse = &FixedResponse{
			ContentType: "text/plain",
			MessageBody: "Default Action: Not Found",
			StatusCode:  "404",
		}
	}

	ttl := 60
	if args.Proxied {
		ttl = 1
	}

	acmOutput, err := acm.CreateACM(ctx, &acm.ACMArgs{
		CloudZoneName: args.CloudZoneName,
		Environment:   args.Environment,
		Domain:        args.Domain,
	})
	if err != nil {
		return nil, err
	}

	subnets, err := ec2.GetSubnets(ctx, &ec2.GetSubnetsArgs{
		Filters: []ec2.GetSubnetsFilter{
			{
				Name: "vpc-id",
				Values: []string{
					args.VpcId,
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	loadBalancer, err := lb.NewLoadBalancer(ctx, "alb", &lb.LoadBalancerArgs{
		Name:                     pulumi.StringPtr(args.Name),
		Internal:                 pulumi.Bool(false),
		LoadBalancerType:         pulumi.String("application"),
		Subnets:                  pulumi.ToStringArray(subnets.Ids),
		SecurityGroups:           pulumi.ToStringArray(args.SecurityGroupIDs),
		EnableDeletionProtection: pulumi.Bool(false),
	})
	if err != nil {
		return nil, err
	}

	action := &lb.ListenerDefaultActionArgs{
		Type: pulumi.String("fixed-response"),
		FixedResponse: &lb.ListenerDefaultActionFixedResponseArgs{
			ContentType: pulumi.String(args.FixedResponse.ContentType),
			MessageBody: pulumi.String(args.FixedResponse.MessageBody),
			StatusCode:  pulumi.String(args.FixedResponse.StatusCode),
		},
	}
	if args.TargetGroupArn != "" {
		action = &lb.ListenerDefaultActionArgs{
			Type:           pulumi.String("forward"),
			TargetGroupArn: pulumi.String(args.TargetGroupArn),
		}
	}
	listener, err := lb.NewListener(ctx, "alb_listener", &lb.ListenerArgs{
		LoadBalancerArn: loadBalancer.Arn,
		Port:            pulumi.Int(args.Listener.Port),
		Protocol:        pulumi.String(args.Listener.Protocol),
		AlpnPolicy:      pulumi.String(args.Listener.AlpnPolicy),
		CertificateArn:  acmOutput.CertificateArn,
		SslPolicy:       pulumi.String("ELBSecurityPolicy-2016-08"),
		DefaultActions: lb.ListenerDefaultActionArray{
			action,
		},
	})
	if err != nil {
		return nil, err
	}

	for _, domain := range args.ExtraDomains {
		acmOutput, err := acm.CreateACM(ctx, &acm.ACMArgs{
			CloudZoneName: args.CloudZoneName,
			Environment:   args.Environment,
			Domain:        domain,
		})
		if err != nil {
			return nil, err
		}

		_, err = lb.NewListenerCertificate(ctx, "alb_listener_certificate", &lb.ListenerCertificateArgs{
			ListenerArn:    listener.Arn,
			CertificateArn: acmOutput.CertificateArn,
		})
		if err != nil {
			return nil, err
		}
	}

	zone, err := route53.LookupZone(ctx, &route53.LookupZoneArgs{
		Name:        pulumi.StringRef(args.Route53HostedZone),
		PrivateZone: pulumi.BoolRef(false),
	})
	if err != nil {
		return nil, err
	}

	if args.Domain != "" {
		_, err = route53.NewRecord(ctx, "alb_record", &route53.RecordArgs{
			ZoneId: pulumi.String(zone.ZoneId),
			Name:   pulumi.String(args.Domain),
			Type:   pulumi.String(route53.RecordTypeCNAME),
			Aliases: route53.RecordAliasArray{
				&route53.RecordAliasArgs{
					Name:   loadBalancer.DnsName,
					ZoneId: loadBalancer.ZoneId,
				},
			},
			Ttl:            pulumi.IntPtr(ttl),
			AllowOverwrite: pulumi.Bool(true),
		})
		if err != nil {
			return nil, err
		}
	}

	if len(args.ExtraDomains) > 0 {
		for _, domain := range args.ExtraDomains {
			_, err = route53.NewRecord(ctx, "alb_record", &route53.RecordArgs{
				ZoneId: pulumi.String(zone.ZoneId),
				Name:   pulumi.String(domain),
				Type:   pulumi.String(route53.RecordTypeCNAME),
				Aliases: route53.RecordAliasArray{
					&route53.RecordAliasArgs{
						Name:   loadBalancer.DnsName,
						ZoneId: loadBalancer.ZoneId,
					},
				},
				Ttl:            pulumi.IntPtr(ttl),
				AllowOverwrite: pulumi.Bool(true),
			})
			if err != nil {
				return nil, err
			}
		}
	}

	return nil, nil
}
