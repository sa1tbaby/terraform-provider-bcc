package bcc_terraform

import (
	"context"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceStorageProfile() *schema.Resource {
	args := Defaults()
	args.injectContextRequiredVdc()
	args.injectResultStorageProfile()
	args.injectContextGetStorageProfile()

	return &schema.Resource{
		ReadContext: dataSourceStorageProfileRead,
		Schema:      args,
	}
}

func dataSourceStorageProfileRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	vdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-012] crash via getting vdc: %s", err)
	}

	target, err := checkDatasourceNameOrId(d)
	if err != nil {
		return diag.Errorf("[ERROR-012] crash via getting chose target: %s", err)
	}

	var storageProfile *bcc.StorageProfile
	if target == "id" {
		storageProfileId := d.Get("id").(string)
		storageProfile, err = vdc.GetStorageProfile(storageProfileId)
		if err != nil {
			return diag.Errorf("[ERROR-012] crash via getting storage profile by id: %s", err)
		}
	} else {
		storageProfile, err = GetStorageProfileByName(d, manager, vdc)
		if err != nil {
			return diag.Errorf("[ERROR-012] crash via getting storage profile by name: %s", err)
		}
	}

	fields := map[string]interface{}{
		"id":            storageProfile.ID,
		"name":          storageProfile.Name,
		"max_disk_size": storageProfile.MaxDiskSize,
		"enabled":       storageProfile.Enabled,
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-012] crash via set attrs: %s", err)
	}

	return nil
}
