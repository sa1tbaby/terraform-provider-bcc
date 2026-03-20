package bcc_terraform

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/hashstructure/v2"
)

func dataSourceTemplates() *schema.Resource {
	args := Defaults()
	args.injectContextRequiredVdc()
	args.injectResultListTemplate()

	return &schema.Resource{
		ReadContext: dataSourceTemplatesRead,
		Schema:      args,
	}
}

func dataSourceTemplatesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	vdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-018] crash via getting vdc: %s", err)
	}

	templateList, err := vdc.GetTemplates()
	if err != nil {
		return diag.Errorf("[ERROR-018] crash via retrieving templates: %s", err)
	}

	templateMap := make([]map[string]interface{}, len(templateList))
	for i, template := range templateList {
		templateMap[i] = map[string]interface{}{
			"id":       template.ID,
			"name":     template.Name,
			"min_cpu":  template.MinCpu,
			"min_ram":  template.MinRam,
			"min_disk": template.MinHdd,
		}
	}

	hash, err := hashstructure.Hash(templateList, hashstructure.FormatV2, nil)
	if err != nil {
		diag.Errorf("[ERROR-018] crash via calculate hash: %s", err)
	}

	fields := map[string]interface{}{
		"id":        fmt.Sprintf("templates/%d", hash),
		"vdc_id":    vdc.ID,
		"templates": templateMap,
	}
	if err := setResourceDataFromMap(d, fields); err != nil {
		diag.Errorf("[ERROR-018] crash via set attrs: %s", err)
	}

	return nil
}
