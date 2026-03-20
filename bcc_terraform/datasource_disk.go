package bcc_terraform

import (
	"context"
	"strings"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDisk() *schema.Resource {
	args := Defaults()
	args.injectContextRequiredVdc()
	args.injectContextDataDisk()
	args.injectContextGetDisk()

	return &schema.Resource{
		ReadContext: dataSourceDiskRead,
		Schema:      args,
	}
}

func dataSourceDiskRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	target, err := checkDatasourceNameOrId(d)
	if err != nil {
		return diag.Errorf("[ERROR-015] crash getting disk: %s", err)
	}

	var disk *bcc.Disk
	if strings.EqualFold(target, "id") {
		diskId := d.Get("id").(string)
		disk, err = manager.GetDisk(diskId)
		if err != nil {
			return diag.Errorf("[ERROR-015] crash via getting disk by id=%s: %s", diskId, err)
		}

	} else {
		targetVdc, err := GetVdcById(d, manager)
		if err != nil {
			return diag.Errorf("[ERROR-015] crash via getting vdc: %s", err)
		}

		disk, err = GetDiskByName(d, manager, targetVdc)
		if err != nil {
			return diag.Errorf("[ERROR-015] crash via getting disk by name: %s", err)
		}
	}

	fields := map[string]interface{}{
		"id":                   disk.ID,
		"name":                 disk.Name,
		"size":                 disk.Size,
		"storage_profile_id":   disk.StorageProfile.ID,
		"storage_profile_name": disk.StorageProfile.Name,
		"external_id":          disk.ExternalID,
		"tags":                 marshalTagNames(disk.Tags),
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
