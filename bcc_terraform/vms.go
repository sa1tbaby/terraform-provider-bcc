package bcc_terraform

import (
	"fmt"
	"log"
	"strings"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func (args *Arguments) injectContextGetVm() {
	args.merge(Arguments{
		"name": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "name of the Vm",
		},
		"id": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "id of the vm",
		},
	})
}

func (args *Arguments) injectContextVmById() {
	args.merge(Arguments{
		"vm_id": {
			Type:     schema.TypeString,
			Required: true,
			ValidateDiagFunc: validation.ToDiagFunc(
				validation.StringIsNotEmpty,
			),
			Description: "id of the Vm",
		},
	})
}

func (args *Arguments) injectCreateVm() {
	systemDisk := Defaults()
	systemDisk.injectSystemDisk()

	args.merge(Arguments{
		"name": {
			Type:     schema.TypeString,
			Required: true,
			ValidateFunc: validation.All(
				validation.NoZeroValues,
				validation.StringLenBetween(1, 100),
			),
			Description: "name of the Vm",
		},
		"description": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "description of the Vm",
		},
		"platform": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			ForceNew:    true,
			Description: "platform of the Vm",
		},
		"cpu": {
			Type:         schema.TypeInt,
			Required:     true,
			ValidateFunc: validation.IntBetween(1, 128),
			Description:  "the number of virtual cpus",
		},
		"ram": {
			Type:        schema.TypeFloat,
			Required:    true,
			Description: "memory of the Vm in gigabytes",
		},
		"template_id": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "id of the Template",
		},
		"user_data": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "script for cloud-init",
		},
		"system_disk": {
			Type:     schema.TypeList,
			Required: true,
			MinItems: 1,
			MaxItems: 1,
			Elem: &schema.Resource{
				Schema: systemDisk,
			},
			Description: "System disk.",
		},
		"disks": {
			Type:        schema.TypeSet,
			Optional:    true,
			Description: "list of Disks attached to the Vm",
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"ports": {
			Type:        schema.TypeList,
			Optional:    true,
			Computed:    true,
			MinItems:    1,
			MaxItems:    10,
			Description: "List of Ports connected to the Vm",
			Deprecated:  "Use networks instead of ports",
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"networks": {
			Type:        schema.TypeList,
			Optional:    true,
			MinItems:    1,
			MaxItems:    10,
			Description: "List of Ports connected to the Vm",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"id": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "Id of the network",
					},
					"ip_address": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "IP of the Port",
					},
				},
			},
		},
		"floating": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "enable floating ip for the Vm",
		},
		"floating_ip": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "floating ip for the Vm. May be omitted",
		},
		"tags": newTagNamesResourceSchema("tags of the Vm"),
		"power": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     true,
			Description: "power of vw on/off",
		},
		"hot_add": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Enabling resources hot swap for vm",
		},
		"affinity_groups": {
			Type:        schema.TypeList,
			Optional:    true,
			MinItems:    1,
			MaxItems:    10,
			Description: "List of Affinity Groups connected to the Vm",
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
	})
}

func (args *Arguments) injectResultVm() {
	systemDisk := Defaults()
	systemDisk.injectDataSystemDisk()

	args.merge(Arguments{
		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "id of the Vm",
		},
		"name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "name of the Vm",
		},
		"description": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "description of the Vm",
		},
		"cpu": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "the number of virtual cpus",
		},
		"ram": {
			Type:        schema.TypeFloat,
			Computed:    true,
			Description: "memory of the Vm in gigabytes",
		},
		"template_id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "id of the Template",
		},
		"template_name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "name of the Template",
		},
		"floating": {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "enable floating ip for the Vm",
		},
		"floating_ip": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "floating_ip of the Vm. May be omitted",
		},
		"power": {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "power of vw on/off",
		},
		"ports": {
			Type:        schema.TypeList,
			Computed:    true,
			Description: "List of Ports connected to the Vm",
			Deprecated:  "Use networks instead of ports",
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"networks": {
			Type:        schema.TypeList,
			Computed:    true,
			Description: "List of Ports connected to the Vm",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"id": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "Id of the network",
					},
					"ip_address": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "IP of the Port",
					},
				},
			},
		},
		"platform": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "platform of the Vm",
		},
		"system_disk": {
			Type:        schema.TypeList,
			Computed:    true,
			Elem:        &schema.Resource{Schema: systemDisk},
			Description: "System disk.",
		},
		"disks": {
			Type:        schema.TypeSet,
			Computed:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "list of Disks attached to the Vm",
		},
		"hot_add": {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "Enabling resources hot swap for vm",
		},
		"affinity_groups": {
			Type:        schema.TypeList,
			Computed:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "List of Affinity Groups connected to the Vm",
		},
		"tags": newTagNamesDataSchema("tags of the Vm"),
	})
}

func (args *Arguments) injectSystemDisk() {
	args.merge(Arguments{
		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "id of the System Disk",
		},
		"name": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"size": {
			Type:     schema.TypeInt,
			Required: true,
		},
		"storage_profile_id": {
			Type:     schema.TypeString,
			Required: true,
		},
		"external_id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "external id of the volume. It can be empty",
		},
	})
}

func (args *Arguments) injectDataSystemDisk() {
	args.merge(Arguments{
		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "id of the System Disk",
		},
		"name": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"size": {
			Type:     schema.TypeInt,
			Computed: true,
		},
		"storage_profile_id": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"external_id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "external id of the volume. It can be empty",
		},
	})
}

func (args *Arguments) injectResultListVm() {
	s := Defaults()
	s.injectResultVm()

	args.merge(Arguments{
		"vms": {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: s,
			},
		},
	})
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

	if strings.EqualFold(targetDefinition, "ports") {
		for idx, item := range newNetworksRaw.([]interface{}) {
			newNetworks[idx] = item.(string)
		}
		for idx, item := range oldNetworksRaw.([]interface{}) {
			oldNetworks[idx] = item.(string)
		}
	} else if strings.EqualFold(targetDefinition, "networks") {
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

func syncVmDisks(d *schema.ResourceData, manager *bcc.Manager, vdc *bcc.Vdc, vm *bcc.Vm) (err error) {
	oldDisks, newDisks := d.GetChange("disks")
	newDisksMap := make(map[string]bool)
	oldDisksMap := make(map[string]bool)

	for _, item := range newDisks.(*schema.Set).List() {
		newDisksMap[item.(string)] = true
	}
	for _, item := range oldDisks.(*schema.Set).List() {
		_item := item.(string)

		if newDisksMap[_item] {
			delete(newDisksMap, _item)
		} else {
			oldDisksMap[_item] = true
		}
	}

	if err = detachVmDisks(oldDisksMap, manager, vm); err != nil {
		return fmt.Errorf("crash via detaching vm disks: %s", err)
	}

	if err = attachVmDisks(newDisksMap, manager, vm); err != nil {
		return fmt.Errorf("crash via attaching vm disks: %s", err)
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
		storageProfile, err := vdc.GetStorageProfile(storageProfileId)
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

func attachVmDisks(disks map[string]bool, manager *bcc.Manager, vm *bcc.Vm) (err error) {
	for diskId, ok := range disks {
		if !ok {
			continue
		}
		disk, err := manager.GetDisk(diskId)
		if err != nil {
			return fmt.Errorf("disk with id %s not found: %s", diskId, err)
		}

		if disk.Vm != nil && disk.Vm.ID != vm.ID {
			return fmt.Errorf("disk %s is already attached to another vm. please detach it before conecting", disk.ID)
		} else if disk.Vm == nil {
			if err = vm.AttachDisk(disk); err != nil {
				return fmt.Errorf("crash via attaching disk with id='%s': %s", disk.ID, err)
			}
		}
	}

	return
}

func detachVmDisks(disks map[string]bool, manager *bcc.Manager, vm *bcc.Vm) (err error) {
	for diskId, ok := range disks {
		if !ok {
			continue
		}
		disk, err := manager.GetDisk(diskId)
		if err != nil {
			return fmt.Errorf("disk with id %s not found: %s", diskId, err)
		}

		if disk.Vm != nil && disk.Vm.ID == vm.ID {
			log.Printf("Disk %s found on vm and not mentioned in the state. Disk will be detached", disk.ID)
			if err = vm.DetachDisk(disk); err != nil {
				return fmt.Errorf("crash via detaching disk with id='%s': %s", disk.ID, err)
			}
		}
	}

	return
}
