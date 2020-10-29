package multipass

import (
	"context"
	"encoding/json"
	"log"
	"os/exec"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Disk - types for proper json import
type Disk struct {
	Total string `json:"total"`
	Used  string `json:"used"`
}

// Memory - types for proper json import
type Memory struct {
	Total int64 `json:"total"`
	Used  int64 `json:"used"`
}

// Mount - types for proper json import
type Mount struct {
	GIDMappings []string `json:"gid_mappings"`
	SourcePath  string   `json:"source_path"`
	UIDMappings []string `json:"uid_mappings"`
}

// Instance - types for proper json import
type Instance struct {
	Disks        map[string]*Disk `json:"disks"`
	ImageHash    string           `json:"image_hash"`
	ImageRelease string           `json:"image_release"`
	Ipv4         []string         `json:"ipv4"`
	// Load []string `json:"load"`
	Memory  *Memory           `json:"memory"`
	Mounts  map[string]*Mount `json:"mounts"`
	Release string            `json:"release"`
	State   string            `json:"state"`
}

// Info - types for proper json import
type Info struct {
	Errors []string             `json:"errors"`
	Info   map[string]*Instance `json:"info"`
}

func dataSourceInstance() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceInstanceRead,
		Schema: map[string]*schema.Schema{
			"instances": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						// re-define another perfectly good object.
						"id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"disks": {
							Type:     schema.TypeSet,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"device": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"total": {
										Type:     schema.TypeString,
										Computed: true,
									},
									// "used": {
									// 	Type:     schema.TypeString,
									// 	Computed: true,
									// },
								},
							},
						},
						"image_hash": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"image_release": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"ipv4": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"memory_total": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"mounts": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"mount_path": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"gid_mappings": {
										Type: schema.TypeList,
										Elem: &schema.Schema{
											Type:     schema.TypeString,
											Computed: true,
										},
									},
									"uid_mappings": {
										Type: schema.TypeList,
										Elem: &schema.Schema{
											Type:     schema.TypeString,
											Computed: true,
										},
									},
								},
							},
						},
						"release": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"state": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

// FML: why can't tf just adopt complex types. This need to convert to flat lists is some bull.
// Basically I need to convert all my maps to sets or lists?
// sets can have their own schemas
// lists are just simple types

func dataSourceInstanceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	cmd := exec.Command("multipass", "info", "--all", "--format", "json")
	out, err := cmd.StdoutPipe()
	if err != nil {
		return diag.FromErr(err)
	}

	err = cmd.Start()
	if err != nil {
		return diag.FromErr(err)
	}

	// decode into Info Object - will at least validate the json schema
	info := Info{}
	err = json.NewDecoder(out).Decode(&info)
	if err != nil {
		return diag.Errorf("multipass info json decode - %v", err)
	}

	err = cmd.Wait()
	if err != nil {
		return diag.FromErr(err)
	}

	pretty, err := json.MarshalIndent(info.Info, "", "  ")
	log.Printf("[DEBUG] %s\n", pretty)

	instances := make([]interface{}, len(info.Info)-1)
	for k, v := range info.Info {
		i := make(map[string]interface{})
		i["id"] = k
		i["disks"] = flattenDisks(v.Disks)
		i["image_hash"] = v.ImageHash
		i["image_release"] = v.ImageRelease
		i["ipv4"] = v.Ipv4
		i["memory_total"] = v.Memory.Total
		i["mounts"] = flattenMounts(v.Mounts)
		i["release"] = v.Release
		i["state"] = v.State

		instances = append(instances, i)
	}

	pretty, err = json.MarshalIndent(instances, "", "  ")
	log.Printf("[DEBUG] %s\n", pretty)

	err = d.Set("instances", instances)
	if err != nil {
		return diag.Errorf("Setting instance value - %v", err)
	}

	// always set a id?
	d.SetId("0")

	return diags
}

func flattenDisks(disks map[string]*Disk) []interface{} {
	if disks != nil {
		tfDisks := make([]interface{}, len(disks)-1)

		for k, v := range disks {
			tfDisk := make(map[string]interface{})

			tfDisk["device"] = k
			tfDisk["total"] = v.Total
			// tfDisk["used"] = v.Used

			tfDisks = append(tfDisks, tfDisk)
		}

		return tfDisks
	}

	return make([]interface{}, 0)
}

func flattenMounts(mounts map[string]*Mount) []interface{} {
	if mounts != nil {
		tfMounts := make([]interface{}, len(mounts)-1)

		for k, v := range mounts {
			tfMount := make(map[string]interface{})

			tfMount["mount_path"] = k
			tfMount["gid_mappings"] = v.GIDMappings
			tfMount["uid_mappings"] = v.UIDMappings
			tfMount["source_path"] = v.SourcePath

			// tfDisk["used"] = v.Used

			tfMounts = append(tfMounts, tfMount)
		}

		return tfMounts
	}

	return make([]interface{}, 0)
}
