package record

import (
	"strings"

	"github.com/pulumi/pulumi-cloudflare/sdk/v5/go/cloudflare"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type RecordArgs struct {
	Proxied bool
	Ttl     int
	Type    string
	Value   string
	Domain  string
}

type RecordOutput struct {
	CloudflareZoneName string
}

func CreateRecord(ctx *pulumi.Context, args *RecordArgs) (*RecordOutput, error) {
	parts := strings.Split(args.Domain, ".")
	zoneName := strings.Join(parts[len(parts)-2:], ".")
	ttl := args.Ttl
	if args.Proxied {
		ttl = 1
	}

	zone, err := cloudflare.LookupZone(ctx, &cloudflare.LookupZoneArgs{
		Name: pulumi.StringRef(zoneName),
	})
	if err != nil {
		return nil, err
	}

	_, err = cloudflare.NewRecord(ctx, "record", &cloudflare.RecordArgs{
		ZoneId:         pulumi.String(zone.ZoneId),
		Name:           pulumi.String(args.Domain),
		Type:           pulumi.String(args.Type),
		Proxied:        pulumi.Bool(args.Proxied),
		Value:          pulumi.String(args.Value),
		Ttl:            pulumi.Int(ttl),
		AllowOverwrite: pulumi.Bool(true),
	})
	if err != nil {
		return nil, err
	}

	return &RecordOutput{
		CloudflareZoneName: zoneName,
	}, nil
}
