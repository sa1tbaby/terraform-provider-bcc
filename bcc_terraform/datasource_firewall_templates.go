package bcc_terraform

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/hashstructure/v2"
)

func dataSourceFirewallTemplates() *schema.Resource {
	args := Defaults()
	args.injectContextRequiredVdc()
	args.injectContextDataFirewallTemplateList()

	return &schema.Resource{
		ReadContext: dataSourceFirewallTemplatesRead,
		Schema:      args,
	}
}

func dataSourceFirewallTemplatesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	vdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-020] crash via getting vdc: %s", err)
	}

	fwTmpList, err := vdc.GetFirewallTemplates()
	if err != nil {
		return diag.Errorf("[ERROR-020] crash via retrieving firewall templates: %s", err)
	}

	var fwTmpMap []map[string]interface{}
	for _, ft := range fwTmpList {
		fields := map[string]interface{}{
			"id":          ft.ID,
			"name":        ft.Name,
			"description": ft.Description,
			"rules_count": ft.RulesCount,
			"tags":        marshalTagNames(ft.Tags),
		}
		fwTmpMap = append(fwTmpMap, fields)
	}

	hash, err := hashstructure.Hash(fwTmpList, hashstructure.FormatV2, nil)
	if err != nil {
		diag.Errorf("[ERROR-020] crash via calculate hash: %s", err)
	}

	fields := map[string]interface{}{
		"id":                 fmt.Sprintf("firewall_templates/%d", hash),
		"vdc_id":             vdc.ID,
		"firewall_templates": fwTmpMap,
	}
	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-020] crash via datasource `firewall_templates`: %s", err)
	}

	return nil
}
