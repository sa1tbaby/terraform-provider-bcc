package bcc_terraform

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/hashstructure/v2"
)

func dataSourceKubernetess() *schema.Resource {
	args := Defaults()
	args.injectContextRequiredVdc()
	args.injectContextDataK8sList()

	return &schema.Resource{
		ReadContext: dataSourceKubernetessRead,
		Schema:      args,
	}
}

func dataSourceKubernetessRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	vdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-035] crash via getting vdc: %s", err)
	}

	k8sList, err := vdc.GetKubernetes()
	if err != nil {
		return diag.Errorf("[ERROR-035] crash via retrieving Kubernetess: %s", err)
	}

	k8sMap := make([]map[string]interface{}, len(k8sList))
	for i, targetKubernetes := range k8sList {

		dashboard, err := targetKubernetes.GetKubernetesDashBoardUrl()
		if err != nil {
			return diag.Errorf("[ERROR-035] crash via getting Kubernetes dashboard url: %s", err)
		}

		err = targetKubernetes.GetKubernetesConfigUrl()
		if err != nil {
			return diag.Errorf("[ERROR-035] crash via creating Kubernetes config file url: %s", err)
		}

		vms := make([]*string, len(targetKubernetes.Vms))
		for i, vm := range targetKubernetes.Vms {
			vms[i] = &vm.ID
		}
		k8sMap[i] = map[string]interface{}{
			"id":                      targetKubernetes.ID,
			"name":                    targetKubernetes.Name,
			"node_cpu":                targetKubernetes.NodeCpu,
			"node_ram":                targetKubernetes.NodeRam,
			"template_id":             targetKubernetes.Template.ID,
			"node_disk_size":          targetKubernetes.NodeDiskSize,
			"nodes_count":             targetKubernetes.NodesCount,
			"user_public_key_id":      targetKubernetes.UserPublicKey,
			"node_storage_profile_id": targetKubernetes.NodeStorageProfile.ID,
			"floating":                false,
			"floating_ip":             "",
			"vms":                     vms,
			"dashboard_url":           fmt.Sprint(manager.BaseURL, *dashboard.DashBoardUrl),
			"tags":                    marshalTagNames(targetKubernetes.Tags),
		}

		if targetKubernetes.Floating != nil {
			k8sMap[i]["floating"] = true
			k8sMap[i]["floating_ip"] = targetKubernetes.Floating.IpAddress
		}
	}

	hash, err := hashstructure.Hash(k8sList, hashstructure.FormatV2, nil)
	if err != nil {
		diag.Errorf("[ERROR-035] crash via calculate hash: %s", err)
	}

	fields := map[string]interface{}{
		"id":          fmt.Sprintf("kubernetess/%d", hash),
		"vdc_id":      vdc.ID,
		"kubernetess": k8sMap,
	}

	if err = setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-035] crash via set attrs: %s", err)
	}

	return nil
}
