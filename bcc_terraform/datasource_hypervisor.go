package bcc_terraform

import (
	"context"
	"strings"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceHypervisor() *schema.Resource {
	args := Defaults()
	args.injectResultHypervisor()
	args.injectContextRequiredProject()
	args.injectContextGetHypervisor()

	return &schema.Resource{
		ReadContext: dataSourceHypervisorRead,
		Schema:      args,
	}
}

func dataSourceHypervisorRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	project, err := GetProjectById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-004] crash via getting project: %s", err)
	}

	target, err := checkDatasourceNameOrId(d)
	if err != nil {
		return diag.Errorf("[ERROR-004] crash via getting hypervisor: %s", err)
	}

	var hypervisor *bcc.Hypervisor
	if strings.EqualFold(target, "id") {
		hypervisor, err = GetHypervisorByIdRead(d, manager, project)
		if err != nil {
			return diag.Errorf("[ERROR-004] crash via getting hypervisor: %s", err)
		}
	} else {
		hypervisor, err = GetHypervisorByName(d, manager, project)
		if err != nil {
			return diag.Errorf("[ERROR-004] crash via getting hypervisor: %s", err)
		}
	}

	fields := map[string]interface{}{
		"id":               hypervisor.ID,
		"name":             hypervisor.Name,
		"type":             hypervisor.Type,
		"project_id":       project.ID,
		"cpu_per_vm":       hypervisor.CpuPerVm,
		"ram_per_vm":       hypervisor.RamPerVm,
		"ports_per_device": hypervisor.PortsPerDevice,
		"disks_per_vm":     hypervisor.DisksPerVm,
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-004] crash via set attrs: %s", err)
	}
	return nil
}
