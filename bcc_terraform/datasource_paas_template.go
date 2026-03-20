package bcc_terraform

import (
	"context"
	"strings"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourcePaasTemplate() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourcePaasTemplateRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Paas Template identifier",
			},
			"vdc_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Vdc identifier",
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Paas Template name",
			},
		},
	}
}

func dataSourcePaasTemplateRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	manager = manager.WithContext(ctx)

	vdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-010] crash via getting vdc: %s", err)
	}

	err = ensureLocationCreated(vdc.ID, manager)
	if err != nil {
		return diag.FromErr(err)
	}

	target, err := checkDatasourceNameOrId(d)
	if err != nil {
		return diag.Errorf("[ERROR-041] crash via chose target: %s", err)
	}

	var paasTmp *bcc.PaasTemplate
	if strings.EqualFold(target, "id") {
		paasTmpId := d.Get("id").(string)
		paasTmp, err = manager.GetPaasTemplate(paasTmpId, vdc.ID)
		if err != nil {
			return diag.Errorf("Error getting paas template: %s", err)
		}
	} else {
		paasTmp, err = GetPaasTemplateByName(d, manager, vdc)
		if err != nil {
			return diag.Errorf("[ERROR-041] crash via getting paas template: %s", err)
		}
	}

	fields := map[string]interface{}{
		"id":     paasTmp.ID,
		"vdc_id": vdc.ID,
		"name":   paasTmp.Name,
	}
	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.FromErr(err)
	}
	return nil
}
