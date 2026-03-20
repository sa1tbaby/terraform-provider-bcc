package bcc_terraform

import (
	"context"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceS3Storage() *schema.Resource {
	args := Defaults()
	args.injectContextRequiredProject()
	args.injectContextDataS3()
	args.injectContextGetS3()

	return &schema.Resource{
		ReadContext: dataSourceS3StorageRead,
		Schema:      args,
	}
}

func dataSourceS3StorageRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	target, err := checkDatasourceNameOrId(d)
	if err != nil {
		return diag.Errorf("[ERRORR-032] crash via getting s3 storage: %s", err)
	}

	var s3Storage *bcc.S3Storage
	if target == "id" {
		s3StorageId := d.Get("id").(string)
		s3Storage, err = manager.GetS3Storage(s3StorageId)
		if err != nil {
			return diag.Errorf("[ERRORR-032] crash via getting storage by id: %s", err)
		}
	} else {
		s3Storage, err = GetS3ByName(d, manager)
		if err != nil {
			return diag.Errorf("[ERRORR-032] crash via getting storage by name: %s", err)
		}
	}

	fields := map[string]interface{}{
		"id":              s3Storage.ID,
		"name":            s3Storage.Name,
		"backend":         s3Storage.Backend,
		"client_endpoint": s3Storage.ClientEndpoint,
		"access_key":      s3Storage.AccessKey,
		"secret_key":      s3Storage.SecretKey,
		"tags":            marshalTagNames(s3Storage.Tags),
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERRORR-032] crash via set attrs: %s", err)
	}

	return nil
}
