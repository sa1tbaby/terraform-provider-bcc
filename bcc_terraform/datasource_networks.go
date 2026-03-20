package bcc_terraform

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/hashstructure/v2"
)

func dataSourceNetworks() *schema.Resource {
	args := Defaults()
	args.injectContextRequiredVdc()
	args.injectContextDataNetworkList()

	return &schema.Resource{
		ReadContext: dataSourceNetworksRead,
		Schema:      args,
	}
}

func dataSourceNetworksRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	vdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-011] crash via getting vdc: %s", err)
	}

	networkList, err := vdc.GetNetworks()
	if err != nil {
		return diag.Errorf("[ERROR-011] crash via retrieving networks: %s", err)
	}

	networkMap := make([]map[string]interface{}, len(networkList))
	for i, network := range networkList {
		subnets, err := network.GetSubnets()
		if err != nil {
			return diag.Errorf("[ERROR-011] crash via getting subnets")
		}

		subnetMap := make([]map[string]interface{}, len(subnets))
		for i2, subnet := range subnets {
			dnsStrings := make([]string, len(subnet.DnsServers))
			for i3, dns := range subnet.DnsServers {
				dnsStrings[i3] = dns.DNSServer
			}

			subnetMap[i2] = map[string]interface{}{
				"id":       subnet.ID,
				"cidr":     subnet.CIDR,
				"dhcp":     subnet.IsDHCP,
				"gateway":  subnet.Gateway,
				"start_ip": subnet.StartIp,
				"end_ip":   subnet.EndIp,
				"dns":      dnsStrings,
			}
		}

		networkMap[i] = map[string]interface{}{
			"id":       network.ID,
			"name":     network.Name,
			"mtu":      network.Mtu,
			"subnets":  subnetMap,
			"external": network.External,
			"tags":     marshalTagNames(network.Tags),
		}
	}

	hash, err := hashstructure.Hash(networkList, hashstructure.FormatV2, nil)
	if err != nil {
		diag.Errorf("[ERROR-011] crash via calculate hash: %s", err)
	}

	fields := map[string]interface{}{
		"id":       fmt.Sprintf("networks/%d", hash),
		"vdc_id":   vdc.ID,
		"networks": networkMap,
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-011] crash via set attrs: %s", err)
	}

	return nil
}
