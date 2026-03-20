package bcc_terraform

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func (args *Arguments) injectContextGetDns() {
	args.merge(Arguments{
		"name": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "dns name",
		},
		"id": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "dns identifier",
		},
	})
}

func (args *Arguments) injectContextRequiredDns() {
	args.merge(Arguments{
		"dns_id": {
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
			Description: "id of the Dns",
		},
	})
}

func (args *Arguments) injectContextResourceDns() {
	args.merge(Arguments{
		"name": {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
			ValidateFunc: validation.All(
				validation.NoZeroValues,
				validation.StringMatch(regexp.MustCompile(`\.$`), "DNS name must end by dot"),
				validation.StringLenBetween(1, 255),
			),
			Description: "name of the Dns",
		},
		"tags": newTagNamesResourceSchema("tags of the Vm"),
	})
}

func (args *Arguments) injectContextDataDns() {
	args.merge(Arguments{
		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "id of the Dns",
		},
		"name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "name of the Dns",
		},
		"tags": newTagNamesDataSchema("tags of the Dns"),
	})
}

func (args *Arguments) injectContextDataDnsList() {
	s := Defaults()
	s.injectContextDataDns()

	args.merge(Arguments{
		"dnss": {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: s,
			},
		},
	})
}
