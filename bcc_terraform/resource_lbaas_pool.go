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
			StateContext: resourceLbaasPoolImport,
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

	lbaasId := d.Get("lbaas_id").(string)
	connlimit := d.Get("connlimit").(int)
	cookieName := d.Get("cookie_name")
	method := d.Get("method").(string)
	port := d.Get("port").(int)
	protocol := d.Get("protocol").(string)
	sessionPersistence := d.Get("session_persistence").(string)
	members := d.Get("member").([]interface{})

	lbaas, err := manager.GetLoadBalancer(lbaasId)
	if err != nil {
		return diag.Errorf("[ERROR-050]: crash via getting Lbaas: %s", err)
	}

	poolMembers := make([]*bcc.PoolMember, len(members))
	for i, item := range members {
		_item := item.(map[string]interface{})
		vm, err := manager.GetVm(_item["vm_id"].(string))
		if err != nil {
			return diag.Errorf("[ERROR-050]: crash via getting vm by id: %s", err)
		}

		tmpVm := bcc.TmpVm{
			ID: vm.ID, Name: vm.Name, Cpu: vm.Cpu, Ram: vm.Ram, Power: vm.Power, Platform: vm.Platform.ID, Vdc: vm.Vdc,
		}

		newMember := bcc.NewLoadBalancerPoolMember(_item["port"].(int), _item["weight"].(int), &tmpVm)
		poolMembers[i] = &newMember
	}

	newPool := bcc.NewLoadBalancerPool(
		*lbaas, port, connlimit, poolMembers,
		method, protocol, sessionPersistence, cookieName,
	)

	err = lbaas.CreatePool(&newPool)
	if err != nil {
		return diag.Errorf("[ERROR-050]: crash via creating Lbaas pool: %s", err)
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
		return diag.Errorf("[ERROR-050]: crash via getting lbaas by id: %s", err)
	}

	lbaasPool, err := lbaas.GetLoadBalancerPool(d.Id())
	if err != nil {
		if err.(*bcc.ApiError).Code() == 404 {
			d.SetId("")
			return nil
		} else {
			return diag.Errorf("[ERROR-050] crash via getting LbaasPool for Read: %s", err)
		}
	}

	flattenedPools := make([]map[string]interface{}, len(lbaasPool.Members))
	for i, member := range lbaasPool.Members {
		flattenedPools[i] = map[string]interface{}{
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
		"member":              flattenedPools,
		"cookie_name":         lbaas.Name,
	}

	if err := setResourceDataFromMap(d, fields); err != nil {
		return diag.Errorf("[ERROR-050] crash via reading LbaasPool: %s", err)
	}

	return
}

func resourceLbaasPoolUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	lbaas, err := manager.GetLoadBalancer(d.Get("lbaas_id").(string))
	if err != nil {
		return diag.Errorf("[ERROR-050]: crash via getting lbaas by id: %s", err)
	}

	lbaasPool, err := lbaas.GetLoadBalancerPool(d.Id())
	if err != nil {
		return diag.Errorf("[ERROR-050] crash via getting LbaasPool for Update: %s", err)
	}

	if d.HasChange("port") {
		lbaasPool.Port = d.Get("port").(int)
	}
	if d.HasChange("connlimit") {
		lbaasPool.Connlimit = d.Get("connlimit").(int)
	}
	if d.HasChange("cookie_name") {
		if d.Get("cookie_name") != nil {
			cookieName := d.Get("cookie_name").(string)
			lbaasPool.CookieName = &cookieName
		}
	}
	if d.HasChange("method") {
		lbaasPool.Method = d.Get("method").(string)
	}
	if d.HasChange("protocol") {
		lbaasPool.Protocol = d.Get("protocol").(string)
	}
	if d.HasChange("session_persistence") {
		lbaasPool.SessionPersistence = d.Get("session_persistence").(string)
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
				return diag.Errorf("[ERROR-050]: crash via getting vm by id: %s", err)
			}
			tmpVm := bcc.TmpVm{
				ID: vm.ID, Name: vm.Name, Cpu: vm.Cpu, Ram: vm.Ram, Power: vm.Power, Platform: vm.Platform.ID, Vdc: vm.Vdc,
			}

			newMember := bcc.NewLoadBalancerPoolMember(port, weight, &tmpVm)
			members[i] = &newMember
		}
		lbaasPool.Members = members
	}
	err = lbaas.UpdatePool(&lbaasPool)
	if err != nil {
		return diag.Errorf("[ERROR-050]: crash via updating Lbaas lbaasPool: %s", err)
	}
	if err = lbaas.WaitLock(); err != nil {
		return diag.Errorf("[ERROR-050]: %s", err)
	}

	return resourceLbaasPoolRead(ctx, d, meta)
}

func resourceLbaasPoolDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	manager := meta.(*CombinedConfig).Manager()

	lbaas, err := manager.GetLoadBalancer(d.Get("lbaas_id").(string))
	if err != nil {
		return diag.Errorf("[ERROR-050]: crash via getting lbaas by id: %s", err)
	}

	_, err = lbaas.GetLoadBalancerPool(d.Id())
	if err != nil {
		return diag.Errorf("[ERROR-050] crash via getting LbaasPool for delete: %s", err)
	}

	if err := lbaas.DeletePool(d.Id()); err != nil {
		return diag.Errorf("[ERROR-050] crash via deletting LbaasPool: %s", err)
	}
	lbaas.WaitLock()

	log.Printf("[INFO-050] LbaasPool deleted, ID: %s", d.Get("lbaas_id").(string))
	d.SetId("")

	return nil
}

func resourceLbaasPoolImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	manager := meta.(*CombinedConfig).Manager()

	id := d.Id()
	ids := strings.Split(id, ",")

	lbaas, err := manager.GetLoadBalancer(ids[0])
	if err != nil {
		return nil, fmt.Errorf("[ERROR-050]: crash via getting lbaas by Id: %s", err)
	}

	lbaasPool, err := lbaas.GetLoadBalancerPool(ids[1])
	if err != nil {
		return nil, fmt.Errorf("[ERROR-050]: crash via getting lbaasPool for import %s", err)
	}

	d.SetId(lbaasPool.ID)
	if err := d.Set("lbaas_id", lbaas.ID); err != nil {
		return nil, fmt.Errorf("[ERROR-050]: crasg via setting lbaas_id: %s", err)
	}

	return []*schema.ResourceData{d}, nil
}
