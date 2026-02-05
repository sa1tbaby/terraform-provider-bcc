package bcc_terraform

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func (args *Arguments) injectContextGetHypervisor() {
	args.merge(Arguments{
		"name": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "resource pool name",
		},
		"id": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "resource pool identifier",
		},
	})
}

func (args *Arguments) injectContextHypervisorById() {
	args.merge(Arguments{
		"hypervisor_id": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "name of the Hypervisor",
		},
	})
}

func (args *Arguments) injectResultHypervisor() {
	args.merge(Arguments{
		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "resource pool identifier",
		},
		"name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "resource pool name",
		},
		"type": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "resource pool type",
		},
	})
}

func (args *Arguments) injectResultListHypervisor() {
	s := Defaults()
	s.injectResultHypervisor()

	args.merge(Arguments{
		"hypervisors": {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: s,
			},
		},
	})
}
