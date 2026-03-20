package bcc_terraform

import (
	"context"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceTemplate() *schema.Resource {
	args := Defaults()
	args.injectResultTemplate()
	args.injectContextRequiredVdc()
	args.injectContextGetTemplate()

	return &schema.Resource{
		ReadContext: dataSourceTemplateRead,
		Schema:      args,
	}
}

func dataSourceTemplateRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	vdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-017] crash via getting vdc: %s", err)
	}

	target, err := checkDatasourceNameOrId(d)
	if err != nil {
		return diag.Errorf("[ERROR-017] crash via chose target: %s", err)
	}

	var template *bcc.Template
	if target == "id" {
		templateId := d.Get("id").(string)
		template, err = manager.GetTemplate(templateId)
		if err != nil {
			return diag.Errorf("[ERROR-017] crash via getting template by id: %s", err)
		}
	} else {
		template, err = GetTemplateByName(d, manager, vdc)
		if err != nil {
			return diag.Errorf("[ERROR-017] crash via getting template by name: %s", err)
		}
	}

	fields := map[string]interface{}{
		"id":       template.ID,
		"name":     template.Name,
		"min_cpu":  template.MinCpu,
		"min_ram":  template.MinRam,
		"min_disk": template.MinHdd,
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-017] crash via set attrs: %s", err)
	}

	return nil
}
