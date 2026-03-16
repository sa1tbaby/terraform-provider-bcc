package bcc_terraform

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/hashstructure/v2"
)

func dataSourceLoadBalancers() *schema.Resource {
	args := Defaults()
	args.injectContextRequiredVdc()
	args.injectContextDataLbaasList()

	return &schema.Resource{
		ReadContext: dataSourceLoadBalancersRead,
		Schema:      args,
	}
}

func dataSourceLoadBalancersRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	vdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-031] crash via getting vdc: %s", err)
	}

	lbaasList, err := vdc.GetLoadBalancers()
	if err != nil {
		return diag.Errorf("[ERROR-031] crash via retrieving lbs: %s", err)
	}

	lbaasMap := make([]map[string]interface{}, len(lbaasList))
	for i, lb := range lbaasList {
		lbaasPort := make([]interface{}, 1)
		lbaasPort[0] = map[string]interface{}{
			"ip_address": lb.Port.IpAddress,
			"network_id": lb.Port.Network.ID,
		}

		lbaasMap[i] = map[string]interface{}{
			"id":          lb.ID,
			"name":        lb.Name,
			"port":        lbaasPort,
			"tags":        marshalTagNames(lb.Tags),
			"floating":    false,
			"floating_ip": "",
		}

		if lb.Floating != nil {
			lbaasMap[i]["floating"] = true
			lbaasMap[i]["floating_ip"] = lb.Floating.IpAddress
		}
	}

	hash, err := hashstructure.Hash(lbaasList, hashstructure.FormatV2, nil)
	if err != nil {
		diag.Errorf("[ERROR-031] crash via calculating hash: %s", err)
	}

	fields := map[string]interface{}{
		"id":     fmt.Sprintf("lbs/%d", hash),
		"vdc_id": vdc.ID,
		"lbaass": lbaasMap,
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-031] crash via set attrs: %s", err)
	}

	return nil
}
