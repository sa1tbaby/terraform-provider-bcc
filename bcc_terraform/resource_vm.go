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
			StateContext: resourceVmImport,
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
		return diag.Errorf("[ERROR-021]: %s", err)
	}

	template, err := GetTemplateById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-021]: %s", err)
	}

	config := struct {
		name                string
		cpu                 int
		ram                 float64
		platformId          string
		userData            string
		hotAdd              bool
		sysDisk             interface{}
		sysDiskSize         int
		sysStorageProfileId string
		AffinityGroups      []string
	}{
		name:                d.Get("name").(string),
		cpu:                 d.Get("cpu").(int),
		ram:                 d.Get("ram").(float64),
		platformId:          d.Get("platform_id").(string),
		userData:            d.Get("user_data").(string),
		hotAdd:              d.Get("hot_add").(bool),
		sysDiskSize:         d.Get("system_disk.0.size").(int),
		sysStorageProfileId: d.Get("system_disk.0.storage_profile_id").(string),
		AffinityGroups:      d.Get("affinity_groups").([]string),
		sysDisk:             d.Get("system_disk"),
	}

	// System disk creation
	storageProfile, err := targetVdc.GetStorageProfile(config.sysStorageProfileId)
	if err != nil {
		return diag.Errorf("[ERROR-021]: %s", err)
	}
	systemDiskList := make([]*bcc.Disk, 1)
	newDisk := bcc.NewDisk("Основной диск", config.sysDiskSize, storageProfile)
	systemDiskList[0] = &newDisk

	// Ports creation
	portsIds := collectVmNetworks(d)
	ports := make([]*bcc.Port, len(portsIds))
	for i, portId := range portsIds {
		port, err := manager.GetPort(portId)
		if err != nil {
			return diag.Errorf("[ERROR-021]: %s", err)
		}
		ports[i] = port
	}
	var floatingIp *string = nil
	if d.Get("floating").(bool) {
		floatingIpStr := "RANDOM_FIP"
		floatingIp = &floatingIpStr
	}
	newVm := bcc.NewVm(
		config.name, config.cpu, config.ram, template, nil,
		&config.userData, ports, systemDiskList, floatingIp,
	)

	newVm.HotAdd = config.hotAdd
	newVm.Tags = unmarshalTagNames(d.Get("tags"))

	if config.platformId != "" {
		newVm.Platform, err = manager.GetPlatform(config.platformId)
		if err != nil {
			return diag.Errorf("[ERROR-021]: crash via getting template: %s", err)
		}
	}

	for _, item := range config.AffinityGroups {
		newVm.AffinityGroups = append(newVm.AffinityGroups, &bcc.AffinityGroup{ID: item})
	}

	if err = targetVdc.CreateVm(&newVm); err != nil {
		return diag.Errorf("[ERROR-021]: %s", err)
	}
	if err = newVm.WaitLock(); err != nil {
		return diag.Errorf("[ERROR-021]: %s", err)
	}

	vmPower := d.Get("power").(bool)
	if !vmPower {
		if err = newVm.PowerOff(); err != nil {
			return diag.Errorf("[ERROR-021]: %s", err)
		}
	}

	systemDisk := make([]interface{}, 1)
	systemDisk[0] = map[string]interface{}{
		"id":                 newVm.Disks[0].ID,
		"name":               "Основной диск",
		"size":               newVm.Disks[0].Size,
		"storage_profile_id": newVm.Disks[0].StorageProfile.ID,
	}

	syncVmDisks(d, manager, targetVdc, &newVm)

	//if err = syncVmDisks(d, manager, targetVdc, &newVm); err != nil {
	//	return diag.Errorf("[ERROR-021]: %s", err)
	//}

	d.SetId(newVm.ID)
	fields := map[string]interface{}{
		"user_data":   config.userData,
		"system_disk": config.sysDisk,
	}
	if err = setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-021]: %s", err)
	}
	log.Printf("[INFO] VM created, ID: %s", d.Id())

	return resourceVmRead(ctx, d, meta)
}

func resourceVmImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	manager := meta.(*CombinedConfig).Manager()

	vm, err := manager.GetVm(d.Id())
	if err != nil {
		return nil, err
	}

	d.SetId(vm.ID)

	return []*schema.ResourceData{d}, nil
}

func resourceVmRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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
				return diag.Errorf("error with setting system_disk %w", err)
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

	fields := map[string]interface{}{
		"vdc_id":          vm.Vdc.ID,
		"name":            vm.Name,
		"cpu":             vm.Cpu,
		"ram":             vm.Ram,
		"template_id":     vm.Template.ID,
		"power":           vm.Power,
		"hot_add":         vm.HotAdd,
		"platform":        vm.Platform.ID,
		"tags":            marshalTagNames(vm.Tags),
		"affinity_groups": vm.AffinityGroups,
		"disks":           flattenDisks,
		"ports":           flattenPorts,
		"networks":        flattenNetworks,
		"floating":        vm.Floating != nil,
		"floating_ip":     "",
	}

	if err = setResourceDataFromMap(d, fields); err != nil {
		return diag.FromErr(err)
	}

	if vm.Floating != nil {
		if err = d.Set("floating_ip", vm.Floating.IpAddress); err != nil {
			return diag.Errorf("error with setting floating_ip: %w", err)
		}
	}

	return nil
}

func resourceVmUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	targetVdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-021]: %s", err)
	}

	hasFlavorChanged := false
	needUpdate := false

	vm, err := manager.GetVm(d.Id())
	if err != nil {
		return diag.Errorf("[ERROR-021]: %s", err)
	}

	if diags := syncVmNetworks(d, manager, vm); diags.HasError() {
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
			return diag.Errorf("[ERROR-021]: %s", err)
		}
		needPowerOn = true
	}

	if d.HasChange("tags") {
		needUpdate = true
		vm.Tags = unmarshalTagNames(d.Get("tags"))
	}

	if d.HasChange("affinity_groups") {
		needUpdate = true
		var _affGrs []*bcc.AffinityGroup
		for _, item := range d.Get("affinity_groups").([]interface{}) {
			_affGrs = append(_affGrs, &bcc.AffinityGroup{ID: item.(string)})
		}
		vm.AffinityGroups = _affGrs
	}

	if needUpdate {
		if err := repeatOnError(vm.Update, vm); err != nil {
			return diag.Errorf("[ERROR-021]: crash via updating vm: %s", err)
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

func syncVmNetworks(d *schema.ResourceData, manager *bcc.Manager, vm *bcc.Vm) (err diag.Diagnostics) {
	var targetDefinition string

	if d.HasChange("networks") {
		targetDefinition = "networks"
	} else if d.HasChange("ports") {
		targetDefinition = "ports"
	} else {
		if _, ok := d.GetOk("networks"); ok {
			targetDefinition = "networks"
		} else if _, ok := d.GetOk("ports"); ok {
			targetDefinition = "ports"
		} else {
			return nil
		}
	}

	oldNetworksRaw, newNetworksRaw := d.GetChange(targetDefinition)
	olfFloating, newFloating := d.GetChange("floating")

	newNetworksSet := make(map[string]bool)
	oldNetworks := make([]string, len(oldNetworksRaw.([]interface{})))
	newNetworks := make([]string, len(newNetworksRaw.([]interface{})))

	if targetDefinition == "ports" {
		for idx, item := range newNetworksRaw.([]interface{}) {
			newNetworks[idx] = item.(string)
		}
		for idx, item := range oldNetworksRaw.([]interface{}) {
			oldNetworks[idx] = item.(string)
		}
	} else if targetDefinition == "networks" {
		for idx, item := range newNetworksRaw.([]interface{}) {
			_item := item.(map[string]interface{})
			newNetworks[idx] = _item["id"].(string)
		}
		for idx, item := range oldNetworksRaw.([]interface{}) {
			_item := item.(map[string]interface{})
			oldNetworks[idx] = _item["id"].(string)
		}
	}

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
			if err = disconnectVmOldPort(item, manager, vm); err != nil {
				log.Printf("Error disconnecting new port: %v", err)
				return err
			}
		}
	}

	for item := range newNetworksSet {
		if err = connectVmNewPort(item, manager, vm); err != nil {
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

func connectVmNewPort(portId string, manager *bcc.Manager, vm *bcc.Vm) diag.Diagnostics {
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

func disconnectVmOldPort(portId string, manager *bcc.Manager, vm *bcc.Vm) diag.Diagnostics {
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

func syncVmDisks(d *schema.ResourceData, manager *bcc.Manager, vdc *bcc.Vdc, vm *bcc.Vm) error {
	targetVdc, err := GetVdcById(d, manager)
	if err != nil {
		return fmt.Errorf("crash via getting VDC: %s", err)
	}

	// Which disks are present on vm and not mentioned in the state?
	// Detach disks
	if err = detachVmOldDisk(d, manager, vm); err != nil {
		return fmt.Errorf("crash via detaching old disk: %s", err)
	}

	// List disks to join
	if err = attachVmNewDisk(d, manager, vm); err != nil {
		return fmt.Errorf("crash via attaching new disk: %s", err)
	}

	// System disk resize
	if d.HasChange("system_disk") {
		systemDiskArgs := d.Get("system_disk.0").(map[string]interface{})
		systemDiskId := systemDiskArgs["id"].(string)
		diskSize := systemDiskArgs["size"].(int)
		systemDisk, err := manager.GetDisk(systemDiskId)
		if err != nil {
			return err
		}

		if err = systemDisk.Resize(diskSize); err != nil {
			return fmt.Errorf("crash via resizing disk: %s", err)
		}

		if !d.HasChange("system_disk.0.storage_profile_id") {
			return err
		}

		storageProfileId := d.Get("system_disk.0.storage_profile_id").(string)
		storageProfile, err := targetVdc.GetStorageProfile(storageProfileId)
		if err != nil {
			return err
		}

		err = systemDisk.UpdateStorageProfile(*storageProfile)
		if err != nil {
			return err
		}
	}

	return nil
}

func attachVmNewDisk(d *schema.ResourceData, manager *bcc.Manager, vm *bcc.Vm) error {
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
				return err
			}

			if disk.Vm != nil && disk.Vm.ID != vmId {
				log.Printf("Disk %s found on other vm and will be detached for attached to vm.", disk.ID)
				if err = vm.DetachDisk(disk); err != nil {
					return err
				}
				if err = vm.Reload(); err != nil {
					return err
				}
				if err = vm.WaitLock(); err != nil {
					return err
				}
			}
			log.Printf("Disk `%s` will be Attached", disk.ID)

			if err = vm.AttachDisk(disk); err != nil {
				return err
			}

			needReload = true
		}
	}

	if needReload {
		if err := vm.Reload(); err != nil {
			return err
		}
	}
	return nil
}

func detachVmOldDisk(d *schema.ResourceData, manager *bcc.Manager, vm *bcc.Vm) error {
	var needReload bool
	vmId := vm.ID

	systemDiskResource := d.Get("system_disk.0")
	systemDisk := systemDiskResource.(map[string]interface{})["id"].(string)

	disksIds := d.Get("disks").(*schema.Set).List()
	disksIds = append(disksIds, systemDisk)

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
				return err
			}

			if disk.Vm != nil && disk.Vm.ID == vmId {
				log.Printf("Disk %s found on vm and not mentioned in the state."+
					" Disk will be detached", disk.ID)
				if err = vm.DetachDisk(disk); err != nil {
					return err
				}
				needReload = true
			}
		}
	}

	if needReload {
		if err := vm.Reload(); err != nil {
			return err
		}
	}
	return nil
}
