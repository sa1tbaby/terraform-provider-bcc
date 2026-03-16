package bcc_terraform

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceFirewallTemplate() *schema.Resource {
	args := Defaults()
	args.injectContextRequiredVdc()
	args.injectContextResourceFirewallTemplate()

	return &schema.Resource{
		CreateContext: resourceFirewallTemplateCreate,
		ReadContext:   resourceFirewallTemplateRead,
		UpdateContext: resourceFirewallTemplateUpdate,
		DeleteContext: resourceFirewallTemplateDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceFirewallTemplateImport,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
		Schema: args,
	}
}

func resourceFirewallTemplateCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	targetVdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-043]: crash via getting VDC by id: %s", err)
	}

	newFirewallTemplate := bcc.NewFirewallTemplate(d.Get("name").(string))
	newFirewallTemplate.Tags = unmarshalTagNames(d.Get("tags"))
	newFirewallTemplate.Description = d.Get("description").(string)

	err = targetVdc.CreateFirewallTemplate(&newFirewallTemplate)
	if err != nil {
		return diag.Errorf("[ERROR-043]: crash via creating Firewall Template: %s", err)
	}

	d.SetId(newFirewallTemplate.ID)
	log.Printf("[INFO]: firewallTemplate created, ID: %s", d.Id())

	return resourceFirewallTemplateRead(ctx, d, meta)
}

func resourceFirewallTemplateRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	firewallTemplate, err := manager.GetFirewallTemplate(d.Id())
	if err != nil {
		return resourceReadCheck(d, err, "[ERROR-043]:")
	}

	fields := map[string]interface{}{
		"name":        firewallTemplate.Name,
		"tags":        marshalTagNames(firewallTemplate.Tags),
		"description": firewallTemplate.Description,
		"rules_count": firewallTemplate.RulesCount,
		"vdc_id":      firewallTemplate.Vdc.ID,
	}

	if err = setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-043]: crash via reading Firewall Template: %s", err)
	}

	return nil
}

func resourceFirewallTemplateUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	firewallTemplate, err := manager.GetFirewallTemplate(d.Id())
	if err != nil {
		return diag.Errorf("[ERROR-043]: crash via getting FirewallTemplate by id: %s", err)
	}

	if d.HasChange("name") {
		firewallTemplate.Name = d.Get("name").(string)
	}
	if d.HasChange("tags") {
		firewallTemplate.Tags = unmarshalTagNames(d.Get("tags"))
	}
	if d.HasChange("description") {
		firewallTemplate.Description = d.Get("description").(string)
	}
	if err = firewallTemplate.UpdateFirewallTemplate(); err != nil {
		return diag.Errorf("[ERROR-043]: crash via update: %s", err)
	}

	return resourceFirewallTemplateRead(ctx, d, meta)
}

func resourceFirewallTemplateDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	FirewallTemplate, err := manager.GetFirewallTemplate(d.Id())
	if err != nil {
		return diag.Errorf("[ERROR-043]: crash via getting FirewallTemplate: %s", err)
	}

	err = FirewallTemplate.Delete()
	if err != nil {
		return diag.Errorf("[ERROR-043]: crash via deleting FirewallTemplate: %s", err)
	}

	return nil
}

func resourceFirewallTemplateImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	manager := meta.(*CombinedConfig).Manager()

	firewallTemplate, err := manager.GetFirewallTemplate(d.Id())
	if err != nil {
		return nil, fmt.Errorf("[ERROR-043]: crash via getting Firewall Template by id=%s: %s", d.Id(), err)
	}

	d.SetId(firewallTemplate.ID)

	return []*schema.ResourceData{d}, nil
}
