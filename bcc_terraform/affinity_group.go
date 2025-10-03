package bcc_terraform

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func (args *Arguments) injectCreateAffinityGroup() {
	args.merge(Arguments{
		"name": {
			Type:     schema.TypeString,
			Required: true,
			ValidateFunc: validation.All(
				validation.NoZeroValues,
				validation.StringLenBetween(1, 100),
			),
			Description: "name of the affinity group",
		},
		"description": {
			Type:        schema.TypeString,
			Optional:    true,
			Default:     "",
			Description: "description of the affinity group",
		},
		"policy": {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
			ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
				v := val.(string)
				validValues := map[string]bool{
					"soft-affinity":   true,
					"strong-affinity": true,
				}
				if !validValues[v] {
					errs = append(errs, fmt.Errorf("%q must be one of 'soft-affinity' or 'strong-affinity'", key))
				}
				return
			},
			Description: "The affinity type. Can be either 'soft-affinity' or 'strong-affinity'.",
		},
		"reboot": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Whether to reboot the affinity group",
		},
		"vms": {
			Type:        schema.TypeList,
			Optional:    true,
			Default:     make(map[string]interface{}),
			Description: "List of VMs",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"id": {
						Type:        schema.TypeString,
						Required:    true,
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
