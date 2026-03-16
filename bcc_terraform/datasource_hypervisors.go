package bcc_terraform

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/hashstructure/v2"
)

func dataSourceHypervisors() *schema.Resource {
	args := Defaults()
	args.injectResultListHypervisor()
	args.injectContextRequiredProject()

	return &schema.Resource{
		ReadContext: dataSourceHypervisorsRead,
		Schema:      args,
	}
}

func dataSourceHypervisorsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	project, err := GetProjectById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-005] crash via getting project: %s", err)
	}

	hypervisorList, err := project.GetAvailableHypervisors()
	if err != nil {
		return diag.Errorf("[ERROR-005] crash via getting available hypervisors")
	}

	hypervisorMap := make([]map[string]interface{}, len(hypervisorList))
	for i, hypervisor := range hypervisorList {
		hypervisorMap[i] = map[string]interface{}{
			"id":               hypervisor.ID,
			"name":             hypervisor.Name,
			"type":             hypervisor.Type,
			"cpu_per_vm":       hypervisor.CpuPerVm,
			"ram_per_vm":       hypervisor.RamPerVm,
			"disks_per_vm":     hypervisor.DisksPerVm,
			"ports_per_device": hypervisor.PortsPerDevice,
		}
	}

	hash, err := hashstructure.Hash(hypervisorList, hashstructure.FormatV2, nil)
	if err != nil {
		diag.Errorf("[ERROR-005] crash via calculate hash: %s", err)
	}

	fields := map[string]interface{}{
		"id":          fmt.Sprintf("hypervisors/%d", hash),
		"project_id":  project.ID,
		"hypervisors": hypervisorMap,
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-005] crash via set attrs: %s", err)
	}

	return nil
}
