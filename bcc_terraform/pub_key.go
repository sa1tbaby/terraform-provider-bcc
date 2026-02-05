package bcc_terraform

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

func (args *Arguments) injectContextPublicKeyById() {
	args.merge(Arguments{
		"pub_key_id": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "id of the Public Key",
		},
	})
}

func (args *Arguments) injectContextGetPublicKey() {
	args.merge(Arguments{
		"name": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "name of the Public Key",
		},
		"id": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "id of the Public Key",
		},
	})
}

func (args *Arguments) injectResultPublicKey() {
	args.merge(Arguments{
		"id": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Public Key identifier",
		},
		"name": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Public Key name",
		},
		"fingerprint": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Public Key fingerprint",
		},
		"public_key": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "public key text",
		},
	})
}

func (args *Arguments) injectResultListPublicKeys() {
	s := Defaults()
	s.injectResultPublicKey()

	args.merge(Arguments{
		"public_keys": {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: s,
			},
		},
	})
}
