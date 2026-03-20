package bcc_terraform

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func (args *Arguments) injectContextResourceAffinityGroup() {
	args.merge(Arguments{
		"name": {
			Type:     schema.TypeString,
			Required: true,
			ValidateFunc: validation.All(
				validation.NoZeroValues,
				validation.StringLenBetween(1, 100),
			),
			Description: "affinity group name",
		},
		"description": {
			Type:        schema.TypeString,
			Optional:    true,
			Default:     "",
			Description: "affinity group description",
		},
		"policy": {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
			ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
				v := val.(string)
				validValues := map[string]bool{
					"soft-affinity":      true,
					"soft-anti-affinity": true,
				}
				if !validValues[v] {
					errs = append(errs, fmt.Errorf("%q must be one of 'soft-affinity' or 'soft-anti-affinity'", key))
				}
				return
			},
			Description: "The affinity type. Can be either 'soft-affinity' or 'soft-anti-affinity'.",
		},
		"vms": {
			Type:        schema.TypeList,
			Computed:    true,
			Description: "List of VMs",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"id": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "Id of the vm",
					},
					"name": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "Name of the vm",
					},
				},
			},
		},
	})
}

func (args *Arguments) injectContextGetAffinityGroup() {
	args.merge(Arguments{
		"id": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "affinity group identifier",
		},
		"name": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "affinity group name",
		},
	})
}

func (args *Arguments) injectContextDataAffinityGroup() {
	args.merge(Arguments{
		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "affinity group identifier",
		},
		"name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "affinity group name",
		},
		"description": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "affinity group description",
		},
		"policy": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "affinity group policy",
		},
		"vms": {
			Type:        schema.TypeList,
			Computed:    true,
			Description: "List of VMs",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"id": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "identifier of the vm in group",
					},
					"name": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "name of the vm in group",
					},
				},
			},
		},
	})

}

func (args *Arguments) injectContextDataListAffinityGroup() {
	s := Defaults()
	s.injectContextDataAffinityGroup()

	args.merge(Arguments{
		"affinity_groups": {
			Type:        schema.TypeList,
			Computed:    true,
			Description: "List of affinity groups",
			Elem: &schema.Resource{
				Schema: s,
			},
		},
	})
}
