package acm

import (
	"strings"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/acm"
	"github.com/pulumi/pulumi-cloudflare/sdk/v5/go/cloudflare"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var _caaIssuers = []string{
	"amazon.com",
	"amazontrust.com",
	"awstrust.com",
	"amazonaws.com",
}

type ACMArgs struct {
	CloudZoneName string
	Environment   string
	Domain        string
}

type ACMOutput struct{}

func CreateACM(ctx *pulumi.Context, args *ACMArgs) (*ACMOutput, error) {
	cloudflareZone, err := cloudflare.LookupZone(ctx, &cloudflare.LookupZoneArgs{
		Name: &args.CloudZoneName,
	}, nil)
	if err != nil {
		return nil, err
	}

	caa, err := cloudflare.NewRecord(ctx, "caa", &cloudflare.RecordArgs{
		ZoneId: pulumi.String(cloudflareZone.ZoneId),
		Name:   pulumi.String(args.Domain),
		Type:   pulumi.String("CAA"),
		Data: cloudflare.RecordDataArgs{
			Flags: pulumi.StringPtr("0"),
			Tag:   pulumi.StringPtr("issue"),
			Value: pulumi.StringPtr(_caaIssuers[0]),
		},
		Ttl:            pulumi.Int(3600),
		Proxied:        pulumi.BoolPtr(true),
		AllowOverwrite: pulumi.BoolPtr(true),
	})
	if err != nil {
		return nil, err
	}

	certificate, err := acm.NewCertificate(ctx, "acm_cert", &acm.CertificateArgs{
		DomainName:       pulumi.String(args.Domain),
		ValidationMethod: pulumi.String("DNS"),
	}, pulumi.DependsOn([]pulumi.Resource{
		caa,
	}))
	if err != nil {
		return nil, err
	}

	_, err = acm.NewCertificateValidation(ctx, "acm_cert_validation", &acm.CertificateValidationArgs{
		CertificateArn: certificate.Arn,
		ValidationRecordFqdns: pulumi.StringArray{
			caa.Hostname,
		},
	})
	if err != nil {
		return nil, err
	}

	_ = certificate.DomainValidationOptions.ApplyT(func(options []acm.CertificateDomainValidationOption) error {
		for _, option := range options {
			_, err = cloudflare.NewRecord(ctx, "validation", &cloudflare.RecordArgs{
				ZoneId:         pulumi.String(cloudflareZone.ZoneId),
				Name:           pulumi.String(*option.ResourceRecordName),
				Type:           pulumi.String(*option.ResourceRecordType),
				Value:          pulumi.StringPtr(strings.TrimSuffix(*option.ResourceRecordValue, ".")),
				Ttl:            pulumi.Int(60),
				Proxied:        pulumi.BoolPtr(false),
				AllowOverwrite: pulumi.BoolPtr(true),
			})
			if err != nil {
				return err
			}
		}

		return nil
	})

	return nil, nil
}
