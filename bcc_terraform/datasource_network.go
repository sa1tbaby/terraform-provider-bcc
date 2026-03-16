package bcc_terraform

import (
	"context"
	"strings"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceNetwork() *schema.Resource {
	args := Defaults()
	args.injectContextDataNetwork()
	args.injectContextRequiredVdcForData()
	args.injectContextGetNetwork()

	return &schema.Resource{
		ReadContext: dataSourceNetworkRead,
		Schema:      args,
	}
}

func dataSourceNetworkRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	target, err := checkDatasourceNameOrId(d)
	if err != nil {
		return diag.Errorf("[ERROR-010] crash via chose target: %s", err)
	}

	var network *bcc.Network
	if strings.EqualFold(target, "id") {
		networkId := d.Get("id").(string)
		network, err = manager.GetNetwork(networkId)
		if err != nil {
			return diag.Errorf("[ERROR-010] crash via getting network by id=%s: %s", networkId, err)
		}
	} else if strings.EqualFold(target, "name") {
		vdc, err := GetVdcById(d, manager)
		if err != nil {
			return diag.Errorf("[ERROR-010] crash via getting vdc: %s", err)
		}

		network, err = GetNetworkByName(d, manager, vdc)
		if err != nil {
			return diag.Errorf("[ERROR-010] crash via getting network: %s", err)

		}
	} else {
		return diag.Errorf("For getting the network must be specified id or name")
	}

	subnets, err := network.GetSubnets()
	if err != nil {
		return diag.Errorf("[ERROR-010] crash via getting subnets")
	}

	subnetsMap := make([]map[string]interface{}, len(subnets))
	for i, subnet := range subnets {
		dnsStrings := make([]string, len(subnet.DnsServers))
		for j, dns := range subnet.DnsServers {
			dnsStrings[j] = dns.DNSServer
		}

		subnetsMap[i] = map[string]interface{}{
			"id":       subnet.ID,
			"cidr":     subnet.CIDR,
			"dhcp":     subnet.IsDHCP,
			"gateway":  subnet.Gateway,
			"start_ip": subnet.StartIp,
			"end_ip":   subnet.EndIp,
			"dns":      dnsStrings,
		}
	}

	fields := map[string]interface{}{
		"id":       network.ID,
		"name":     network.Name,
		"vdc_id":   network.Vdc.Id,
		"subnets":  subnetsMap,
		"mtu":      network.Mtu,
		"external": network.External,
		"tags":     marshalTagNames(network.Tags),
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-010] crash via set attrs: %s", err)
	}

	return nil
}
