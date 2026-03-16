package bcc_terraform

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceKubernetes() *schema.Resource {
	args := Defaults()
	args.injectContextResourceK8s()
	args.injectContextRequiredVdc()
	args.injectContextKubernetesTemplateById()

	return &schema.Resource{
		CreateContext: resourceKubernetesCreate,
		ReadContext:   resourceKubernetesRead,
		UpdateContext: resourceKubernetesUpdate,
		DeleteContext: resourceKubernetesDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceKubernetesImport,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
		Schema: args,
	}
}

func resourceKubernetesCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	fields := struct {
		Name             string `json:"name"`
		NodeCpu          int    `json:"node_cpu"`
		NodeRam          int    `json:"node_ram"`
		NodesCount       int    `json:"nodes_count"`
		NodeDiskSize     int    `json:"node_disk_size"`
		TemplateId       string `json:"template_id"`
		Floating         bool   `json:"floating"`
		PlatformId       string `json:"platform_id"`
		UserPublicKeyId  string `json:"user_public_key_id"`
		StorageProfileId string `json:"node_storage_profile_id"`
	}{
		Name:             d.Get("name").(string),
		NodeCpu:          d.Get("node_cpu").(int),
		NodeRam:          d.Get("node_ram").(int),
		NodesCount:       d.Get("nodes_count").(int),
		NodeDiskSize:     d.Get("node_disk_size").(int),
		Floating:         d.Get("floating").(bool),
		TemplateId:       d.Get("template_id").(string),
		PlatformId:       d.Get("platform").(string),
		UserPublicKeyId:  d.Get("user_public_key_id").(string),
		StorageProfileId: d.Get("node_storage_profile_id").(string),
	}

	vdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-053]: crash via getting VDC: %s", err)
	}

	if strings.EqualFold(vdc.Hypervisor.Type, "Vmware") && fields.PlatformId == "" {
		return diag.Errorf("[ERROR-053]: field 'platform' is required for %s Hypervisor", vdc.Hypervisor.Type)
	}

	template, err := GetKubernetesTemplateById(d, manager, vdc)
	if err != nil {
		return diag.Errorf("[ERROR-053]: crash via getting k8s template: %s", err)
	}

	storageProfile, err := vdc.GetStorageProfile(fields.StorageProfileId)
	if err != nil {
		return diag.Errorf("[ERROR-053]: storage profile %s not found", fields.StorageProfileId)
	}

	pubKey, err := manager.GetPublicKey(fields.UserPublicKeyId)
	if err != nil {
		return diag.Errorf("[ERROR-053]: user public key %s not found", fields.UserPublicKeyId)
	}

	log.Printf(fields.Name, fields.NodeCpu, fields.NodeRam, template.Name)

	newKubernetes := bcc.NewKubernetes(
		fields.Name, fields.NodeCpu, fields.NodeRam, fields.NodesCount, fields.NodeDiskSize,
		nil, template, storageProfile, pubKey.ID, nil,
	)

	if fields.PlatformId != "" {
		newKubernetes.NodePlatform, err = manager.GetPlatform(fields.PlatformId)
		if err != nil {
			return diag.Errorf("[ERROR-053]: crash via getting template: %s", err)
		}
	}

	if fields.Floating {
		_floating := "RANDOM_FIP"
		newKubernetes.Floating = &bcc.Port{IpAddress: &_floating}
	}

	newKubernetes.Tags = unmarshalTagNames(d.Get("tags"))

	if err = vdc.CreateKubernetes(&newKubernetes); err != nil {
		return diag.Errorf("[ERROR-053]: crash via creating Kubernetes: %s", err)
	}

	if err = newKubernetes.WaitLock(); err != nil {
		return diag.Errorf("[ERROR-053]: crash via wait lock")
	}

	d.SetId(newKubernetes.ID)
	d.Set("user_public_key_id", pubKey.PublicKey)
	log.Printf("[INFO] Kubernetes created, ID: %s", d.Id())

	return resourceKubernetesRead(ctx, d, meta)
}

func resourceKubernetesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	k8s, err := manager.GetKubernetes(d.Id())
	if err != nil {
		return resourceReadCheck(d, err, "[ERROR-053]:")
	}

	vms := make([]*string, len(k8s.Vms))
	for i, vm := range k8s.Vms {
		vms[i] = &vm.ID
	}

	err = k8s.GetKubernetesConfigUrl()
	if err != nil {
		return diag.Errorf("[ERROR-053]: crash via getting k8s config: %s", err)
	}

	dashboard, err := k8s.GetKubernetesDashBoardUrl()
	if err != nil {
		return diag.Errorf("[ERROR-053]: crash via getting k8s dashboard url: %s", err)
	}

	fields := map[string]interface{}{
		"vdc_id":                  k8s.Vdc.ID,
		"name":                    k8s.Name,
		"node_cpu":                k8s.NodeCpu,
		"node_ram":                k8s.NodeRam,
		"nodes_count":             k8s.NodesCount,
		"node_disk_size":          k8s.NodeDiskSize,
		"platform":                k8s.NodePlatform.ID,
		"template_id":             k8s.Template.ID,
		"node_storage_profile_id": k8s.NodeStorageProfile.ID,
		"tags":                    marshalTagNames(k8s.Tags),
		"vms":                     vms,
		"floating":                false,
		"floating_ip":             "",
		"dashboard_url":           fmt.Sprint(manager.BaseURL, *dashboard.DashBoardUrl),
	}

	if k8s.Floating != nil {
		fields["floating"] = true
		fields["floating_ip"] = k8s.Floating.IpAddress
	}

	if err = setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-053]: crash via set attrs: %s", err)
	}

	return nil
}

func resourceKubernetesUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	vdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-053]: crash via getting VDC: %s", err)
	}

	needUpdate := false

	kubernetes, err := manager.GetKubernetes(d.Id())
	if err != nil {
		return diag.Errorf("[ERROR-053]: err with getting kubernetes 'id'=%s: %s", d.Id(), err)
	}

	// Detect Kubernetes changes
	if d.HasChange("name") {
		needUpdate = true
		kubernetes.Name = d.Get("name").(string)
	}
	if d.HasChange("tags") {
		needUpdate = true
		kubernetes.Tags = unmarshalTagNames(d.Get("tags"))
	}

	spId := d.Get("node_storage_profile_id").(string)
	storageProfile, err := vdc.GetStorageProfile(spId)
	if err != nil {
		return diag.Errorf("[ERROR-053]: err with getting 'storage_profile_id': %s ", spId)
	}

	userPublicKey := d.Get("user_public_key_id").(string)
	pubKey, err := manager.GetPublicKey(userPublicKey)
	if err != nil {
		return diag.Errorf("[ERROR-053]: err with getting 'userPublicKey': %s ", userPublicKey)
	}

	kubernetes.NodeRam = d.Get("node_ram").(int)
	kubernetes.NodeCpu = d.Get("node_cpu").(int)
	kubernetes.UserPublicKey = pubKey.ID
	kubernetes.NodeStorageProfile = storageProfile
	kubernetes.NodeDiskSize = d.Get("node_disk_size").(int)
	kubernetes.NodesCount = d.Get("nodes_count").(int)

	ncOld, ncNew := d.GetChange("nodes_count")
	if ncOld.(int) > ncNew.(int) {
		return diag.Errorf("[ERROR-053]: cannot down scale Kubernetes 'nodes_count'")
	}

	if d.HasChange("floating") {
		needUpdate = true
		if !d.Get("floating").(bool) {
			kubernetes.Floating = &bcc.Port{IpAddress: nil}
		} else {
			kubernetes.Floating = &bcc.Port{ID: "RANDOM_FIP"}
		}
	}

	if needUpdate {
		if err := repeatOnError(kubernetes.Update, kubernetes); err != nil {
			return diag.Errorf("[ERROR-053]: err with updating Kubernetes: %s", err)
		}
	}

	return resourceKubernetesRead(ctx, d, meta)
}

func resourceKubernetesDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	kubernetes, err := manager.GetKubernetes(d.Id())
	if err != nil {
		return diag.Errorf("[ERROR-053]: err with getting kubernetes 'id'=%s: %s", d.Id(), err)
	}

	err = kubernetes.Delete()
	if err != nil {
		return diag.Errorf("[ERROR-053]: crash via deleting Kubernetes: %s", err)
	}
	kubernetes.WaitLock()

	return nil
}

func resourceKubernetesImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	manager := meta.(*CombinedConfig).Manager()
	k8s, err := manager.GetKubernetes(d.Id())
	if err != nil {
		if err.(*bcc.ApiError).Code() == 404 {
			d.SetId("")
			return nil, nil
		} else {
			return nil, fmt.Errorf("[ERROR-053]: err with getting kubernetes 'id': %s", err)
		}
	}

	d.SetId(k8s.ID)
	d.Set("user_public_key_id", k8s.UserPublicKey)

	return []*schema.ResourceData{d}, nil
}
