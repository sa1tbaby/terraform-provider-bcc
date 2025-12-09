package bcc_terraform

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceS3StorageBucket() *schema.Resource {
	args := Defaults()
	args.injectCreateS3StorageBucket()
	args.injectContextS3StorageById()

	return &schema.Resource{
		CreateContext: resourceS3StorageBucketCreate,
		ReadContext:   resourceS3StorageBucketRead,
		UpdateContext: resourceS3StorageBucketUpdate,
		DeleteContext: resourceS3StorageBucketDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceS3StorageBucketImport,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
		Schema: args,
	}
}

var reForName = regexp.MustCompile(`^[A-z0-9\-]+$`)

func resourceS3StorageBucketCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	s3Id := d.Get("s3_storage_id").(string)
	s3, err := manager.GetS3Storage(s3Id)
	if err != nil {
		return diag.Errorf("[ERROR-052]: crash via getting S3Storage by 'id'=%s: %s", s3Id, err)
	}

	var S3StorageBucket bcc.S3StorageBucket
	if len(reForName.FindStringSubmatch(d.Get("name").(string))) > 0 {
		S3StorageBucket = bcc.NewS3StorageBucket(d.Get("name").(string))
	} else {
		return diag.Errorf("[ERROR-052]: wrong name format should be A-z, 1-0 and `-`")
	}

	err = s3.CreateBucket(&S3StorageBucket)
	if err != nil {
		return diag.Errorf("[ERROR-052]: crash via creating S3StorageBucket: %s", err)
	}

	d.SetId(S3StorageBucket.ID)
	log.Printf("[INFO-052] S3StorageBucket created, ID: %s", d.Id())

	return resourceS3StorageBucketRead(ctx, d, meta)
}

func resourceS3StorageBucketUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	s3Id := d.Get("s3_storage_id").(string)

	s3, err := manager.GetS3Storage(s3Id)
	if err != nil {
		return diag.Errorf("[ERROR-052]: crash via getting S3Storage by 'id'=%s: %s", s3Id, err)
	}

	bucket, err := s3.GetBucket(d.Id())
	if err != nil {
		return diag.Errorf("[ERROR-052]: crash via getting S3StorageBucket by 'id'=%s: %s", d.Id(), err)
	}
	if d.HasChange("name") {
		if len(reForName.FindStringSubmatch(d.Get("name").(string))) > 0 {
			bucket.Name = d.Get("name").(string)
		} else {
			return diag.Errorf("[ERROR-052]: wrong name format should be A-z, 1-0 and `-`")
		}
	}

	if err = bucket.Update(); err != nil {
		return diag.Errorf("[ERROR-052]: crash via updating S3StorageBucket: %s", err)
	}
	log.Printf("[INFO-052] S3StorageBucket updated, ID: %s", d.Id())

	return resourceS3StorageBucketRead(ctx, d, meta)
}

func resourceS3StorageBucketRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	s3Id := d.Get("s3_storage_id").(string)

	s3, err := manager.GetS3Storage(s3Id)
	if err != nil {
		return diag.Errorf("[ERROR-052]: crash via getting S3Storage by 'id'=%bucket: %bucket", s3Id, err)
	}

	bucket, err := s3.GetBucket(d.Id())
	if err != nil {
		if err.(*bcc.ApiError).Code() == 404 {
			d.SetId("")
			return nil
		} else {
			return diag.Errorf("[ERROR-052]: crash via getting S3StorageBucket by 'id'=%bucket: %bucket", d.Id(), err)
		}
	}

	fields := map[string]interface{}{
		"s_3_storage_id": s3.ID,
		"bucket_name":    bucket.Name,
		"external_name":  bucket.ExternalName,
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-052]: crash via reading S3StorageBucket: %bucket", err)
	}

	return nil
}

func resourceS3StorageBucketDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	s3Id := d.Get("s3_storage_id").(string)
	s3, err := manager.GetS3Storage(s3Id)
	if err != nil {
		return diag.Errorf("[ERROR-052]: crash via getting S3Storage by 'id'=%s: %s", s3Id, err)
	}

	bucket, err := s3.GetBucket(d.Id())
	if err != nil {
		return diag.Errorf("[ERROR-052]: crash via getting S3StorageBucket by 'id'=%s: %s", d.Id(), err)
	}

	if err = bucket.Delete(); err != nil {
		return diag.Errorf("[ERROR-052]: crash via deleting S3StorageBucket: %s", err)
	}

	d.SetId("")
	log.Printf("[INFO] S3StorageBucket deleted, ID: %s", s3Id)

	return nil
}

func resourceS3StorageBucketImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	manager := meta.(*CombinedConfig).Manager()

	id := d.Id()
	ids := strings.Split(id, ",")

	s3, err := manager.GetS3Storage(ids[0])
	if err != nil {
		d.SetId("")
		return nil, fmt.Errorf("[ERROR-052]: crash via getting S3Storage by 'id'=%s: %s", ids[0], err)
	}

	bucket, err := s3.GetBucket(ids[1])
	if err != nil {
		d.SetId("")
		return nil, fmt.Errorf("[ERROR-052]: crash via getting S3StorageBucket by 'id'=%s: %s", ids[1], err)
	}

	d.SetId(bucket.ID)
	if err = d.Set("s3_storage_id", s3.ID); err != nil {
		return nil, fmt.Errorf("[ERROR-052]: crash via setting 's3_storage_id'=%s: %s", s3.ID, err)
	}

	return []*schema.ResourceData{d}, nil
}
