package bcc_terraform

import (
	"context"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceAffinityGroup() *schema.Resource {
	args := Defaults()
	args.injectResultAffinityGroup()
	args.injectContextVdcByIdForData()
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
		return diag.Errorf("error getting target vdc: %s", err)
	}

	target, err := checkDatasourceNameOrId(d)
	if err != nil {
		return diag.Errorf("error getting affinity group: %s", err)
	}

	var targetAffinityGroup *bcc.AffinityGroup
	if target == "id" {
		id := d.Get("id").(string)
		targetAffinityGroup, err = manager.GetAffinityGroup(id)
		if err != nil {
			return diag.Errorf("error getting affinity group by id='%s': %s", id, err)
		}
	} else {
		targetAffinityGroup, err = GetAffinityGroupByName(d, manager, targetVdc)
		if err != nil {
			return diag.Errorf("error getting affinity group by name='%s': %s", d.Get("name").(string), err)
		}
	}

	vms := make([]map[string]interface{}, len(targetAffinityGroup.Vms))
	for i, vm := range targetAffinityGroup.Vms {
		vms[i] = map[string]interface{}{"id": vm.ID, "name": vm.Name}
	}

	fields := map[string]interface{}{
		"vdc_id":      targetAffinityGroup.Vdc.ID,
		"id":          targetAffinityGroup.ID,
		"name":        targetAffinityGroup.Name,
		"description": targetAffinityGroup.Description,
		"policy":      targetAffinityGroup.Policy,
		"reboot":      targetAffinityGroup.Reboot,
		"vms":         vms,
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-042]: crash via setting resource data: %s", err)
	}

	return nil

}
