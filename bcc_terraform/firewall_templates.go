package bcc_terraform

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func (args *Arguments) injectContextGetFirewallTemplate() {
	args.merge(Arguments{
		"id": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "firewall template identifier",
		},
		"name": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "firewall template name",
		},
	})
}

func (args *Arguments) injectContextFirewallTemplateById() {
	args.merge(Arguments{
		"firewall_id": {
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
			Description: "id of the Firewall Template",
		},
	})
}

func (args *Arguments) injectContextDataFirewallTemplate() {

	args.merge(Arguments{
		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "id of the Firewall Template",
		},
		"name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "name of the Firewall Template",
		},
		"description": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "description of the Firewall Template",
		},
		"rules_count": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "number of rules in the Firewall Template",
		},
		"tags": newTagNamesDataSchema("tags of the firewall template"),
	})
}

func (args *Arguments) injectContextDataFirewallTemplateList() {
	firewallTemplate := Defaults()
	firewallTemplate.injectContextDataFirewallTemplate()

	args.merge(Arguments{
		"firewall_templates": {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: firewallTemplate,
			},
		},
	})
}

func (args *Arguments) injectContextResourceFirewallTemplate() {
	args.merge(Arguments{
		"name": {
			Type:     schema.TypeString,
			Required: true,
			ValidateFunc: validation.All(
				validation.NoZeroValues,
				validation.StringLenBetween(1, 100),
			),
			Description: "name of the firewall template",
		},
		"description": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "description of the firewall template",
		},
		"rules_count": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "number of rules in the firewall template",
		},
		"tags": newTagNamesResourceSchema("tags of the firewall template"),
	})
}

func setUpRule(rule *bcc.FirewallRule, d *schema.ResourceData) (err error) {
	rule.DstPortRangeMax = nil
	rule.DstPortRangeMin = nil
	portRange := d.Get("port_range").(string)

	if portRange == "" {
		return nil
	}
	var min, max int
	var re_for_port_range = regexp.MustCompile(`(?m)^(\d+:\d+)$`)
	var re_for_port = regexp.MustCompile(`(?m)^(\d+)$`)
	if len(re_for_port_range.FindStringIndex(portRange)) > 0 {
		fmt.Sscanf(portRange, "%d:%d", &min, &max)
		rule.DstPortRangeMax = &max
		rule.DstPortRangeMin = &min
	} else if len(re_for_port.FindStringIndex(portRange)) > 0 {
		fmt.Sscanf(portRange, "%d", &min)
		rule.DstPortRangeMin = &min
	} else {
		return errors.New("PORT RANGE UNSUPPORTED FORMAT, " +
			"should be `val:val` or `val`")
	}

	return nil
}
