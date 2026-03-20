package bcc_terraform

import (
	"context"
	"fmt"
	"strings"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceKubernetes() *schema.Resource {
	args := Defaults()
	args.injectContextDataK8s()
	args.injectContextRequiredVdc()
	args.injectContextGetK8s()

	return &schema.Resource{
		ReadContext: dataSourceKubernetesRead,
		Schema:      args,
	}
}

func dataSourceKubernetesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	target, err := checkDatasourceNameOrId(d)
	if err != nil {
		return diag.Errorf("[ERROR-034] crash via getting Kubernetes: %s", err)
	}

	var k8s *bcc.Kubernetes
	if strings.EqualFold(target, "id") {
		k8sId := d.Get("id").(string)
		k8s, err = manager.GetKubernetes(k8sId)
		if err != nil {
			return diag.Errorf("[ERROR-034] crash via getting Kubernetes by id=%s: %s", k8sId, err)
		}
	} else {
		vdc, err := GetVdcById(d, manager)
		if err != nil {
			return diag.Errorf("[ERROR-034] crash via getting vdc: %s", err)
		}

		k8s, err = GetKubernetesByName(d, manager, vdc)
		if err != nil {
			return diag.Errorf("[ERROR-034] crash via getting Kubernetes by name: %s", err)
		}
	}

	vms := make([]*string, len(k8s.Vms))
	for i, vm := range k8s.Vms {
		vms[i] = &vm.ID
	}

	dashboard, err := k8s.GetKubernetesDashBoardUrl()
	if err != nil {
		return diag.Errorf("[ERROR-034] crash via getting Kubernetes dashboard url: %s", err)
	}

	fields := map[string]interface{}{
		"id":                      k8s.ID,
		"name":                    k8s.Name,
		"node_cpu":                k8s.NodeCpu,
		"node_ram":                k8s.NodeRam,
		"template_id":             k8s.Template.ID,
		"node_disk_size":          k8s.NodeDiskSize,
		"nodes_count":             k8s.NodesCount,
		"user_public_key_id":      k8s.UserPublicKey,
		"node_storage_profile_id": k8s.NodeStorageProfile.ID,
		"floating":                false,
		"floating_ip":             "",
		"vms":                     vms,
		"dashboard_url":           fmt.Sprint(manager.BaseURL, *dashboard.DashBoardUrl),
		"tags":                    marshalTagNames(k8s.Tags),
	}

	if k8s.Floating != nil {
		fields["floating"] = true
		fields["floating_ip"] = k8s.Floating.IpAddress
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-034] crash via set attrs: %s", err)
	}
	return nil
}
