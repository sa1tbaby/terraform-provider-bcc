package bcc_terraform

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func (args *Arguments) injectContextGetKubernetes() {
	args.merge(Arguments{
		"name": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "kubernetes name",
		},
		"id": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "kubernetes identifier",
		},
	})
}

func (args *Arguments) injectContextKubernetesById() {
	args.merge(Arguments{
		"kubernetes_id": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "id of the kubernetes",
		},
	})
}

func (args *Arguments) injectCreateKubernetes() {
	args.merge(Arguments{
		"name": {
			Type:     schema.TypeString,
			Required: true,
			ValidateFunc: validation.All(
				validation.NoZeroValues,
				validation.StringLenBetween(1, 100),
			),
			Description: "name of the kubernetes",
		},
		"platform": {
			Type:        schema.TypeString,
			Optional:    true,
			Default:     "",
			ForceNew:    true,
			Description: "type of cpu platform",
		},
		"node_cpu": {
			Type:         schema.TypeInt,
			Required:     true,
			ValidateFunc: validation.IntBetween(1, 128),
			Description:  "the number of virtual cpus",
		},
		"node_ram": {
			Type:        schema.TypeInt,
			Required:    true,
			Description: "memory of the kubernetes in gigabytes",
		},
		"floating": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "enable floating ip for the kubernetes",
		},
		"floating_ip": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "floating ip for the kubernetes. May be omitted",
		},
		"node_disk_size": {
			Type:        schema.TypeInt,
			Required:    true,
			Description: "size in gb for the vms disk attached to kubernetes.",
		},
		"nodes_count": {
			Type:        schema.TypeInt,
			Required:    true,
			Description: "count of vms attached to kubernetes",
		},
		"user_public_key_id": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "pub key id for vms attached to kubernetes. Value used only in create and not reading from api",
		},
		"node_storage_profile_id": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "storage_profile_id for vms disks attached to kubernetes.",
		},
		"vms": {
			Type:        schema.TypeSet,
			Optional:    true,
			Computed:    true,
			MinItems:    1,
			Description: "List of Vms connected to the kubernetes",
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"dashboard_url": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "Kubernetes dashboard url",
		},
		"tags": newTagNamesResourceSchema("tags of the Kubernetes"),
	})
}

func (args *Arguments) injectResultKubernetes() {
	args.merge(Arguments{
		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "kubernetes identifier",
		},
		"name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "kubernetes name",
		},
		"node_cpu": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "number of virtual CPUs per cluster node",
		},
		"node_ram": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "amount of RAM per cluster node in GB",
		},
		"template_id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Kubernetes template identifier",
		},
		"floating": {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "whether the cluster has a public IP address",
		},
		"floating_ip": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "cluster public IP address, if available",
		},
		"nodes_count": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "count of vms attached to kubernetes",
		},
		"node_disk_size": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "cluster node disk size in GB, specified at cluster creation",
		},
		"user_public_key_id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "cluster public key",
		},
		"node_storage_profile_id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "storage profile identifier for cluster node disks",
		},
		"vms": {
			Type:        schema.TypeSet,
			Computed:    true,
			Description: "list of server id (String) identifiers belonging to the cluster",
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"dashboard_url": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "cluster management dashboard URL",
		},
	})
}

func (args *Arguments) injectResultListKubernetes() {
	s := Defaults()
	s.injectResultKubernetes()

	args.merge(Arguments{
		"kubernetess": {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: s,
			},
		},
	})
}
