package bcc_terraform

import (
	"context"
	"fmt"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/hashstructure/v2"
)

func dataSourceVdcs() *schema.Resource {
	args := Defaults()
	args.injectContextRequiredProject()
	args.injectContextDataListVdc()

	return &schema.Resource{
		ReadContext: dataSourceVdcsRead,
		Schema:      args,
	}
}

func dataSourceVdcsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	project, err := GetProjectById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-008] crash via getting project: %s", err)
	}

	vdcList, err := manager.GetVdcs(bcc.Arguments{"project": project.ID})
	if err != nil {
		return diag.Errorf("[ERROR-008] crash via retrieving vdcs: %s", err)
	}

	vdcMap := make([]map[string]interface{}, len(vdcList))
	for i, vdc := range vdcList {
		vdcMap[i] = map[string]interface{}{
			"id":              vdc.ID,
			"name":            vdc.Name,
			"hypervisor":      vdc.Hypervisor.Name,
			"hypervisor_type": vdc.Hypervisor.Type,
			"tags":            marshalTagNames(vdc.Tags),
		}

		networks, err := vdc.GetNetworks(bcc.Arguments{"defaults_only": "true"})
		if err != nil {
			return diag.Errorf("[ERROR-006]: %s", err)
		}

		if len(networks) != 0 {
			network := networks[0]
			subnets, err := network.GetSubnets()
			if err != nil {
				return diag.Errorf("[ERROR-006]: %s", err)
			}

			flattenedSubnets := make([]map[string]interface{}, len(subnets))
			for i, subnet := range subnets {
				dnsStrings := make([]string, len(subnet.DnsServers))
				for i2, dns := range subnet.DnsServers {
					dnsStrings[i2] = dns.DNSServer
				}
				flattenedSubnets[i] = map[string]interface{}{
					"id":       subnet.ID,
					"cidr":     subnet.CIDR,
					"dhcp":     subnet.IsDHCP,
					"gateway":  subnet.Gateway,
					"start_ip": subnet.StartIp,
					"end_ip":   subnet.EndIp,
					"dns":      dnsStrings,
				}
			}

			vdcMap[i]["default_network_subnets"] = flattenedSubnets
			vdcMap[i]["default_network_mtu"] = network.Mtu
			vdcMap[i]["default_network_name"] = network.Name
			vdcMap[i]["default_network_id"] = network.ID
		}
	}

	hash, err := hashstructure.Hash(vdcList, hashstructure.FormatV2, nil)
	if err != nil {
		diag.Errorf("[ERROR-008] crash via calculate hash: %s", err)
	}

	fields := map[string]interface{}{
		"id":         fmt.Sprintf("vdcs/%d", hash),
		"project_id": project.ID,
		"vdcs":       vdcMap,
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-008] crash via set attrs: %s", err)
	}

	return nil
}
