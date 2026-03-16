package bcc_terraform

import (
	"context"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourcePublicKey() *schema.Resource {
	args := Defaults()
	args.injectContextAccountById()
	args.injectResultPublicKey()
	args.injectContextGetPublicKey()

	return &schema.Resource{
		ReadContext: dataSourcePublicKeyRead,
		Schema:      args,
	}
}

func dataSourcePublicKeyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	target, err := checkDatasourceNameOrId(d)
	if err != nil {
		return diag.Errorf("[ERROR-038] crash via getting PublicKey: %s", err)
	}

	var publicKey *bcc.PubKey
	if target == "id" {
		pubKeyId := d.Get("id").(string)
		publicKey, err = manager.GetPublicKey(pubKeyId)
		if err != nil {
			return diag.Errorf("[ERROR-038] crash via getting PublicKey by id=%s: %s", pubKeyId, err)
		}
	} else {
		publicKey, err = GetPubKeyByName(d, manager)
		if err != nil {
			return diag.Errorf("Error getting PublicKey by name: %s", err)
		}
	}

	flatten := map[string]interface{}{
		"id":          publicKey.ID,
		"name":        publicKey.Name,
		"public_key":  publicKey.Fingerprint,
		"fingerprint": publicKey.PublicKey,
	}

	if err := setResourceDataFromMap(d, flatten); err != nil {
		return diag.Errorf("[ERROR-038] crash via set attrs: %s", err)
	}

	return nil
}
