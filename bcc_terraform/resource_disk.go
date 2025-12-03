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

func resourceDisk() *schema.Resource {
	args := Defaults()
	args.injectCreateDisk()
	args.injectContextVdcById()

	return &schema.Resource{
		CreateContext: resourceDiskCreate,
		ReadContext:   resourceDiskRead,
		UpdateContext: resourceDiskUpdate,
		DeleteContext: resourceDiskDelete,
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

func resourceDiskCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	config := &struct {
		id               int
		name             string
		size             int
		storageProfileId string
		externalId       string
		vdcId            string
		tags             interface{}
	}{
		id:               d.Get("id").(int),
		name:             d.Get("name").(string),
		size:             d.Get("size").(int),
		vdcId:            d.Get("vdc_id").(string),
		storageProfileId: d.Get("storage_profile_id").(string),
		externalId:       d.Get("external_id").(string),
		tags:             d.Get("tags"),
	}

	targetVdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-014]: %s", err)
	}

	targetStorageProfile, err := GetStorageProfileById(config.storageProfileId, manager, targetVdc)
	if err != nil {
		return diag.Errorf("[ERROR-014]: %s", err)
	}

	newDisk := bcc.NewDisk(config.name, config.size, targetStorageProfile)
	newDisk.Tags = unmarshalTagNames(config.tags)

	if err = targetVdc.WaitLock(); err != nil {
		return diag.Errorf("[ERROR-014]: %s", err)
	}
	if err = targetVdc.CreateDisk(&newDisk); err != nil {
		return diag.Errorf("[ERROR-014]: %s", err)
	}
	if err = newDisk.WaitLock(); err != nil {
		return diag.Errorf("[ERROR-014]: %s", err)
	}

	d.SetId(newDisk.ID)
	log.Printf("[INFO] Disk created, ID: %s", d.Id())

	return resourceDiskRead(ctx, d, meta)
}

func resourceDiskRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	disk, err := manager.GetDisk(d.Id())
	if err != nil {
		if err.(*bcc.ApiError).Code() == 404 {
			d.SetId("")
			return nil
		} else {
			return diag.Errorf("id: Error getting disk: %s", err)
		}
	}

	fields := map[string]interface{}{
		"vdc_id":             disk.Vdc.ID,
		"size":               disk.Size,
		"name":               disk.Name,
		"storage_profile_id": disk.StorageProfile.ID,
		"external_id":        disk.ExternalID,
		"tags":               marshalTagNames(disk.Tags),
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-014]: crash via set data for 'Disk': %s", err)
	}

	return nil
}

func resourceDiskUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	disk, err := manager.GetDisk(d.Id())
	if err != nil {
		return diag.Errorf("id: Error getting disk: %s", err)
	}

	needUpdate := false
	if d.HasChange("name") {
		disk.Name = d.Get("name").(string)
		needUpdate = true
	}
	if d.HasChange("tags") {
		disk.Tags = unmarshalTagNames(d.Get("tags"))
		needUpdate = true
	}
	if d.HasChange("size") {
		disk.Size = d.Get("size").(int)
		if disk.Locked {
			disk.WaitLock()
		}
		err = disk.Resize(d.Get("size").(int))
		if err != nil {
			return diag.Errorf("size: Error resizing disk: %s", err)
		}
		if err = disk.Resize(disk.Size); err != nil {
			return diag.Errorf("[ERROR-014]: %s", err)
		}
		needUpdate = false
	}
	if d.HasChange("storage_profile_id") {
		targetVdc, err := GetVdcById(d, manager)
		if err != nil {
			return diag.Errorf("Error getting VDC: %s", err)
		}

		targetStorageProfileId := d.Get("storage_profile_id").(string)
		targetStorageProfile, err := GetStorageProfileById(targetStorageProfileId, manager, targetVdc)
		if err != nil {
			return diag.Errorf("storage_profile: Error getting storage profile: %s", err)
		}
		if disk.Locked {
			disk.WaitLock()
		}
		err = disk.UpdateStorageProfile(*targetStorageProfile)
		if err != nil {
			return diag.Errorf("storage_profile: Error updating storage: %s", err)
		}
		needUpdate = false
	}
	if needUpdate {
		if disk.Locked {
			disk.WaitLock()
		}
		disk.Update()
	}

	return resourceDiskRead(ctx, d, meta)
}

func resourceDiskDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	disk, err := manager.GetDisk(d.Id())
	if err != nil {
		return diag.Errorf("id: Error getting disk: %s", err)
	}

	if disk.Vm != nil {
		vm, err := manager.GetVm(disk.Vm.ID)
		if err != nil {
			return diag.FromErr(err)
		}
		err = vm.DetachDisk(disk)
		if err != nil {
			return diag.FromErr(err)
		}
	}
	err = disk.Delete()
	if err != nil {
		return diag.Errorf("Error deleting disk: %s", err)
	}
	disk.WaitLock()

	return nil
}
