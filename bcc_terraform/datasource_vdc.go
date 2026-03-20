package bcc_terraform

import (
	"context"
	"strings"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVdc() *schema.Resource {
	args := Defaults()
	args.injectContextOptionalProject()
	args.injectContextDataVdc()
	args.injectContextGetVdc()

	return &schema.Resource{
		ReadContext: dataSourceVdcRead,
		Schema:      args,
	}
}

func dataSourceVdcRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	target, err := checkDatasourceNameOrId(d)
	if err != nil {
		return diag.Errorf("[ERROR-007] crash via chose target: %s", err)
	}

	var vdc *bcc.Vdc
	if strings.EqualFold(target, "id") {
		vdc, err = manager.GetVdc(d.Get("id").(string))
		if err != nil {
			return diag.Errorf("[ERROR-007] crash via getting VDC by id: %s", err)
		}

	} else if strings.EqualFold(target, "name") {
		if _, exist := d.GetOk("project_id"); !exist {
			return diag.Errorf("[ERROR-007] project_id must be specified when you would get Vdc by name")
		}

		project, err := GetProjectById(d, manager)
		if err != nil {
			return diag.Errorf("[ERROR-007] crash via getting project: %s", err)
		}

		vdc, err = GetVdcByName(d, manager, project)
		if err != nil {
			return diag.Errorf("[ERROR-007] crash via getting VDC by name: %s", err)
		}
	} else {
		return diag.Errorf("[ERROR-007] Invalid target: %s", target)
	}

	fields := map[string]interface{}{
		"id":              vdc.ID,
		"name":            vdc.Name,
		"hypervisor":      vdc.Hypervisor.Name,
		"hypervisor_type": vdc.Hypervisor.Type,
		"tags":            marshalTagNames(vdc.Tags),
	}

	networks, err := vdc.GetNetworks(bcc.Arguments{"defaults_only": "true"})
	if err != nil {
		return diag.Errorf("[ERROR-007]: %s", err)
	}

	if len(networks) != 0 {
		network := networks[0]
		subnets, err := network.GetSubnets()
		if err != nil {
			return diag.Errorf("[ERROR-007]: %s", err)
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

		fields["default_network_subnets"] = flattenedSubnets
		fields["default_network_mtu"] = network.Mtu
		fields["default_network_name"] = network.Name
		fields["default_network_id"] = network.ID
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-007] crash via set attrs: %s", err)
	}

	return nil
}
