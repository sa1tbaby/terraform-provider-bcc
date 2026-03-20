package bcc_terraform

import (
	"context"
	"strings"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceFirewallTemplate() *schema.Resource {
	args := Defaults()
	args.injectContextRequiredVdc()
	args.injectContextDataFirewallTemplate()
	args.injectContextGetFirewallTemplate()

	return &schema.Resource{
		ReadContext: dataSourceFirewallTemplateRead,
		Schema:      args,
	}
}

func dataSourceFirewallTemplateRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	target, err := checkDatasourceNameOrId(d)
	if err != nil {
		return diag.Errorf("[ERROR-019] crash via chose target: %s", err)
	}

	var firewallTemplate *bcc.FirewallTemplate
	if strings.EqualFold(target, "id") {
		fwTmplId := d.Get("id").(string)
		firewallTemplate, err = manager.GetFirewallTemplate(fwTmplId)
		if err != nil {
			return diag.Errorf("[ERROR-019] crash via getting template by id=%s: %s", fwTmplId, err)
		}
	} else {
		vdc, err := GetVdcById(d, manager)
		if err != nil {
			return diag.Errorf("[ERROR-019] crash via getting vdc: %s", err)
		}

		firewallTemplate, err = GetFirewallTemplateByName(d, manager, vdc)
		if err != nil {
			return diag.Errorf("[ERROR-019] crash via getting template by name: %s", err)
		}
	}

	fields := map[string]interface{}{
		"id":          firewallTemplate.ID,
		"name":        firewallTemplate.Name,
		"description": firewallTemplate.Description,
		"rules_count": firewallTemplate.RulesCount,
		"tags":        marshalTagNames(firewallTemplate.Tags),
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-019] crash via set attrs: %s", err)
	}
	return nil
}
