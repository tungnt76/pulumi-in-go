package listenerrule

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/lb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type ListenerRuleArgs struct {
	HostHeaders    []string
	PathPatterns   []string
	ListenerArn    string
	TargetGroupArn string
}

type ListenerRuleOutput struct{}

func CreateListenerRule(ctx *pulumi.Context, args ListenerRuleArgs) (*ListenerRuleOutput, error) {
	if len(args.PathPatterns) == 0 {
		args.PathPatterns = []string{"/"}
	}

	conditions := lb.ListenerRuleConditionArray{}
	if len(args.HostHeaders) > 0 {
		conditions = append(conditions, lb.ListenerRuleConditionArgs{
			HostHeader: lb.ListenerRuleConditionHostHeaderArgs{
				Values: pulumi.ToStringArray(args.HostHeaders),
			},
		})
	}
	if len(args.PathPatterns) > 0 {
		conditions = append(conditions, lb.ListenerRuleConditionArgs{
			PathPattern: lb.ListenerRuleConditionPathPatternArgs{
				Values: pulumi.ToStringArray(args.PathPatterns),
			},
		})
	}

	_, err := lb.NewListenerRule(ctx, "listener-rule", &lb.ListenerRuleArgs{
		ListenerArn: pulumi.String(args.ListenerArn),
		Actions: lb.ListenerRuleActionArray{
			lb.ListenerRuleActionArgs{
				Type:           pulumi.String("forward"),
				TargetGroupArn: pulumi.String(args.TargetGroupArn),
			},
		},
		Conditions: conditions,
	})
	if err != nil {
		return nil, err
	}

	return &ListenerRuleOutput{}, nil
}
