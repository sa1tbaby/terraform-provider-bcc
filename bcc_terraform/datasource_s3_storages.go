package bcc_terraform

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/hashstructure/v2"
)

func dataSourceS3Storages() *schema.Resource {
	args := Defaults()
	args.injectContextRequiredProject()
	args.injectContextDataS3List()

	return &schema.Resource{
		ReadContext: dataSourceS3Read,
		Schema:      args,
	}
}

func dataSourceS3Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	project, err := GetProjectById(d, manager)
	if err != nil {
		return diag.Errorf("[ERROR-033] crash via getting project: %s", err)
	}

	s3storageList, err := project.GetS3Storages()
	if err != nil {
		return diag.Errorf("[ERROR-033] crash via retrieving storages: %s", err)
	}

	s3Map := make([]map[string]interface{}, len(s3storageList))
	for i, s3 := range s3storageList {
		s3Map[i] = map[string]interface{}{
			"id":              s3.ID,
			"name":            s3.Name,
			"backend":         s3.Backend,
			"client_endpoint": s3.ClientEndpoint,
			"access_key":      s3.AccessKey,
			"secret_key":      s3.SecretKey,
			"tags":            marshalTagNames(s3.Tags),
		}

	}

	hash, err := hashstructure.Hash(s3storageList, hashstructure.FormatV2, nil)
	if err != nil {
		diag.Errorf("[ERROR-033] crash via calculate hash: %s", err)
	}

	fields := map[string]interface{}{
		"id":          fmt.Sprintf("s3_storages/%d", hash),
		"project_id":  project.ID,
		"s3_storages": s3Map,
	}

	if err = setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-033] crash via set attrs: %s", err)
	}

	return nil
}
