package bcc_terraform

import (
	"context"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourcePort() *schema.Resource {
	args := Defaults()
	args.injectContextRequiredVdc()
	args.injectContextDataPort()
	args.injectContextGetPort()

	return &schema.Resource{
		ReadContext: dataSourcePortRead,
		Schema:      args,
	}
}

func dataSourcePortRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	vdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-026] crash via getting vdc: %s", err)
	}

	var targetPort *bcc.Port
	_, okID := d.GetOk("id")
	_, okIP := d.GetOk("ip_address")

	if okID {
		targetPort, err = GetPortById(d, manager, vdc)
		if err != nil {
			return diag.Errorf("[ERROR-026] crash via getting port by id: %s", err)
		}
	} else if okIP {
		targetPort, err = GetPortByIp(d, manager, vdc)
		if err != nil {
			return diag.Errorf("[ERROR-026] crash via getting port by ip: %s", err)
		}
	} else {
		return diag.Errorf("[ERROR-026] for getting the port must be specified id or ip")
	}

	firewallTemplates := make([]interface{}, len(targetPort.FirewallTemplates))
	for i, firewall := range targetPort.FirewallTemplates {
		firewallTemplates[i] = firewall.ID
	}

	fields := map[string]interface{}{
		"id":                 targetPort.ID,
		"ip_address":         targetPort.IpAddress,
		"network":            targetPort.Network.ID,
		"firewall_templates": firewallTemplates,
		"tags":               marshalTagNames(targetPort.Tags),
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-026] crash via set attrs: %s", err)
	}
	return nil
}
