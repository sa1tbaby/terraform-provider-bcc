package bcc_terraform

import (
	"context"
	"strings"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVm() *schema.Resource {
	args := Defaults()
	args.injectResultVm()
	args.injectContextRequiredVdc()
	args.injectContextGetVm()

	return &schema.Resource{
		ReadContext: dataSourceVmRead,
		Schema:      args,
	}
}

func dataSourceVmRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	target, err := checkDatasourceNameOrId(d)
	if err != nil {
		return diag.Errorf("[ERROR-022] crash via chose target: %s", err)
	}

	var vm *bcc.Vm
	if strings.EqualFold(target, "id") {
		vmId := d.Get("id").(string)
		vm, err = manager.GetVm(vmId)
		if err != nil {
			return diag.Errorf("[ERROR-022] crash via getting vm by id: %s", err)
		}
	} else if strings.EqualFold(target, "name") {
		vdc, err := GetVdcById(d, manager)
		if err != nil {
			return diag.Errorf("[ERROR-022] crash via getting vdc: %s", err)
		}

		vm, err = GetVmByName(d, manager, vdc)
		if err != nil {
			return diag.Errorf("[ERROR-022] crash via getting vm by name: %s", err)
		}
	} else {
		return diag.Errorf("[ERROR-022] id or name must be specified")
	}

	disks := make([]interface{}, 0)
	systemDisk := make([]interface{}, 1)
	for i, disk := range vm.Disks {
		if i == 0 {
			systemDisk[0] = map[string]interface{}{
				"id":                 disk.ID,
				"name":               "Основной диск",
				"size":               disk.Size,
				"storage_profile_id": disk.StorageProfile.ID,
				"external_id":        disk.ExternalID,
			}
			continue
		}
		disks = append(disks, disk.ID)
	}

	ports := make([]interface{}, len(vm.Ports))
	networks := make([]interface{}, len(vm.Ports))
	for i, port := range vm.Ports {
		ports[i] = port.ID
		networks[i] = map[string]interface{}{
			"id":         port.ID,
			"ip_address": port.IpAddress,
		}
	}

	affGr := make([]string, len(vm.AffinityGroups))
	for i, aff := range vm.AffinityGroups {
		affGr[i] = aff.ID
	}

	fields := map[string]interface{}{
		"id":              vm.ID,
		"name":            vm.Name,
		"description":     vm.Description,
		"cpu":             vm.Cpu,
		"ram":             vm.Ram,
		"template_id":     vm.Template.ID,
		"template_name":   vm.Template.Name,
		"power":           vm.Power,
		"platform":        vm.Platform.ID,
		"hot_add":         vm.HotAdd,
		"ports":           ports,
		"networks":        networks,
		"system_disk":     systemDisk,
		"disks":           disks,
		"affinity_groups": affGr,
		"tags":            marshalTagNames(vm.Tags),
		"floating":        false,
		"floating_ip":     "",
	}

	if vm.Floating != nil {
		fields["floating"] = true
		fields["floating_ip"] = vm.Floating.IpAddress
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-022] crash via set attrs: %s", err)
	}

	return nil
}
