package bcc_terraform

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func (args *Arguments) injectContextGetNetwork() {
	args.merge(Arguments{
		"name": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "Network name",
		},
		"id": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "Network identifier",
		},
	})
}

func (args *Arguments) injectCreateNetwork() {

	args.merge(Arguments{
		"name": {
			Type:     schema.TypeString,
			Required: true,
			ValidateFunc: validation.All(
				validation.NoZeroValues,
				validation.StringLenBetween(1, 100),
			),
			Description: "name of the Network",
		},
		"subnets": {
			Type:     schema.TypeList,
			Required: true,
			MinItems: 1,
			MaxItems: 1, //  doesn't support several subnets
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"id": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "id of the Subnet",
					},
					"cidr": {
						Type:        schema.TypeString,
						ForceNew:    true,
						Required:    true,
						Description: "cidr of the Subnet",
						ValidateFunc: validation.All(
							validation.NoZeroValues,
							validation.StringLenBetween(1, 100),
						),
					},
					"gateway": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "gateway of the Subnet",
						ValidateFunc: validation.All(
							validation.NoZeroValues,
							validation.StringLenBetween(1, 100),
						),
					},
					"start_ip": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "pool start ip of the Subnet",
						ValidateFunc: validation.All(
							validation.NoZeroValues,
							validation.StringLenBetween(1, 100),
						),
					},
					"end_ip": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "pool end ip of the Subnet",
						ValidateFunc: validation.All(
							validation.NoZeroValues,
							validation.StringLenBetween(1, 100),
						),
					},
					"dhcp": {
						Type:        schema.TypeBool,
						Required:    true,
						Description: "enable dhcp service of the Subnet",
					},
					"dns": {
						Type:     schema.TypeList,
						Optional: true,
						Elem: &schema.Schema{
							Type: schema.TypeString,
						},
						Description: "dns servers list",
					},
				},
			},
		},
		"mtu": {
			Type:     schema.TypeInt,
			Optional: true,
			Computed: true,
		},
		"tags": newTagNamesResourceSchema("tags of the Network"),
	})
}

func (args *Arguments) injectResultNetwork() {
	args.merge(Arguments{
		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Network identifier",
		},
		"name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Network name",
		},
		"mtu": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "maximum transmission unit (MTU) of packets in the network;",
		},
		"subnets": {
			Type:        schema.TypeList,
			Computed:    true,
			Description: "list of subnets",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"id": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "Subnet identifier",
					},
					"cidr": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "Subnet cidr",
					},
					"gateway": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "subnet gateway IP address",
					},
					"start_ip": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "starting IP address of the subnet IP range",
					},
					"end_ip": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "ending IP address of the subnet IP range",
					},
					"dhcp": {
						Type:        schema.TypeBool,
						Computed:    true,
						Description: "whether DHCP is enabled for the subnet (true or false)",
					},
					"dns": {
						Type:        schema.TypeList,
						Computed:    true,
						Elem:        &schema.Schema{Type: schema.TypeString},
						Description: "list of DNS servers for the subnet",
					},
				},
			},
		},
	})
}

func (args *Arguments) injectResultListNetwork() {
	s := Defaults()
	s.injectResultNetwork()

	args.merge(Arguments{
		"networks": {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: s,
			},
		},
	})
}
