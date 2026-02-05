package bcc_terraform

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/hashstructure/v2"
)

func dataSourceAffinityGroups() *schema.Resource {
	args := Defaults()
	args.injectContextVdcById()
	args.injectResultListAffinityGroup()

	return &schema.Resource{
		ReadContext: dataSourceAffinityGroupsRead,
		Schema:      args,
	}
}

func dataSourceAffinityGroupsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	targetVdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("error getting target vdc: %s", err)
	}

	allAffinityGroups, err := targetVdc.GetAffinityGroups()
	if err != nil {
		return diag.Errorf("error getting all affinity groups: %s", err)
	}

	fieldsMap := make([]map[string]interface{}, len(allAffinityGroups))
	for i, affinityGroup := range allAffinityGroups {

		vms := make([]map[string]interface{}, len(affinityGroup.Vms))
		for i, vm := range affinityGroup.Vms {
			vms[i] = map[string]interface{}{"id": vm.ID, "name": vm.Name}
		}

		fieldsMap[i] = map[string]interface{}{
			"id":          affinityGroup.ID,
			"name":        affinityGroup.Name,
			"description": affinityGroup.Description,
			"policy":      affinityGroup.Policy,
			"reboot":      affinityGroup.Reboot,
			"vms":         vms,
		}
	}

	hash, err := hashstructure.Hash(allAffinityGroups, hashstructure.FormatV2, nil)
	if err != nil {
		diag.Errorf("error computing hash of all affinity groups: %s", err)
	}

	fields := map[string]interface{}{
		"id":              fmt.Sprintf("affinity_groups/%d", hash),
		"vdc_id":          targetVdc.ID,
		"affinity_groups": fieldsMap,
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("crash via setting data: %s", err)
	}

	return nil
}
