package bcc_terraform

import (
	"fmt"
	"log"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
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

func (args *Arguments) injectContextResourceNetwork() {

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
		"external": {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "whether the network is external",
		},
		"tags": newTagNamesResourceSchema("tags of the Network"),
	})
}

func (args *Arguments) injectContextDataNetwork() {
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
		"external": {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "whether the network is external",
		},
		"tags": newTagNamesDataSchema("tags of the Network"),
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

func (args *Arguments) injectContextDataNetworkList() {
	s := Defaults()
	s.injectContextDataNetwork()

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

func createSubnet(d *schema.ResourceData, manager *bcc.Manager, network *bcc.Network) (err error) {
	subnets := d.Get("subnets").([]interface{})
	log.Printf("[DEBUG] subnets: %#v", subnets)

	for _, subnetInfo := range subnets {
		log.Printf("[DEBUG] subnetInfo: %#v", subnetInfo)
		subnetInfo2 := subnetInfo.(map[string]interface{})

		// Create subnet
		subnet := bcc.NewSubnet(subnetInfo2["cidr"].(string), subnetInfo2["gateway"].(string),
			subnetInfo2["start_ip"].(string), subnetInfo2["end_ip"].(string), subnetInfo2["dhcp"].(bool))
		if err = network.CreateSubnet(&subnet); err != nil {
			return fmt.Errorf("[ERROR-009]: %s", err)
		}

		dnsServersList := subnetInfo2["dns"].([]interface{})
		if len(dnsServersList) > 0 {
			dnsServers := make([]*bcc.SubnetDNSServer, len(dnsServersList))
			for i, dns := range dnsServersList {
				s1 := bcc.NewSubnetDNSServer(dns.(string))
				dnsServers[i] = &s1
			}

			if err = subnet.UpdateDNSServers(dnsServers); err != nil {
				return fmt.Errorf("[ERROR-009]: %s", err)
			}
		}

	}

	return
}

func updateSubnet(d *schema.ResourceData, manager *bcc.Manager) (diagErr diag.Diagnostics) {
	network, err := manager.GetNetwork(d.Id())
	if err != nil {
		return diag.Errorf("[ERROR-009]: %s", err)
	}

	subnetsRaw, err := network.GetSubnets()
	if err != nil {
		return diag.Errorf("[ERROR-009]: %s", err)
	}

	subnets := d.Get("subnets").([]interface{})
	for _, subnetInfo := range subnets {
		subnetInfo2 := subnetInfo.(map[string]interface{})
		var subnet *bcc.Subnet
		for _, currentSubnet := range subnetsRaw {
			if currentSubnet.CIDR == subnetInfo2["cidr"] {
				subnet = currentSubnet
				break
			}
		}
		dnsServersList := subnetInfo2["dns"].([]interface{})
		newDnsServers := make([]*bcc.SubnetDNSServer, len(dnsServersList))
		for i, dns := range dnsServersList {
			s1 := bcc.NewSubnetDNSServer(dns.(string))
			newDnsServers[i] = &s1
		}
		if subnet == nil {
			// create new subnet
			newSubnet := bcc.NewSubnet(subnetInfo2["cidr"].(string), subnetInfo2["gateway"].(string),
				subnetInfo2["start_ip"].(string), subnetInfo2["end_ip"].(string), subnetInfo2["dhcp"].(bool))
			if err = network.CreateSubnet(&newSubnet); err != nil {
				return diag.Errorf("[ERROR-009]: %s", err)
			}
			if err = newSubnet.UpdateDNSServers(newDnsServers); err != nil {
				return diag.Errorf("[ERROR-009]: %s", err)
			}
		} else {
			// update preserved subnet
			shouldUpdate := false
			if subnet.Gateway != subnetInfo2["gateway"] {
				return diag.Errorf("[ERROR-009]: You cannot change gateway")
			}
			if subnet.StartIp != subnetInfo2["start_ip"] || subnet.EndIp != subnetInfo2["end_ip"] || subnet.IsDHCP != subnetInfo2["dhcp"] {
				subnet.EndIp = subnetInfo2["end_ip"].(string)
				subnet.StartIp = subnetInfo2["start_ip"].(string)
				subnet.IsDHCP = subnetInfo2["dhcp"].(bool)
				shouldUpdate = true
			}
			if len(subnet.DnsServers) != len(newDnsServers) {
				subnet.DnsServers = newDnsServers
				shouldUpdate = true
			} else {
				for i, oldDns := range subnet.DnsServers {
					if oldDns.DNSServer != newDnsServers[i].DNSServer {
						subnet.DnsServers = newDnsServers
						shouldUpdate = true
						break
					}
				}
			}
			if shouldUpdate {
				if err = subnet.UpdateDNSServers(subnet.DnsServers); err != nil {
					return diag.Errorf("[ERROR-009]: %s", err)
				}
			}
		}
	}
	for _, subnet := range subnetsRaw {
		var subnetInfo2 map[string]interface{}
		for _, subnetInfo := range subnets {
			subnetInfo2 = subnetInfo.(map[string]interface{})
			if subnet.CIDR == subnetInfo2["cidr"] {
				break
			}
		}
		if subnetInfo2 == nil {
			// delete obsolete subnet
			if err := subnet.Delete(); err != nil {
				return diag.Errorf("[ERROR-009]: %s", err)
			}
		}
	}

	return
}
