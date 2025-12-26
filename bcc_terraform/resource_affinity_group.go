package bcc_terraform

import (
	"context"
	"fmt"
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
		return diag.Errorf("[ERROR-042]: crash via getting VDC: %s", err)
	}

	config := struct {
		VdcId       string
		Name        string
		Description string
		Policy      string
		Reboot      bool
		vms         []interface{}
	}{
		VdcId:       d.Get("vdc_id").(string),
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		Policy:      d.Get("policy").(string),
		Reboot:      d.Get("reboot").(bool),
		vms:         d.Get("vms").([]interface{}),
	}

	newAffGp := bcc.NewAffinityGroup(config.Name, config.Description, config.Policy, nil)
	newAffGp.Reboot = config.Reboot

	for _, item := range config.vms {
		_item := item.(map[string]interface{})
		vm := bcc.MetaData{ID: _item["id"].(string)}
		newAffGp.Vms = append(newAffGp.Vms, &vm)
	}

	if err := targetVdc.CreateAffinityGroup(&newAffGp); err != nil {
		return diag.Errorf("[ERROR-042]: crash via creating AffinityGroup: %s", err)
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
			return diag.Errorf("[ERROR-042]: crash via getting AffinityGroup: %s", err)
		}
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
		"reboot":      affGroup.Reboot,
		"vms":         vms,
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

	if d.HasChange("reboot") {
		needUpdate = true
		affGroup.Reboot = d.Get("reboot").(bool)
	}

	if d.HasChange("vms") {
		needUpdate = true
		raw_vms := d.Get("vms").([]interface{})
		vms := make([]*bcc.MetaData, len(raw_vms))

		for i, item := range raw_vms {
			_item := item.(map[string]interface{})
			vm := bcc.MetaData{ID: _item["id"].(string)}
			vms[i] = &vm
		}
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
