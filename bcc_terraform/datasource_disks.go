package bcc_terraform

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/hashstructure/v2"
)

func dataSourceDisks() *schema.Resource {
	args := Defaults()
	args.injectContextRequiredVdc()
	args.injectContextDataDiskList()

	return &schema.Resource{
		ReadContext: dataSourceDisksRead,
		Schema:      args,
	}
}

func dataSourceDisksRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	vdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-016] crash via getting vdc: %s", err)
	}

	diskList, err := vdc.GetDisks()
	if err != nil {
		return diag.Errorf("[ERROR-016] crash via retrieving disks: %s", err)
	}

	diskMap := make([]map[string]interface{}, len(diskList))
	for i, disk := range diskList {
		diskMap[i] = map[string]interface{}{
			"id":                   disk.ID,
			"name":                 disk.Name,
			"size":                 disk.Size,
			"storage_profile_id":   disk.StorageProfile.ID,
			"storage_profile_name": disk.StorageProfile.Name,
			"external_id":          disk.ExternalID,
			"tags":                 marshalTagNames(disk.Tags),
		}
	}

	hash, err := hashstructure.Hash(diskList, hashstructure.FormatV2, nil)
	if err != nil {
		diag.Errorf("[ERROR-016] crash via compute hash: %s", err)
	}

	fields := map[string]interface{}{
		"id":     fmt.Sprintf("disks/%d", hash),
		"vdc_id": vdc.ID,
		"disks":  diskMap,
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-016] crash via set attrs: %s", err)
	}

	return nil
}
