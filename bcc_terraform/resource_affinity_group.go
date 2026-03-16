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

func resourceAffinityGroup() *schema.Resource {
	args := Defaults()
	args.injectContextResourceAffinityGroup()
	args.injectContextRequiredVdc()

	return &schema.Resource{
		CreateContext: resourceAffinityGroupCreate,
		ReadContext:   resourceAffinityGroupRead,
		UpdateContext: resourceAffinityGroupUpdate,
		DeleteContext: resourceAffinityGroupDelete,

		Importer: &schema.ResourceImporter{
			StateContext: resourceAffinityGroupImport,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: args,
	}
}

func resourceAffinityGroupCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	vdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-042]: crash via getting VDC: %s", err)
	}

	fields := struct {
		Name        string
		Description string
		Policy      string
	}{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		Policy:      d.Get("policy").(string),
	}

	affGroup := bcc.NewAffinityGroup(fields.Name, fields.Description, fields.Policy, nil)

	if err := vdc.CreateAffinityGroup(&affGroup); err != nil {
		return diag.Errorf("[ERROR-042]: crash via creating AffinityGroup: %s", err)
	}
	if err = affGroup.WaitLock(); err != nil {
		return diag.Errorf("[ERROR-042]: %s", err)
	}

	d.SetId(affGroup.ID)
	log.Printf("[INFO] AffGroup created, ID: %s", d.Id())

	return resourceAffinityGroupRead(ctx, d, meta)
}

func resourceAffinityGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	affGroup, err := manager.GetAffinityGroup(d.Id())
	if err != nil {
		return resourceReadCheck(d, err, "[ERROR-042]:")
	}

	vms := make([]map[string]interface{}, len(affGroup.Vms))
	for i, vm := range affGroup.Vms {
		vms[i] = map[string]interface{}{"id": vm.ID, "name": vm.Name}
	}

	fields := map[string]interface{}{
		"vdc_id":      affGroup.Vdc.ID,
		"name":        affGroup.Name,
		"description": affGroup.Description,
		"policy":      affGroup.Policy,
		"vms":         vms,
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-042]: crash via set attrs: %s", err)
	}

	return nil
}

func resourceAffinityGroupUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	needUpdate := false

	affGroup, err := manager.GetAffinityGroup(d.Id())
	if err != nil {
		return diag.Errorf("[ERROR-042]: crash via getting AffinityGroup: %s", err)
	}

	if d.HasChange("name") {
		needUpdate = true
		affGroup.Name = d.Get("name").(string)
	}

	if d.HasChange("description") {
		needUpdate = true
		affGroup.Description = d.Get("description").(string)
	}

	if needUpdate {
		if err := repeatOnError(affGroup.Update, affGroup); err != nil {
			return diag.Errorf("[ERROR-042]: crash via updating AffinityGroup: %s", err)
		}
	}

	return resourceAffinityGroupRead(ctx, d, meta)
}

func resourceAffinityGroupDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	affGroup, err := manager.GetAffinityGroup(d.Id())
	if err != nil {
		return diag.Errorf("[ERROR-042]: crash via getting AffinityGroup: %s", err)
	}

	if err = affGroup.Delete(); err != nil {
		return diag.Errorf("[ERROR-042]: crash via deleting vm: %s", err)
	}
	affGroup.WaitLock()

	return nil
}

func resourceAffinityGroupImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	manager := meta.(*CombinedConfig).Manager()

	affGroup, err := manager.GetAffinityGroup(d.Id())
	if err != nil {
		return nil, fmt.Errorf("[ERROR-042]: crash via getting AffinityGroup: %s", err)
	}

	d.SetId(affGroup.ID)

	return []*schema.ResourceData{d}, nil

}
