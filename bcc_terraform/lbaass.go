package bcc_terraform

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func (args *Arguments) injectContextGetLbaas() {
	args.merge(Arguments{
		"name": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "Lbaas name",
		},
		"id": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "Lbaas identifier",
		},
	})
}

func (args *Arguments) injectContextLbaasByID() {
	args.merge(Arguments{
		"lbaas_id": {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
			ValidateDiagFunc: validation.ToDiagFunc(
				validation.StringIsNotEmpty,
			),
			Description: "id of the Lbaas",
		},
	})
}

func (args *Arguments) injectLbaasPort() {
	args.merge(Arguments{
		"network_id": {
			Type:     schema.TypeString,
			ForceNew: true,
			Required: true,
			ValidateDiagFunc: validation.ToDiagFunc(
				validation.StringIsNotEmpty,
			),
			Description: "id of the Network",
		},
		"ip_address": {
			Type:        schema.TypeString,
			Optional:    true,
			Default:     "",
			Description: "ip_address of the Port",
			DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
				return new == ""
			},
		},
	})
}

func (args *Arguments) injectDataLbaasPort() {
	args.merge(Arguments{
		"network_id": {
			Type:     schema.TypeString,
			Computed: true,
			ValidateDiagFunc: validation.ToDiagFunc(
				validation.StringIsNotEmpty,
			),
			Description: "id of the Network",
		},
		"ip_address": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "ip_address of the Port",
			DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
				return new == ""
			},
		},
	})
}

func (args *Arguments) injectContextResourceLbaas() {
	lbaasPort := Defaults()
	lbaasPort.injectLbaasPort()

	args.merge(Arguments{
		"name": {
			Type:     schema.TypeString,
			Required: true,
			ValidateFunc: validation.All(
				validation.NoZeroValues,
				validation.StringLenBetween(1, 100),
			),
			Description: "name of the Lbaas",
		},
		"port": {
			Type:     schema.TypeList,
			Required: true,
			MinItems: 1,
			MaxItems: 1,
			Elem: &schema.Resource{
				Schema: lbaasPort,
			},
			Description: "Lbaas port.",
		},
		"floating": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "enable floating ip for the Lbaas",
		},
		"floating_ip": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "floating ip for the Lbaas. May be comitted",
		},
		"tags": newTagNamesResourceSchema("tags of the Lbaas"),
	})
}

func (args *Arguments) injectContextDataLbaas() {
	lbaasPort := Defaults()
	lbaasPort.injectLbaasPort()

	args.merge(Arguments{
		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Lbaas identifier",
		},
		"name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Lbaas name",
		},
		"port": {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{

				Schema: lbaasPort,
			},
			Description: "Lbaas port.",
		},
		"floating": {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "whether the load balancer has a public IP address",
		},
		"floating_ip": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "public IP address of the load balancer. May be omitted",
		},
		"tags": newTagNamesDataSchema("tags of the Lbaas"),
	})
}

func (args *Arguments) injectContextDataLbaasList() {
	s := Defaults()
	s.injectContextDataLbaas()

	args.merge(Arguments{
		"lbaass": {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: s,
			},
		},
	})
}
