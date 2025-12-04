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

func resourceKubernetes() *schema.Resource {
	args := Defaults()
	args.injectCreateKubernetes()
	args.injectContextVdcById()
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
	config := struct {
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
		Tags             string `json:"tags"`
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
		Tags:             d.Get("tags").(string),
	}

	targetVdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-053]: crash via getting VDC: %s", err)
	}

	if targetVdc.Hypervisor.Type == "Vmware" && config.PlatformId == "" {
		return diag.Errorf("[ERROR-053]: field 'platform' is required for %s Hypervisor", targetVdc.Hypervisor.Type)
	}

	template, err := GetKubernetesTemplateById(d, manager, targetVdc)
	if err != nil {
		return diag.Errorf("template_id: Error getting template: %s", err)
	}

	storageProfile, err := targetVdc.GetStorageProfile(config.StorageProfileId)
	if err != nil {
		return diag.Errorf("storage_profile_id: Error storage profile %s not found", sp_id)
	}

	pubKey, err := manager.GetPublicKey(config.UserPublicKeyId)
	if err != nil {
		return diag.Errorf("storage_profile_id: Error storage profile %s not found", userPublicKey)
	}
	name := d.Get("name").(string)
	cpu := d.Get("node_cpu").(int)
	ram := d.Get("node_ram").(int)
	nodesCount := d.Get("nodes_count").(int)
	nodeDiskSize := d.Get("node_disk_size").(int)
	log.Printf(name, cpu, ram, template.Name)

	log.Printf(config.Name, config.NodeCpu, config.NodeRam, template.Name)

	newKubernetes := bcc.NewKubernetes(
		config.Name, config.NodeCpu, config.NodeRam, config.NodesCount,
		config.NodeDiskSize, template, storageProfile, pubKey.ID,
	)

	if config.PlatformId != "" {
		newKubernetes.NodePlatform, err = manager.GetPlatform(config.PlatformId)
		if err != nil {
			return diag.Errorf("template_id: Error getting template: %s", err)
		}
	}

	if config.Floating {
		_floating := "RANDOM_FIP"
		newKubernetes.Floating = &bcc.Port{IpAddress: &_floating}
	}

	newKubernetes.Tags = unmarshalTagNames(d.Get("tags"))

	if err = targetVdc.CreateKubernetes(&newKubernetes); err != nil {
		return diag.Errorf("[ERROR-053]: crash via creating Kubernetes: %s", err)
	}

	if err = newKubernetes.WaitLock(); err != nil {
		return diag.Errorf("[ERROR-053]: crash via wait lock")
	}

	d.SetId(newKubernetes.ID)

	log.Printf("[INFO-053] Kubernetes created, ID: %s", d.Id())

	return resourceKubernetesRead(ctx, d, meta)
}

func resourceKubernetesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	k8s, err := manager.GetKubernetes(d.Id())
	if err != nil {
		if err.(*bcc.ApiError).Code() == 404 {
			d.SetId("")
			return nil
		} else {
			return diag.Errorf("id: Error getting Kubernetes: %s", err)
		}
	}

	d.SetId(Kubernetes.ID)
	d.Set("name", Kubernetes.Name)
	d.Set("node_cpu", Kubernetes.NodeCpu)
	d.Set("node_ram", Kubernetes.NodeRam)
	d.Set("nodes_count", Kubernetes.NodesCount)
	d.Set("node_disk_size", Kubernetes.NodeDiskSize)
	d.Set("platform", Kubernetes.NodePlatform.ID)
	d.Set("template_id", Kubernetes.Template.ID)
	d.Set("tags", marshalTagNames(Kubernetes.Tags))

	vms := make([]*string, len(Kubernetes.Vms))
	for i, vm := range Kubernetes.Vms {
		vms[i] = &vm.ID
	}
	d.Set("vms", vms)

	vms := make([]*string, len(k8s.Vms))
	for i, vm := range k8s.Vms {
		vms[i] = &vm.ID
	}

	err = k8s.GetKubernetesConfigUrl()
	if err != nil {
		diagErr = diag.Errorf("config: Error getting Kubernetes config: %s", err)
		return
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
		"floating":                k8s.Floating != nil,
		"floating_ip":             "",
		"dashboard_url":           fmt.Sprint(manager.BaseURL, *dashboard.DashBoardUrl),
	}

	if err = setResourceDataFromMap(d, fields); err != nil {
		return diag.FromErr(err)
	}

	if k8s.Floating != nil {
		d.Set("floating_ip", k8s.Floating.IpAddress)
	}

	return nil
}

func resourceKubernetesUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	targetVdc, err := GetVdcById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-053]: err with getting 'VDC': %s ", err)
	}

	needUpdate := false

	kubernetes, err := manager.GetKubernetes(d.Id())
	if err != nil {
		return diag.Errorf("[ERROR-053]: err with getting kubernetes 'id': %s", err)
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
	needUpdate = true
	sp_id := d.Get("node_storage_profile_id").(string)
	storage_profile, err := targetVdc.GetStorageProfile(sp_id)
	if err != nil {
		return diag.Errorf("[ERROR-053]: err with getting 'storage_profile_id': %s ", sp_id)
	}

	userPublicKey := d.Get("user_public_key_id").(string)
	pub_key, err := manager.GetPublicKey(userPublicKey)
	if err != nil {
		return diag.Errorf("[ERROR-053]: err with getting 'userPublicKey': %s ", userPublicKey)
	}
	kubernetes.NodeRam = d.Get("node_ram").(int)
	kubernetes.NodeCpu = d.Get("node_cpu").(int)
	kubernetes.UserPublicKey = pub_key.ID
	kubernetes.NodeStorageProfile = storage_profile
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
		d.Set("floating", kubernetes.Floating != nil)
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
		return diag.Errorf("id: Error getting Kubernetes: %s", err)
	}

	err = kubernetes.Delete()
	if err != nil {
		return diag.Errorf("Error deleting Kubernetes: %s", err)
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

	return []*schema.ResourceData{d}, nil
}
