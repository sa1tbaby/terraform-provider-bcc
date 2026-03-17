package bcc_terraform

import (
	"context"
	"log"
	"time"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceVm() *schema.Resource {
	args := Defaults()
	args.injectCreateVm()
	args.injectContextRequiredVdc()

	return &schema.Resource{
		CreateContext: resourceVmCreate,
		UpdateContext: resourceVmUpdate,
		ReadContext:   resourceVmRead,
		DeleteContext: resourceVmDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceVmImport,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
		CustomizeDiff: resourceVmCustomizeDiff,
		Schema:        args,
	}
}

func resourceVmCustomizeDiff(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
	if d.HasChange("floating") {
		d.SetNewComputed("floating_ip")
	}
	return nil
}

func resourceVmCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	vdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-021]: %s", err)
	}

	template, err := GetTemplateById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-021]: %s", err)
	}

	config := struct {
		name                string
		description         string
		cpu                 int
		ram                 float64
		platform            string
		userData            string
		hotAdd              bool
		sysDisk             interface{}
		sysDiskSize         int
		sysStorageProfileId string
		disks               []interface{}
		AffinityGroups      []interface{}
	}{
		name:                d.Get("name").(string),
		cpu:                 d.Get("cpu").(int),
		description:         d.Get("description").(string),
		ram:                 d.Get("ram").(float64),
		platform:            d.Get("platform").(string),
		userData:            d.Get("user_data").(string),
		hotAdd:              d.Get("hot_add").(bool),
		sysDisk:             d.Get("system_disk"),
		sysDiskSize:         d.Get("system_disk.0.size").(int),
		sysStorageProfileId: d.Get("system_disk.0.storage_profile_id").(string),
		disks:               d.Get("disks").(*schema.Set).List(),
		AffinityGroups:      d.Get("affinity_groups").([]interface{}),
	}

	// System disk creation
	storageProfile, err := vdc.GetStorageProfile(config.sysStorageProfileId)
	if err != nil {
		return diag.Errorf("[ERROR-021]: %s", err)
	}
	systemDiskList := make([]*bcc.Disk, 1)
	systemDisk := bcc.NewDisk("Основной диск", config.sysDiskSize, storageProfile)
	systemDiskList[0] = &systemDisk

	// Ports creation
	portsIds := collectVmNetworks(d)
	portList := make([]*bcc.Port, len(portsIds))
	for i, portId := range portsIds {
		port, err := manager.GetPort(portId)
		if err != nil {
			return diag.Errorf("[ERROR-021]: %s", err)
		}
		portList[i] = port
	}

	var floatingIp *string = nil
	if d.Get("floating").(bool) {
		floatingIpStr := "RANDOM_FIP"
		floatingIp = &floatingIpStr
	}

	vm := bcc.NewVm(
		config.name, config.cpu, config.ram, template, nil,
		&config.userData, portList, systemDiskList, floatingIp,
	)
	vm.Description = config.description
	vm.HotAdd = config.hotAdd
	vm.Tags = unmarshalTagNames(d.Get("tags"))

	if config.platform != "" {
		vm.Platform, err = manager.GetPlatform(config.platform)
		if err != nil {
			return diag.Errorf("[ERROR-021]: crash via getting template: %s", err)
		}
	}

	for _, item := range config.AffinityGroups {
		vm.AffinityGroups = append(vm.AffinityGroups, &bcc.AffinityGroup{ID: item.(string)})
	}

	if err = vdc.CreateVm(&vm); err != nil {
		return diag.Errorf("[ERROR-021]: %s", err)
	}
	if err = vm.WaitLock(); err != nil {
		return diag.Errorf("[ERROR-021]: %s", err)
	}

	for _, item := range config.disks {
		disk, err := manager.GetDisk(item.(string))
		if err != nil {
			return diag.FromErr(err)
		}

		if disk.Vm != nil && disk.Vm.ID != vm.ID {
			return diag.Errorf("[ERROR-021] disk %s is already attached to another vm. please detach it before conecting", disk.ID)
		} else if disk.Vm == nil {
			if err = vm.AttachDisk(disk); err != nil {
				return diag.Errorf("[ERROR-021] crash via attaching disk with id='%s': %s", disk.ID, err)
			}
		}
	}

	vmPower := d.Get("power").(bool)
	if !vmPower {
		if err = vm.PowerOff(); err != nil {
			return diag.Errorf("[ERROR-021]: %s", err)
		}
	}

	fields := map[string]interface{}{
		"id":        vm.ID,
		"user_data": config.userData,
	}
	if err = setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-021]: %s", err)
	}
	log.Printf("[INFO] VM created, ID: %s", d.Id())

	return resourceVmRead(ctx, d, meta)
}

func resourceVmUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	targetVdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-021]: %s", err)
	}

	vm, err := manager.GetVm(d.Id())
	if err != nil {
		return diag.Errorf("[ERROR-021]: %s", err)
	}

	lifeCycle := &struct {
		NeedUpdate bool
		NeedReload bool
	}{
		NeedUpdate: false,
		NeedReload: false,
	}

	if diags := syncVmNetworks(d, manager, vm); diags.HasError() {
		return diags
	}

	// Detect vm changes
	if d.HasChange("name") {
		lifeCycle.NeedUpdate = true
		vm.Name = d.Get("name").(string)
	}

	if d.HasChange("description") {
		lifeCycle.NeedUpdate = true
		vm.Description = d.Get("description").(string)
	}

	if d.HasChange("cpu") || d.HasChange("ram") {
		lifeCycle.NeedUpdate = true
		if vm.Power && !vm.HotAdd {
			lifeCycle.NeedReload = true
		}
		vm.Cpu = d.Get("cpu").(int)
		vm.Ram = d.Get("ram").(float64)
	}

	if d.HasChange("hot_add") {
		lifeCycle.NeedUpdate = true
		if vm.Power {
			lifeCycle.NeedReload = true
		}
		vm.HotAdd = d.Get("hot_add").(bool)
	}

	if d.HasChange("affinity_groups") {
		lifeCycle.NeedUpdate = true
		var _affGrs []*bcc.AffinityGroup
		for _, item := range d.Get("affinity_groups").([]interface{}) {
			_affGrs = append(_affGrs, &bcc.AffinityGroup{ID: item.(string)})
		}
		vm.AffinityGroups = _affGrs
	}

	if d.HasChange("tags") {
		lifeCycle.NeedUpdate = true
		vm.Tags = unmarshalTagNames(d.Get("tags"))
	}

	if lifeCycle.NeedReload {
		if err = vm.PowerOff(); err != nil {
			return diag.Errorf("[ERROR-021]: %s", err)
		}
	}

	if lifeCycle.NeedUpdate {
		if err := repeatOnError(vm.Update, vm); err != nil {
			return diag.Errorf("[ERROR-021]: crash via updating vm: %s", err)
		}
	}

	if lifeCycle.NeedReload {
		if err = vm.PowerOn(); err != nil {
			return diag.FromErr(err)
		}
	} else if d.HasChange("power") {
		a := d.Get("power").(bool)
		if a {
			if err = vm.PowerOn(); err != nil {
				return diag.Errorf("[ERROR-021]: %s", err)
			}
		} else {
			if err = vm.PowerOff(); err != nil {
				return diag.Errorf("[ERROR-021]: %s", err)
			}
		}
	}

	if err = syncVmDisks(d, manager, targetVdc, vm); err != nil {
		return diag.Errorf("[ERROR-021]: %s", err)
	}

	return resourceVmRead(ctx, d, meta)
}

func resourceVmRead(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	vm, err := manager.GetVm(d.Id())
	if err != nil {
		if err.(*bcc.ApiError).Code() == 404 {
			d.SetId("")
			return nil
		} else {
			return diag.Errorf("[ERROR-021]: crash via getting vm-%s: %s", d.Id(), err)
		}
	}

	flattenDisks := make([]string, len(vm.Disks)-1)
	for i, disk := range vm.Disks {
		if i == 0 {
			systemDisk := make([]interface{}, 1)
			systemDisk[0] = map[string]interface{}{
				"id":                 disk.ID,
				"name":               "Основной диск",
				"size":               disk.Size,
				"storage_profile_id": disk.StorageProfile.ID,
				"external_id":        disk.ExternalID,
			}

			if err = d.Set("system_disk", systemDisk); err != nil {
				return diag.Errorf("error with setting system_disk %s", err)
			}
			continue
		}
		flattenDisks[i-1] = disk.ID
	}

	flattenPorts := make([]string, len(vm.Ports))
	flattenNetworks := make([]interface{}, len(vm.Ports))
	for i, port := range vm.Ports {
		flattenPorts[i] = port.ID
		flattenNetworks[i] = map[string]interface{}{
			"id":         port.ID,
			"ip_address": port.IpAddress,
		}
	}

	affGr := make([]string, len(vm.AffinityGroups))
	for i, aff := range vm.AffinityGroups {
		affGr[i] = aff.ID
	}

	fields := map[string]interface{}{
		"vdc_id":          vm.Vdc.ID,
		"name":            vm.Name,
		"description":     vm.Description,
		"cpu":             vm.Cpu,
		"ram":             vm.Ram,
		"template_id":     vm.Template.ID,
		"power":           vm.Power,
		"hot_add":         vm.HotAdd,
		"platform":        vm.Platform.ID,
		"tags":            marshalTagNames(vm.Tags),
		"affinity_groups": affGr,
		"disks":           flattenDisks,
		"ports":           flattenPorts,
		"networks":        flattenNetworks,
		"floating":        false,
		"floating_ip":     "",
	}

	if vm.Floating != nil {
		fields["floating"] = true
		fields["floating_ip"] = vm.Floating.IpAddress
	}

	if err = setResourceDataFromMap(d, fields); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceVmDelete(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	vm, err := manager.GetVm(d.Id())
	if err != nil {
		return diag.Errorf("id: Error getting vm: %s", err)
	}

	vm.Floating = &bcc.Port{IpAddress: nil}
	if err := repeatOnError(vm.Update, vm); err != nil {
		return diag.Errorf("Error updating vm: %s", err)
	}

	if err = vm.WaitLock(); err != nil {
		return diag.FromErr(err)
	}

	disksIds := d.Get("disks").(*schema.Set).List()
	for _, diskId := range disksIds {
		disk, err := manager.GetDisk(diskId.(string))
		if err != nil {
			return diag.FromErr(err)
		}
		if err = vm.DetachDisk(disk); err != nil {
			return diag.FromErr(err)
		}
	}

	portsIds := collectVmNetworks(d)
	for _, portId := range portsIds {
		port, err := manager.GetPort(portId)
		if err != nil {
			return diag.FromErr(err)
		}
		if err = vm.DisconnectPort(port); err != nil {
			return diag.FromErr(err)
		}
	}

	if err = vm.WaitLock(); err != nil {
		return diag.FromErr(err)
	}
	if err = vm.Delete(); err != nil {
		return diag.Errorf("Error deleting vm: %s", err)
	}
	vm.WaitLock()

	return nil
}

func resourceVmImport(_ context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	manager := meta.(*CombinedConfig).Manager()

	vm, err := manager.GetVm(d.Id())
	if err != nil {
		return nil, err
	}

	d.SetId(vm.ID)

	return []*schema.ResourceData{d}, nil
}
