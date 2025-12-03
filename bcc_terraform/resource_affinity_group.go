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
	args.injectCreateAffinityGroup()
	args.injectContextVdcById()

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

	targetVdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("vdc_id: Error getting VDC: %s", err)
	}

	config := struct {
		VdcId       string
		Name        string
		Description string
		Policy      string
		Reboot      bool
		vms         []*bcc.MetaData
	}{
		VdcId:       d.Get("vdc_id").(string),
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		Policy:      d.Get("policy").(string),
		Reboot:      d.Get("reboot").(bool),
		vms:         d.Get("vms").([]*bcc.MetaData),
	}

	newAffGp := bcc.NewAffinityGroup(config.Name, config.Description, config.Policy, config.vms)
	newAffGp.Reboot = config.Reboot

	if err := targetVdc.CreateAffinityGroup(&newAffGp); err != nil {
		return diag.Errorf("Error creating AffinityGroup: %s", err)
	}

	if err = newAffGp.WaitLock(); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(newAffGp.ID)

	return resourceAffinityGroupRead(ctx, d, meta)
}

func resourceAffinityGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	affGroup, err := manager.GetAffinityGroup(d.Id())
	if err != nil {
		if err.(*bcc.ApiError).Code() == 404 {
			d.SetId("")
			return nil
		} else {
			return diag.Errorf("Error getting affinity group: %s", err)
		}
	}

	fields := map[string]interface{}{
		"vdc_id":      affGroup.Vdc.ID,
		"name":        affGroup.Name,
		"description": affGroup.Description,
		"policy":      affGroup.Policy,
		"reboot":      affGroup.Reboot,
		"vms":         affGroup.Vms,
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-042]: crash via setting resource data: %s", err)
	}

	return nil
}

func resourceAffinityGroupUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	needUpdate := false

	affGroup, err := manager.GetAffinityGroup(d.Id())
	if err != nil {
		if err.(*bcc.ApiError).Code() == 404 {
			d.SetId("")
			return nil
		} else {
			return diag.Errorf("Error getting affinity group: %s", err)
		}
	}

	if d.HasChange("name") {
		needUpdate = true
		affGroup.Name = d.Get("name").(string)
	}

	if d.HasChange("description") {
		needUpdate = true
		affGroup.Description = d.Get("description").(string)
	}

	if d.HasChange("reboot") {
		needUpdate = true
		affGroup.Reboot = d.Get("reboot").(bool)
	}

	if d.HasChange("vms") {
		needUpdate = true
		affGroup.Vms = d.Get("vms").([]*bcc.MetaData)
	}

	if needUpdate {
		if err := repeatOnError(affGroup.Update, affGroup); err != nil {
			return diag.Errorf("Error updating Affinity Group: %s", err)
		}
	}

	return resourceAffinityGroupRead(ctx, d, meta)
}

func resourceAffinityGroupDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	affGroup, err := manager.GetAffinityGroup(d.Id())
	if err != nil {
		if err.(*bcc.ApiError).Code() == 404 {
			d.SetId("")
			return nil
		} else {
			return diag.Errorf("Error getting affinity group: %s", err)
		}
	}

	if err = affGroup.Delete(); err != nil {
		return diag.Errorf("Error deleting vm: %s", err)
	}

	if err = affGroup.WaitLock(); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceAffinityGroupImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	manager := meta.(*CombinedConfig).Manager()

	affGroup, err := manager.GetAffinityGroup(d.Id())
	if err != nil {
		return nil, fmt.Errorf("error getting affinity group: %s", err)
	}

	d.SetId(affGroup.ID)

	return []*schema.ResourceData{d}, nil

}
