package bcc_terraform

import (
	"context"
	"strings"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceLbaas() *schema.Resource {
	args := Defaults()
	args.injectContextRequiredVdc()
	args.injectContextDataLbaas()
	args.injectContextGetLbaas()

	return &schema.Resource{
		ReadContext: dataSourceLbaasRead,
		Schema:      args,
	}
}

func dataSourceLbaasRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	target, err := checkDatasourceNameOrId(d)
	if err != nil {
		return diag.Errorf("[ERROR-030] crash via chose target : %s", err)
	}

	var lbaas *bcc.LoadBalancer
	if strings.EqualFold(target, "id") {
		lbaasId := d.Get("id").(string)
		lbaas, err = manager.GetLoadBalancer(lbaasId)
		if err != nil {
			return diag.Errorf("[ERROR-030] crash via getting Lbaas by id=%s: %s", lbaasId, err)
		}
	} else if strings.EqualFold(target, "name") {
		vdc, err := GetVdcById(d, manager)
		if err != nil {
			return diag.Errorf("[ERROR-030] crash via getting vdc: %s", err)
		}

		lbaas, err = GetLbaasByName(d, manager, vdc)
		if err != nil {
			return diag.Errorf("[ERROR-030] crash via getting Lbaas: %s", err)
		}
	} else {
		return diag.Errorf("[ERROR-030] for Lbaas must be specified id or name")
	}

	lbaasPort := make([]interface{}, 1)
	lbaasPort[0] = map[string]interface{}{
		"ip_address": lbaas.Port.IpAddress,
		"network_id": lbaas.Port.Network.ID,
	}

	fields := map[string]interface{}{
		"id":          lbaas.ID,
		"name":        lbaas.Name,
		"tags":        marshalTagNames(lbaas.Tags),
		"port":        lbaasPort,
		"vdc_id":      lbaas.Vdc.ID,
		"floating":    false,
		"floating_ip": "",
	}

	if lbaas.Floating != nil {
		fields["floating"] = true
		fields["floating_ip"] = lbaas.Floating.IpAddress
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-030] crash via set attrs: %s", err)
	}

	return nil
}
