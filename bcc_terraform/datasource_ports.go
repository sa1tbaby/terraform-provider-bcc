package bcc_terraform

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/hashstructure/v2"
)

func dataSourcePorts() *schema.Resource {
	args := Defaults()
	args.injectContextRequiredVdc()
	args.injectContextDataPortList()

	return &schema.Resource{
		ReadContext: dataSourcePortsRead,
		Schema:      args,
	}
}

func dataSourcePortsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	vdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-027] crash via getting vdc: %s", err)
	}

	portList, err := vdc.GetPorts()
	if err != nil {
		return diag.Errorf("[ERROR-027] crash via retrieving ports: %s", err)
	}

	portMap := make([]map[string]interface{}, len(portList))
	for i, port := range portList {
		firewallTemplates := make([]string, len(port.FirewallTemplates))
		for i, firewall := range port.FirewallTemplates {
			firewallTemplates[i] = firewall.ID
		}

		portMap[i] = map[string]interface{}{
			"id":                 port.ID,
			"ip_address":         port.IpAddress,
			"network":            port.Network.ID,
			"firewall_templates": firewallTemplates,
			"tags":               marshalTagNames(port.Tags),
		}
	}

	hash, err := hashstructure.Hash(portList, hashstructure.FormatV2, nil)
	if err != nil {
		diag.Errorf("[ERROR-027] crash via calculate hash: %s", err)
	}

	fields := map[string]interface{}{
		"id":     fmt.Sprintf("port/%d", hash),
		"vdc_id": vdc.ID,
		"ports":  portMap,
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-027] crash via set attrs: %s", err)
	}
	return nil
}
