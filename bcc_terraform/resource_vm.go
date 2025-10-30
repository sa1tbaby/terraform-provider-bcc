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
	args.injectContextVdcById()

	return &schema.Resource{
		CreateContext: resourceVmCreate,
		ReadContext:   resourceVmRead,
		UpdateContext: resourceVmUpdate,
		DeleteContext: resourceVmDelete,
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

func resourceVmCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	targetVdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("vdc_id: Error getting VDC: %s", err)
	}

	template, err := GetTemplateById(d, manager)
	if err != nil {
		return diag.Errorf("template_id: Error getting template: %s", err)
	}

	vmName := d.Get("name").(string)
	cpu := d.Get("cpu").(int)
	ram := d.Get("ram").(float64)
	userData := d.Get("user_data").(string)
	hotAdd := d.Get("hot_add").(bool)
	platform := d.Get("platform").(string)
	log.Printf(vmName, cpu, ram, userData, template.Name, platform)

	// System disk creation
	systemDiskArgs := d.Get("system_disk.0").(map[string]interface{})
	diskSize := systemDiskArgs["size"].(int)
	storageProfileId := systemDiskArgs["storage_profile_id"].(string)
	storageProfile, err := targetVdc.GetStorageProfile(storageProfileId)
	if err != nil {
		return diag.Errorf("storage_profile_id: Error storage profile %s not found", storageProfileId)
	}

	systemDiskList := make([]*bcc.Disk, 1)
	newDisk := bcc.NewDisk("Основной диск", diskSize, storageProfile)
	systemDiskList[0] = &newDisk

	portsIds := collectVmNetworks(d)
	ports := make([]*bcc.Port, len(portsIds))
	for i, portId := range portsIds {
		port, err := manager.GetPort(portId)
		if err != nil {
			return diag.FromErr(err)
		}
		ports[i] = port
	}

	var floatingIp *string = nil
	if d.Get("floating").(bool) {
		floatingIpStr := "RANDOM_FIP"
		floatingIp = &floatingIpStr
	}

	newVm := bcc.NewVm(
		vmName, cpu, ram, template, nil, platform,
		&userData, ports, systemDiskList, floatingIp, hotAdd,
	)
	newVm.Tags = unmarshalTagNames(d.Get("tags"))
	newVm.AffinityGroups = d.Get("affinity_groups")

	if err = targetVdc.CreateVm(&newVm); err != nil {
		return diag.Errorf("Error creating vm: %s", err)
	}
	if err = newVm.WaitLock(); err != nil {
		return diag.FromErr(err)
	}

	vmPower := d.Get("power").(bool)
	if !vmPower {
		if err = newVm.PowerOff(); err != nil {
			return diag.FromErr(err)
		}
	}

	systemDisk := make([]interface{}, 1)
	systemDisk[0] = map[string]interface{}{
		"id":                 newVm.Disks[0].ID,
		"name":               "Основной диск",
		"size":               newVm.Disks[0].Size,
		"storage_profile_id": newVm.Disks[0].StorageProfile.ID,
	}

	syncDisks(d, manager, targetVdc, &newVm)

	d.Set("system_disk", systemDisk)
	d.SetId(newVm.ID)

	log.Printf("[INFO] VM created, ID: %s", d.Id())

	return resourceVmRead(ctx, d, meta)
}

func resourceVmRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	vm, err := manager.GetVm(d.Id())
	if err != nil {
		if err.(*bcc.ApiError).Code() == 404 {
			d.SetId("")
			return nil
		} else {
			return diag.Errorf("id: Error getting vm: %s", err)
		}
	}

	d.SetId(vm.ID)
	d.Set("name", vm.Name)
	d.Set("cpu", vm.Cpu)
	d.Set("ram", vm.Ram)
	d.Set("template_id", vm.Template.ID)
	d.Set("power", vm.Power)
	d.Set("hot_add", vm.HotAdd)
	d.Set("platform", vm.Platform)
	d.Set("tags", marshalTagNames(vm.Tags))

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

			d.Set("system_disk", systemDisk)
			continue
		}
		flattenDisks[i-1] = disk.ID
	}
	d.Set("disks", flattenDisks)

	flattenPorts := make([]string, len(vm.Ports))
	flattenNetworks := make([]map[string]interface{}, 0, len(vm.Ports))
	for i, port := range vm.Ports {
		flattenPorts[i] = port.ID
		flattenNetworks = append(flattenNetworks, map[string]interface{}{
			"id":         port.ID,
			"ip_address": port.IpAddress,
		})
	}
	d.Set("ports", flattenPorts)
	d.Set("networks", flattenNetworks)
	d.Set("floating", vm.Floating != nil)
	d.Set("floating_ip", "")

	if vm.Floating != nil {
		d.Set("floating_ip", vm.Floating.IpAddress)
	}

	affGrs := make([]string, len(vm.AffinityGroups))
	for idx, item := range vm.AffinityGroups {
		affGrs[idx] = item.ID
	}
	d.Set("affinity_groups", affGrs)

	return nil
}

func resourceVmUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	targetVdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("vdc_id: Error getting VDC: %s", err)
	}

	hasFlavorChanged := false
	needUpdate := false

	vm, err := manager.GetVm(d.Id())
	if err != nil {
		return diag.Errorf("id: Error getting vm: %s", err)
	}

	if diags := syncNetworks(d, manager, vm); diags.HasError() {
		return diags
	}

	// Detect vm changes
	if d.HasChange("name") {
		needUpdate = true
		vm.Name = d.Get("name").(string)
	}

	if d.HasChange("cpu") || d.HasChange("ram") {
		needUpdate = true
		hasFlavorChanged = true
		vm.Cpu = d.Get("cpu").(int)
		vm.Ram = d.Get("ram").(float64)
	}

	if d.HasChange("hot_add") {
		needUpdate = true
		vm.HotAdd = d.Get("hot_add").(bool)
	}

	needPowerOn := false
	if hasFlavorChanged && !vm.HotAdd && vm.Power {
		if err = vm.PowerOff(); err != nil {
			return diag.FromErr(err)
		}
		needPowerOn = true
	}

	if d.HasChange("tags") {
		needUpdate = true
		vm.Tags = unmarshalTagNames(d.Get("tags"))
	}

	if d.HasChange("affinity_groups") {
		needUpdate = true
		vm.AffinityGroups = d.Get("affinity_groups")
	}

	if needUpdate {
		if err := repeatOnError(vm.Update, vm); err != nil {
			return diag.Errorf("Error updating vm: %s", err)
		}
	}

	if needPowerOn {
		if err = vm.PowerOn(); err != nil {
			return diag.FromErr(err)
		}
	} else if d.HasChange("power") {
		a := d.Get("power").(bool)
		if a {
			if err = vm.PowerOn(); err != nil {
				return diag.FromErr(err)
			}
		} else {
			if err = vm.PowerOff(); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if diags := syncDisks(d, manager, targetVdc, vm); diags.HasError() {
		return diags
	}

	return resourceVmRead(ctx, d, meta)
}

func resourceVmDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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
		if err := vm.DisconnectPort(port); err != nil {
			return diag.FromErr(err)
		}
	}

	if err = vm.WaitLock(); err != nil {
		return diag.FromErr(err)
	}

	err = vm.Delete()
	if err != nil {
		return diag.Errorf("Error deleting vm: %s", err)
	}

	if err = vm.WaitLock(); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func syncNetworks(d *schema.ResourceData, manager *bcc.Manager, vm *bcc.Vm) (err diag.Diagnostics) {
	var targetDefinition string

	if d.HasChange("networks") {
		targetDefinition = "networks"
	} else if d.HasChange("ports") {
		targetDefinition = "ports"
	} else {
		return nil
	}

	olfFloating, newFloating := d.GetChange("floating")
	oldNetworksRaw, newNetworksRaw := d.GetChange(targetDefinition)
	oldNetworks := make([]string, len(oldNetworksRaw.([]interface{})))
	newNetworks := make([]string, len(newNetworksRaw.([]interface{})))
	if targetDefinition == "ports" {
		for idx, item := range newNetworksRaw.([]interface{}) {
			newNetworks[idx] = item.(string)
		}
		for idx, item := range oldNetworksRaw.([]interface{}) {
			oldNetworks[idx] = item.(string)
		}
	} else {
		for idx, item := range newNetworksRaw.([]interface{}) {
			_item := item.(map[string]interface{})
			newNetworks[idx] = _item["id"].(string)
		}
		for idx, item := range oldNetworksRaw.([]interface{}) {
			_item := item.(map[string]interface{})
			oldNetworks[idx] = _item["id"].(string)
		}
	}

	newNetworksSet := make(map[string]bool)
	for _, item := range newNetworks {
		newNetworksSet[item] = true
	}

	if len(newNetworksSet) == 0 && newFloating.(bool) {
		return diag.Errorf("floating cannot be added without existing networks")
	}

	if olfFloating.(bool) && !newFloating.(bool) {
		vm.Floating = &bcc.Port{IpAddress: nil}
		if err := d.Set("floating", vm.Floating != nil); err != nil {
			return diag.Errorf("Error setting floating: %s", err)
		}
		if err := repeatOnError(vm.Update, vm); err != nil {
			return diag.Errorf("Error with deletting floating for vm: %s", err)
		}
	}

	for _, item := range oldNetworks {
		if newNetworksSet[item] {
			delete(newNetworksSet, item)
		} else {
			if err = DisconnectOldPort(item, manager, vm); err != nil {
				log.Printf("Error disconnecting new port: %v", err)
				return err
			}
		}
	}

	for item := range newNetworksSet {
		if err = ConnectNewPort(item, manager, vm); err != nil {
			log.Printf("Error connecting new port: %v", err)
			return err
		}
	}

	if newFloating.(bool) {
		vm.Floating = &bcc.Port{ID: "RANDOM_FIP"}
		if err := d.Set("floating", vm.Floating != nil); err != nil {
			return diag.Errorf("Error setting floating: %s", err)
		}
		if err := repeatOnError(vm.Update, vm); err != nil {
			return diag.Errorf("Error with adding floating for vm: %s", err)
		}
	}

	return nil
}

func collectVmNetworks(d *schema.ResourceData) []string {
	portsIds := parseVmPorts(d.Get("ports"))
	networksIds := parseVmNetworks(d.Get("networks"))

	if len(networksIds) == 0 && len(portsIds) != 0 {
		return portsIds
	} else {
		return networksIds
	}
}

func parseVmPorts(d interface{}) (portsIds []string) {
	ports := d.([]interface{})
	portsIds = make([]string, 0, len(ports))
	for _, portIdValue := range ports {
		portsIds = append(portsIds, portIdValue.(string))
	}

	return
}

func parseVmNetworks(d interface{}) (networksIds []string) {
	networks := d.([]interface{})
	networksIds = make([]string, 0, len(networks))
	for _, network := range networks {
		portMap := network.(map[string]interface{})
		networksIds = append(networksIds, portMap["id"].(string))
	}
	return
}

func ConnectNewPort(portId string, manager *bcc.Manager, vm *bcc.Vm) diag.Diagnostics {
	port, err := manager.GetPort(portId)
	if err != nil {
		return diag.FromErr(err)
	}

	if port.Connected != nil && port.Connected.ID != vm.ID {
		if err = vm.DisconnectPort(port); err != nil {
			return diag.FromErr(err)
		}
		if err = vm.WaitLock(); err != nil {
			return diag.FromErr(err)
		}
	}

	log.Printf("Port `%s` will be Attached", port.ID)

	if err = vm.ConnectPort(port, true); err != nil {
		return diag.Errorf("Ports: Error Cannot attach port `%s`: %s", port.ID, err)
	}

	return nil
}

func DisconnectOldPort(portId string, manager *bcc.Manager, vm *bcc.Vm) diag.Diagnostics {
	port, err := manager.GetPort(portId)
	if err != nil {
		return diag.FromErr(err)
	}

	if port.Connected != nil && port.Connected.ID == vm.ID {
		log.Printf("Port %s found on vm and not mentioned in the state."+
			" Port will be detached", port.ID)

		if err := vm.DisconnectPort(port); err != nil {
			return diag.FromErr(err)
		}
		if err = vm.WaitLock(); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func syncDisks(d *schema.ResourceData, manager *bcc.Manager, vdc *bcc.Vdc, vm *bcc.Vm) (diagErr diag.Diagnostics) {
	targetVdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("vdc_id: Error getting VDC: %s", err)
	}

	// Which disks are present on vm and not mentioned in the state?
	// Detach disks
	diagErr = detachOldDisk(d, manager, vm)
	if diagErr != nil {
		return
	}

	// List disks to join
	diagErr = attachNewDisk(d, manager, vm)
	if diagErr != nil {
		return
	}

	// System disk resize
	if d.HasChange("system_disk") {
		systemDiskArgs := d.Get("system_disk.0").(map[string]interface{})
		systemDiskId := systemDiskArgs["id"].(string)
		diskSize := systemDiskArgs["size"].(int)
		systemDisk, err := manager.GetDisk(systemDiskId)
		if err != nil {
			return diag.Errorf("system_disk: Error getting system disk id: %s", err)
		}

		if err = systemDisk.Resize(diskSize); err != nil {
			return diag.Errorf("size: Error resizing disk: %s", err)
		}

		if !d.HasChange("system_disk.0.storage_profile_id") {
			return
		}

		storageProfileId := d.Get("system_disk.0.storage_profile_id").(string)
		storageProfile, err := targetVdc.GetStorageProfile(storageProfileId)
		if err != nil {
			return diag.Errorf("storage_profile_id: Error getting storage profile: %s", err)
		}

		err = systemDisk.UpdateStorageProfile(*storageProfile)
		if err != nil {
			return diag.Errorf("Error updating storage: %s", err)
		}
	}

	return
}

func attachNewDisk(d *schema.ResourceData, manager *bcc.Manager, vm *bcc.Vm) diag.Diagnostics {
	disksIds := d.Get("disks").(*schema.Set).List()
	// Save system_disk
	systemDiskResource := d.Get("system_disk.0")
	systemDisk := systemDiskResource.(map[string]interface{})["id"].(string)
	var needReload bool
	disksIds = append(disksIds, systemDisk)
	vmId := vm.ID

	for _, diskId := range disksIds {
		found := false
		for _, disk := range vm.Disks {
			if diskId == disk.ID {
				found = true
				break
			}
		}

		if !found {
			disk, err := manager.GetDisk(diskId.(string))
			if err != nil {
				return diag.FromErr(err)
			}

			if disk.Vm != nil && disk.Vm.ID != vmId {
				log.Printf("Disk %s found on other vm and will be detached for attached to vm.", disk.ID)
				if err = vm.DetachDisk(disk); err != nil {
					return diag.FromErr(err)
				}
				if err = vm.Reload(); err != nil {
					return diag.FromErr(err)
				}
				if err = vm.WaitLock(); err != nil {
					return diag.FromErr(err)
				}
			}
			log.Printf("Disk `%s` will be Attached", disk.ID)

			if err = vm.AttachDisk(disk); err != nil {
				return diag.Errorf("ERROR. Cannot attach disk `%s`: %s", disk.ID, err)
			}

			needReload = true
		}
	}

	if needReload {
		if err := vm.Reload(); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func detachOldDisk(d *schema.ResourceData, manager *bcc.Manager, vm *bcc.Vm) diag.Diagnostics {
	disksIds := d.Get("disks").(*schema.Set).List()
	systemDiskResource := d.Get("system_disk.0")
	systemDisk := systemDiskResource.(map[string]interface{})["id"].(string)
	var needReload bool
	disksIds = append(disksIds, systemDisk)
	vmId := vm.ID

	for _, disk := range vm.Disks {
		found := false
		for _, diskId := range disksIds {
			if diskId == disk.ID {
				found = true
				break
			}
		}

		if !found {
			disk, err := manager.GetDisk(disk.ID)
			if err != nil {
				return diag.FromErr(err)
			}

			if disk.Vm != nil && disk.Vm.ID == vmId {
				log.Printf("Disk %s found on vm and not mentioned in the state."+
					" Disk will be detached", disk.ID)

				if err = vm.DetachDisk(disk); err != nil {
					return diag.FromErr(err)
				}

				needReload = true
			}
		}
	}

	if needReload {
		if err := vm.Reload(); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}
