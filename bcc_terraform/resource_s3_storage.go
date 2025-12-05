package bcc_terraform

import (
	"context"
	"log"
	"time"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceS3Storage() *schema.Resource {
	args := Defaults()
	args.injectContextProjectById()
	args.injectCreateS3Storage()

	return &schema.Resource{
		CreateContext: resourceS3StorageCreate,
		ReadContext:   resourceS3StorageRead,
		UpdateContext: resourceS3StorageUpdate,
		DeleteContext: resourceS3StorageDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceS3StorageImport,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
		Schema: args,
	}
}

func resourceS3StorageCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	project, err := GetProjectById(d, manager)
	if err != nil {
		return diag.Errorf("project_id: Error getting Project: %s", err)
	}
	name := d.Get("name").(string)
	backend := d.Get("backend").(string)
	newS3Storage := bcc.NewS3Storage(name, backend)
	newS3Storage.Tags = unmarshalTagNames(d.Get("tags"))

	err = project.CreateS3Storage(&newS3Storage)
	if err != nil {
		return diag.Errorf("Error creating S3Storage: %s", err)
	}

	newS3Storage.WaitLock()
	d.SetId(newS3Storage.ID)
	log.Printf("[INFO] S3Storage created, ID: %s", d.Id())

	return resourceS3StorageRead(ctx, d, meta)
}

func resourceS3StorageUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	s3, err := manager.GetS3Storage(d.Id())
	if err != nil {
		return diag.Errorf("Error getting S3Storage: %s", err)
	}
	if d.HasChange("name") {
		s3.Name = d.Get("name").(string)
	}
	if d.HasChange("tags") {
		s3.Tags = unmarshalTagNames(d.Get("tags"))
	}

	err = s3.Update()
	if err != nil {
		return diag.Errorf("Error updating S3Storage: %s", err)
	}
	s3.WaitLock()
	log.Printf("[INFO] S3Storage updated, ID: %s", d.Id())

	return resourceS3StorageRead(ctx, d, meta)
}

func resourceS3StorageRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	S3Storage, err := manager.GetS3Storage(d.Id())
	if err != nil {
		if err.(*bcc.ApiError).Code() == 404 {
			d.SetId("")
			return nil
		} else {
			return diag.Errorf("id: Error getting S3Storage: %s", err)
		}
	}

	fields := map[string]interface{}{
		"name":            s3Storage.Name,
		"backend":         s3Storage.Backend,
		"project_id":      s3Storage.Project.ID,
		"client_endpoint": s3Storage.ClientEndpoint,
		"secret_key":      s3Storage.SecretKey,
		"access_key":      s3Storage.AccessKey,
		"tags":            marshalTagNames(s3Storage.Tags),
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-051]: %s3Storage", err)
	}

	return nil
}

func resourceS3StorageDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	s3_id := d.Id()
	s3, err := manager.GetS3Storage(d.Id())
	if err != nil {
		return diag.Errorf("id: Error getting S3Storage: %s", err)
	}

	err = s3.Delete()
	if err != nil {
		return diag.Errorf("Error deleting S3Storage: %s", err)
	}
	s3.WaitLock()

	d.SetId("")
	log.Printf("[INFO] S3Storage deleted, ID: %s", s3_id)

	return nil
}

func resourceS3StorageImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	manager := meta.(*CombinedConfig).Manager()
	S3Storage, err := manager.GetS3Storage(d.Id())
	if err != nil {
		d.SetId("")
		return nil, fmt.Errorf("[ERROR-051]: crash via getting S3Storage by 'id'=%s: %s", d.Id(), err)
	}

	d.SetId(S3Storage.ID)

	return []*schema.ResourceData{d}, nil
}
