package bcc_terraform

import (
	"context"
	"strings"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceAffinityGroup() *schema.Resource {
	args := Defaults()
	args.injectContextDataAffinityGroup()
	args.injectContextRequiredVdcForData()
	args.injectContextGetAffinityGroup()

	return &schema.Resource{
		ReadContext: dataSourceAffinityGroupRead,
		Schema:      args,
	}
}

func dataSourceAffinityGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	targetVdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-055] crash via getting vdc by id: %s", err)
	}

	target, err := checkDatasourceNameOrId(d)
	if err != nil {
		return diag.Errorf("[ERROR-055]: %s", err)
	}

	var affinityGroup *bcc.AffinityGroup
	if strings.EqualFold(target, "id") {
		id := d.Get("id").(string)
		affinityGroup, err = manager.GetAffinityGroup(id)
		if err != nil {
			return diag.Errorf("[ERROR-055] crash via getting affinity group by id='%s': %s", id, err)
		}
	} else {
		affinityGroup, err = GetAffinityGroupByName(d, manager, targetVdc)
		if err != nil {
			return diag.Errorf("[ERROR-055] crash via getting affinity group by name='%s': %s", d.Get("name").(string), err)
		}
	}

	vms := make([]map[string]interface{}, len(affinityGroup.Vms))
	for i, vm := range affinityGroup.Vms {
		vms[i] = map[string]interface{}{"id": vm.ID, "name": vm.Name}
	}

	fields := map[string]interface{}{
		"vdc_id":      affinityGroup.Vdc.ID,
		"id":          affinityGroup.ID,
		"name":        affinityGroup.Name,
		"description": affinityGroup.Description,
		"policy":      affinityGroup.Policy,
		"vms":         vms,
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-055]: crash via setting resource data: %s", err)
	}

	return nil

}
