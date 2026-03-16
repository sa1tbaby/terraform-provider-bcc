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
	args.injectContextResourceDisk()
	args.injectContextRequiredVdc()

	return &schema.Resource{
		CreateContext: resourceDiskCreate,
		ReadContext:   resourceDiskRead,
		UpdateContext: resourceDiskUpdate,
		DeleteContext: resourceDiskDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceDiskImport,
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

	config := struct {
		name             string
		size             int
		storageProfileId string
	}{
		name:             d.Get("name").(string),
		size:             d.Get("size").(int),
		storageProfileId: d.Get("storage_profile_id").(string),
	}

	vdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-014] crash via get vdc: %s", err)
	}

	storageProfile, err := GetStorageProfileById(config.storageProfileId, manager, vdc)
	if err != nil {
		return diag.Errorf("[ERROR-014] crash via get storage profile: %s", err)
	}

	disk := bcc.NewDisk(config.name, config.size, storageProfile)
	disk.Tags = unmarshalTagNames(d.Get("tags"))

	if err = vdc.WaitLock(); err != nil {
		return diag.Errorf("[ERROR-014] crash via vdc waitlock: %s", err)
	}
	if err = vdc.CreateDisk(&disk); err != nil {
		return diag.Errorf("[ERROR-014] crash via disk create: %s", err)
	}
	if err = disk.WaitLock(); err != nil {
		return diag.Errorf("[ERROR-014] crash via disk waitlock: %s", err)
	}

	d.SetId(disk.ID)
	log.Printf("[INFO] Disk created, ID: %s", d.Id())

	return resourceDiskRead(ctx, d, meta)
}

func resourceDiskRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	disk, err := manager.GetDisk(d.Id())
	if err != nil {
		return resourceReadCheck(d, err, "[ERROR-014]:")
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
		return diag.Errorf("[ERROR-014]: crash via set attrs: %s", err)
	}

	return nil
}

func resourceDiskUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	disk, err := manager.GetDisk(d.Id())
	if err != nil {
		return diag.Errorf("[ERROR-014] crash via get disk: %s", err)
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
			if err = disk.WaitLock(); err != nil {
				return diag.Errorf("[ERROR-014] crash via waitlock: %s", err)
			}
		}
		if err = disk.Resize(disk.Size); err != nil {
			return diag.Errorf("[ERROR-014] crash via resize: %s", err)
		}
		needUpdate = false
	}
	if d.HasChange("storage_profile_id") {
		vdc, err := GetVdcById(d, manager)
		if err != nil {
			return diag.Errorf("[ERROR-014] crash via getting vdc: %s", err)
		}

		storageProfileId := d.Get("storage_profile_id").(string)
		storageProfile, err := GetStorageProfileById(storageProfileId, manager, vdc)
		if err != nil {
			return diag.Errorf("[ERROR-014] crash via get storage profile: %s", err)
		}
		if disk.Locked {
			if err = disk.WaitLock(); err != nil {
				return diag.Errorf("[ERROR-014] crash via waitlock: %s", err)
			}
		}
		err = disk.UpdateStorageProfile(*storageProfile)
		if err != nil {
			return diag.Errorf("[ERROR-014] crash via update storage profile: %s", err)
		}
		needUpdate = false
	}
	if needUpdate {
		if disk.Locked {
			if err = disk.WaitLock(); err != nil {
				return diag.Errorf("[ERROR-014] crash via disk waitlock: %s", err)
			}
		}
		if err = disk.Update(); err != nil {
			return diag.Errorf("[ERROR-014] crash via disk update: %s", err)
		}
	}

	return resourceDiskRead(ctx, d, meta)
}

func resourceDiskDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	disk, err := manager.GetDisk(d.Id())
	if err != nil {
		return diag.Errorf("[ERROR-014] crash via get disk: %s", err)
	}

	if disk.Vm != nil {
		vm, err := manager.GetVm(disk.Vm.ID)
		if err != nil {
			return diag.Errorf("[ERROR-014] crash via get vm: %s", err)
		}
		err = vm.DetachDisk(disk)
		if err != nil {
			return diag.Errorf("[ERROR-014] crash via detach disk: %s", err)
		}
	}

	if err = disk.Delete(); err != nil {
		return diag.Errorf("[ERROR-014] crash via delete disk: %s", err)
	}
	disk.WaitLock()

	return nil
}

func resourceDiskImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	manager := meta.(*CombinedConfig).Manager()

	disk, err := manager.GetDisk(d.Id())
	if err != nil {
		return nil, fmt.Errorf("[ERROR-014]: crash via get disk: %s", err)
	}

	d.SetId(disk.ID)

	return []*schema.ResourceData{d}, nil
}
