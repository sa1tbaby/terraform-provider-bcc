package bcc_terraform

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/hashstructure/v2"
)

func dataSourceVms() *schema.Resource {
	args := Defaults()
	args.injectContextRequiredVdc()
	args.injectResultListVm()

	return &schema.Resource{
		ReadContext: dataSourceVmsRead,
		Schema:      args,
	}
}

func dataSourceVmsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	vdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-023] crash via getting vdc: %s", err)
	}

	vmsList, err := vdc.GetVms()
	if err != nil {
		return diag.Errorf("[ERROR-023] crash via retrieving vms: %s", err)
	}

	vmsMap := make([]map[string]interface{}, len(vmsList))
	for i, vm := range vmsList {

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

		vmsMap[i] = map[string]interface{}{
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
		}

		if vm.Floating != nil {
			vmsMap[i]["floating"] = true
			vmsMap[i]["floating_ip"] = vm.Floating.IpAddress
		}
	}

	hash, err := hashstructure.Hash(vmsList, hashstructure.FormatV2, nil)
	if err != nil {
		diag.Errorf("[ERROR-023] crash via calculate hash: %s", err)
	}

	fields := map[string]interface{}{
		"id":     fmt.Sprintf("vms/%d", hash),
		"vdc_id": vdc.ID,
		"vms":    vmsMap,
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-023] crash via set attrs: %s", err)
	}

	return nil
}
