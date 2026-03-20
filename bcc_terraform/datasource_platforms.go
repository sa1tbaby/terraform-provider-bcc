package bcc_terraform

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/hashstructure/v2"
)

func dataSourcePlatforms() *schema.Resource {
	args := Defaults()
	args.injectResultListPlatforms()
	args.injectContextRequiredVdc()

	return &schema.Resource{
		ReadContext: dataSourcePlatformsRead,
		Schema:      args,
	}
}

func dataSourcePlatformsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	vdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-040] crash via getting vdc: %s", err)
	}

	platformList, err := manager.GetPlatforms(vdc.ID)
	if err != nil {
		return diag.Errorf("[ERROR-040] crash via retrieving platforms: %s", err)
	}

	platformMap := make([]map[string]interface{}, len(platformList))
	for i, platform := range platformList {
		platformMap[i] = map[string]interface{}{
			"id":   platform.ID,
			"name": platform.Name,
		}
	}

	hash, err := hashstructure.Hash(platformList, hashstructure.FormatV2, nil)
	if err != nil {
		diag.Errorf("[ERROR-040] crash via calculate hash: %s", err)
	}

	fields := map[string]interface{}{
		"id":        fmt.Sprintf("platforms/%d", hash),
		"vdc_id":    vdc.ID,
		"platforms": platformMap,
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-040] crash via set attrs: %s", err)
	}

	return nil
}
