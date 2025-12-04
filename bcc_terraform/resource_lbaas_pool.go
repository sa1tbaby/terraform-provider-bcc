package bcc_terraform

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/basis-cloud/bcc-go/bcc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceLbaasPool() *schema.Resource {
	args := Defaults()
	args.injectContextLbaasByID()
	args.injectCreateLbaasPool()

	return &schema.Resource{
		CreateContext: resourceLbaasPoolCreate,
		ReadContext:   resourceLbaasPoolRead,
		UpdateContext: resourceLbaasPoolUpdate,
		DeleteContext: resourceLbaasPoolDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: args,
	}
}

func resourceLbaasPoolCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	type member struct {
		Id     string `json:"id"`
		Port   int    `json:"port"`
		Weight int    `json:"weight"`
	}

	config := &struct {
		lbaasId            string
		conlimit           int
		cookieName         string
		method             string
		port               int
		protocol           string
		sessionPersistence string
		members            []*member
	}{
		lbaasId:            d.Get("lbaas_id").(string),
		conlimit:           d.Get("conlimit").(int),
		cookieName:         d.Get("cookie_name").(string),
		method:             d.Get("method").(string),
		port:               d.Get("port").(int),
		protocol:           d.Get("protocol").(string),
		sessionPersistence: d.Get("session_persistence").(string),
		members:            d.Get("member").([]*member),
	}

	lbaas, err := manager.GetLoadBalancer(config.lbaasId)
	members := make([]*bcc.PoolMember, len(config.members))
	if err != nil {
		return diag.Errorf("id: Error getting Lbaas: %s", err)
	}
	// Get members
	membersCount := d.Get("member.#").(int)
	members := make([]*bcc.PoolMember, membersCount)

	for i, item := range config.members {
		vm, err := manager.GetVm(item.Id)
		if err != nil {
			return diag.Errorf("vm_id: Error getting vm: %s", err)
		}

		newMember := bcc.NewLoadBalancerPoolMember(item.Port, item.Weight, vm)
		members[i] = &newMember
	}

	newPool := bcc.NewLoadBalancerPool(
		*lbaas,
		d.Get("port").(int),
		d.Get("connlimit").(int),
		members,
		d.Get("method").(string),
		d.Get("protocol").(string),
		d.Get("session_persistence").(string),
		config.method, config.protocol, config.sessionPersistence,
	)
	err = lbaas.CreatePool(&newPool)
	if err != nil {
		return diag.Errorf("id: Error creating Lbaas pool: %s", err)
	}
	if err = lbaas.WaitLock(); err != nil {
		return diag.Errorf("[ERROR-050]: %s", err)
	}

	d.SetId(newPool.ID)
	return resourceLbaasPoolRead(ctx, d, meta)
}

func resourceLbaasPoolRead(ctx context.Context, d *schema.ResourceData, meta interface{}) (diagErr diag.Diagnostics) {
	manager := meta.(*CombinedConfig).Manager()
	lbaasId := d.Get("lbaas_id").(string)

	lbaas, err := manager.GetLoadBalancer(lbaasId)
	if err != nil {
		return diag.Errorf("id: Error getting Lbaas: %s", err)
	}

	lbaasPool, err := lbaas.GetLoadBalancerPool(d.Id())
	if err != nil {
		if err.(*bcc.ApiError).Code() == 404 {
			d.SetId("")
			return nil
		} else {
			return diag.Errorf("Error getting LbaasPool: %s", err)
		}
	}

	flattenedPools := make([]map[string]interface{}, len(lbaasPool.Members))
	for i, member := range lbaasPool.Members {
		flattenedPools[i] = map[string]interface{}{
			"id":     member.ID,
			"port":   member.Port,
			"weight": member.Weight,
			"vm_id":  member.Vm.ID,
		}
	}

	fields := map[string]interface{}{
		"lbaas_id":            lbaas.ID,
		"port":                lbaasPool.Port,
		"connlimit":           lbaasPool.Connlimit,
		"method":              lbaasPool.Method,
		"protocol":            lbaasPool.Protocol,
		"session_persistence": lbaasPool.SessionPersistence,
		"members":             flattenedPools,
		"cookie_name":         lbaas.Name,
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-050] crash via reading LbaasPool: %s", err)
	}

	return
}

func resourceLbaasPoolUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()
	lbaasPoolId := d.Id()
	lbaasId := d.Get("lbaas_id").(string)

	lbaas, err := manager.GetLoadBalancer(d.Get("lbaas_id").(string))
	if err != nil {
		return diag.Errorf("id: Error getting Lbaas: %s", err)
	}

	lbaasPool, err := lbaas.GetLoadBalancerPool(d.Id())
	if err != nil {
		return diag.Errorf("Error getting LbaasPool: %s", err)
	}

	if d.HasChange("port") {
		lbaasPool.Port = d.Get("port").(int)
	}
	if d.HasChange("connlimit") {
		lbaasPool.Connlimit = d.Get("connlimit").(int)
	}
	if d.HasChange("method") {
		lbaasPool.Method = d.Get("method").(string)
	}
	if d.HasChange("protocol") {
		lbaasPool.Protocol = d.Get("protocol").(string)
	}
	if d.HasChange("session_persistence") {
		sessionPersistence := d.Get("session_persistence").(string)
		lbaasPool.SessionPersistence = &sessionPersistence
	}
	if d.HasChange("member") {
		membersCount := d.Get("member.#").(int)
		members := make([]*bcc.PoolMember, membersCount)
		for i := 0; i < membersCount; i++ {
			memberPrefix := fmt.Sprint("member.", i)
			member := d.Get(memberPrefix).(map[string]interface{})
			vm_id := member["vm_id"].(string)
			port := member["port"].(int)
			weight := member["weight"].(int)

			vm, err := manager.GetVm(vm_id)
			if err != nil {
				return diag.Errorf("vm_id: Error getting vm: %s", err)
			}

			newMember := bcc.NewLoadBalancerPoolMember(port, weight, vm)
			members[i] = &newMember
		}
		lbaasPool.Members = members
	}
	err = lbaas.UpdatePool(&lbaasPool)
	if err != nil {
		return diag.Errorf("Error updating Lbaas pool: %s", err)
	}
	lbaas.WaitLock()

	return resourceLbaasPoolRead(ctx, d, meta)
}

func resourceLbaasPoolDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	lbaas, err := manager.GetLoadBalancer(d.Get("lbaas_id").(string))
	if err != nil {
		return diag.Errorf("Error getting LbaasPool: %s", err)
	}

	_, err = lbaas.GetLoadBalancerPool(d.Id())
	if err != nil {
		return diag.Errorf("[ERROR-050] crash via getting LbaasPool: %s", err)
	}

	if err := lbaas.DeletePool(d.Id()); err != nil {
		return diag.Errorf("[ERROR-050] crash via getting LbaasPool: %s", err)
	}
	lbaas.WaitLock()

	d.SetId("")
	log.Printf("[INFO] LbaasPool deleted, ID: %s", lbaasId)

	return nil
}
