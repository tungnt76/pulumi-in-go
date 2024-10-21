package acm

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/acm"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/route53"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var _caaIssuers = []string{
	"amazon.com",
	"amazontrust.com",
	"awstrust.com",
	"amazonaws.com",
}

type ACMArgs struct {
	Route53HostedZone string
	CloudZoneName     string
	Domain            string
	Environment       string
}

type ACMOutput struct {
	CertificateArn pulumi.StringOutput
	CAAIssuers     []string
}

func CreateACM(ctx *pulumi.Context, args *ACMArgs) (*ACMOutput, error) {
	zone, err := route53.LookupZone(ctx, &route53.LookupZoneArgs{
		Name:        pulumi.StringRef(args.Route53HostedZone),
		PrivateZone: pulumi.BoolRef(true),
	})
	if err != nil {
		return nil, err
	}

	_, err = route53.NewRecord(ctx, "caa", &route53.RecordArgs{
		ZoneId: pulumi.String(zone.ZoneId),
		Name:   pulumi.String(args.Domain),
		Type:   pulumi.String(route53.RecordTypeCAA),
		Ttl:    pulumi.Int(86400),
		Records: func() pulumi.StringArray {
			var records []string
			for _, issuer := range _caaIssuers {
				records = append(records, fmt.Sprintf("%s %s", issuer, "0 issue;"))
			}
			return pulumi.ToStringArray(records)
		}(),
		AllowOverwrite: pulumi.Bool(true),
	})
	if err != nil {
		return nil, err
	}

	certificate, err := acm.NewCertificate(ctx, "caa", &acm.CertificateArgs{
		DomainName:       pulumi.String(args.Domain),
		ValidationMethod: pulumi.String("DNS"),
		Tags: pulumi.ToStringMap(map[string]string{
			"Environment": args.Environment,
		}),
	})
	if err != nil {
		return nil, err
	}

	fqdns := []string{}
	_ = certificate.DomainValidationOptions.ApplyT(func(options []acm.CertificateDomainValidationOption) error {
		for _, option := range options {
			record, err := route53.NewRecord(ctx, "route53_record", &route53.RecordArgs{
				ZoneId:         pulumi.String(zone.ZoneId),
				Name:           pulumi.String(*option.ResourceRecordName),
				Type:           pulumi.String(*option.ResourceRecordType),
				Records:        pulumi.ToStringArray([]string{*option.ResourceRecordValue}),
				Ttl:            pulumi.Int(60),
				AllowOverwrite: pulumi.Bool(true),
			})
			if err != nil {
				return err
			}

			record.Fqdn.ApplyT(func(fqdn string) error {
				fqdns = append(fqdns, fqdn)
				return nil
			})
		}

		return nil
	})

	if len(fqdns) > 0 {
		_, err = acm.NewCertificateValidation(ctx, "certificate_validation", &acm.CertificateValidationArgs{
			CertificateArn:        certificate.Arn,
			ValidationRecordFqdns: pulumi.ToStringArray(fqdns),
		})
		if err != nil {
			return nil, err
		}
	}

	return &ACMOutput{
		CertificateArn: certificate.Arn,
		CAAIssuers:     _caaIssuers,
	}, nil
}
