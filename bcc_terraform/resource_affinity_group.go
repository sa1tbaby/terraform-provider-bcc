package bcc_terraform

import (
	"context"
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
			StateContext: schema.ImportStatePassthroughContext,
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

	name := d.Get("name").(string)
	description := d.Get("description").(string)
	policy := d.Get("policy").(string)
	reboot := d.Get("reboot").(bool)
	log.Printf(name, description, policy, reboot)

	var vms []*bcc.MetaData
	for _, vm := range d.Get("vms").([]interface{}) {
		vmMap := vm.(map[string]interface{})
		vms = append(vms, &bcc.MetaData{ID: vmMap["id"].(string), Name: vmMap["name"].(string)})
	}

	newAffGp := bcc.NewAffinityGroup(name, description, policy, vms)
	newAffGp.Reboot = reboot
	if err := targetVdc.CreateAffinityGroup(&newAffGp); err != nil {
		return diag.Errorf("Error creating AffinityGroup: %s", err)
	}

	if err = newAffGp.WaitLock(); err != nil {
		return diag.FromErr(err)
	}

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

	d.SetId(affGroup.ID)
	d.Set("name", affGroup.Name)
	d.Set("description", affGroup.Description)
	d.Set("policy", affGroup.Policy)
	d.Set("reboot", affGroup.Reboot)
	d.Set("vms", affGroup.Vms)

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

		var vms []*bcc.MetaData
		for _, _vm := range d.Get("vms").([]interface{}) {
			vmMap := _vm.(map[string]interface{})
			vms = append(vms, &bcc.MetaData{ID: vmMap["id"].(string), Name: vmMap["name"].(string)})
		}

		affGroup.Vms = vms
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
