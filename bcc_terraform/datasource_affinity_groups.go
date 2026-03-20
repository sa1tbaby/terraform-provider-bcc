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
	args.injectContextRequiredVdc()
	args.injectContextDataListAffinityGroup()

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

	affGroupList, err := targetVdc.GetAffinityGroups()
	if err != nil {
		return diag.Errorf("[ERROR-056] crash getting all affinity groups: %s", err)
	}

	affGroupsMap := make([]map[string]interface{}, len(affGroupList))
	for i, affinityGroup := range affGroupList {

		vms := make([]map[string]interface{}, len(affinityGroup.Vms))
		for i, vm := range affinityGroup.Vms {
			vms[i] = map[string]interface{}{"id": vm.ID, "name": vm.Name}
		}

		affGroupsMap[i] = map[string]interface{}{
			"id":          affinityGroup.ID,
			"name":        affinityGroup.Name,
			"description": affinityGroup.Description,
			"policy":      affinityGroup.Policy,
			"vms":         vms,
		}
	}

	hash, err := hashstructure.Hash(affGroupList, hashstructure.FormatV2, nil)
	if err != nil {
		diag.Errorf("[ERROR-056] crash via computing hash: %s", err)
	}

	fields := map[string]interface{}{
		"id":              fmt.Sprintf("affinity_groups/%d", hash),
		"vdc_id":          targetVdc.ID,
		"affinity_groups": affGroupsMap,
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-056] crash via setting data: %s", err)
	}

	return nil
}
