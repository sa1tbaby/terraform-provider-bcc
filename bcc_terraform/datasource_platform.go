package bcc_terraform

import (
	"context"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourcePlatform() *schema.Resource {
	args := Defaults()
	args.injectContextRequiredVdc()
	args.injectResultPlatform()
	args.injectContextGetPlatform()

	return &schema.Resource{
		ReadContext: dataSourcePlatformRead,
		Schema:      args,
	}
}

func dataSourcePlatformRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	target, err := checkDatasourceNameOrId(d)
	if err != nil {
		return diag.Errorf("[ERROR-039] crash via getting Platform: %s", err)
	}

	var platform *bcc.Platform
	if target == "id" {
		platformId := d.Get("id").(string)
		platform, err = manager.GetPlatform(platformId)
		if err != nil {
			return diag.Errorf("[ERROR-039] crash via getting Platform by id=%s: %s", platformId, err)
		}

	} else {
		vdc, err := GetVdcById(d, manager)
		if err != nil {
			return diag.Errorf("[ERROR-039] crash via getting vdc: %s", err)
		}

		platform, err = GetPlatformByName(d, manager, vdc)
		if err != nil {
			return diag.Errorf("[ERROR-039] crash via getting Platform by namee: %s", err)
		}
	}

	fields := map[string]interface{}{
		"id":   platform.ID,
		"name": platform.Name,
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
