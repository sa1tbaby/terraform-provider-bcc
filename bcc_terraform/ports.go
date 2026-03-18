package bcc_terraform

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func (args *Arguments) injectContextResourcePort() {
	args.merge(Arguments{
		"vdc_id": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "id of the VDC",
		},
		"network_id": {
			Type:        schema.TypeString,
			ForceNew:    true,
			Required:    true,
			Description: "id of the Network",
		},
		"ip_address": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "ip_address of the Port",
			ValidateFunc: validation.All(
				validation.StringIsNotEmpty,
				validation.StringDoesNotMatch(regexp.MustCompile(`^0\.0\.0\.0`), "remove ip_address to choose random IP"),
			),
		},
		"firewall_templates": {
			Type:        schema.TypeSet,
			Optional:    true,
			Computed:    true,
			Description: "list of firewall templates ids of the Port",
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"tags": newTagNamesResourceSchema("tags of the Port"),
	})
}

func (args *Arguments) injectContextGetPort() {
	args.merge(Arguments{
		"id": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "Port identifier",
		},
	})
}

func (args *Arguments) injectContextDataPort() {
	args.merge(Arguments{
		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Port identifier",
		},
		"network": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Network identifier",
		},
		"ip_address": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "ip_address of the Port",
		},
		"firewall_templates": {
			Type:        schema.TypeSet,
			Computed:    true,
			Description: "list of firewall templates ids of the Port",
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"tags": newTagNamesDataSchema("tags of the Port"),
	})
}

func (args *Arguments) injectContextDataPortList() {
	Port := Defaults()
	Port.injectContextDataPort()

	args.merge(Arguments{
		"ports": {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: Port,
			},
		},
	})
}
