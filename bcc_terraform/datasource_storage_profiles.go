package bcc_terraform

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/hashstructure/v2"
)

func dataSourceStorageProfiles() *schema.Resource {
	args := Defaults()
	args.injectContextRequiredVdc()
	args.injectResultListStorageProfile()

	return &schema.Resource{
		ReadContext: dataSourceStorageProfilesRead,
		Schema:      args,
	}
}

func dataSourceStorageProfilesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	vdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-013] crash via getting VDC: %s", err)
	}

	storageProfileList, err := vdc.GetStorageProfiles()
	if err != nil {
		return diag.Errorf("[ERROR-013] crash via getting storage profiles")
	}

	storageProfileMap := make([]map[string]interface{}, len(storageProfileList))
	for i, storageProfile := range storageProfileList {
		storageProfileMap[i] = map[string]interface{}{
			"id":            storageProfile.ID,
			"name":          storageProfile.Name,
			"max_disk_size": storageProfile.MaxDiskSize,
			"enabled":       storageProfile.Enabled,
		}
	}

	hash, err := hashstructure.Hash(storageProfileList, hashstructure.FormatV2, nil)
	if err != nil {
		diag.Errorf("[ERROR-013] crash via calculate hash: %s", err)
	}

	fields := map[string]interface{}{
		"id":               fmt.Sprintf("storage_profiles/%d", hash),
		"vdc_id":           vdc.ID,
		"storage_profiles": storageProfileMap,
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-013] crash via set attrs: %s", err)
	}

	return nil
}
