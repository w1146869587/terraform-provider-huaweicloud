package huaweicloud

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/huaweicloud/golangsdk"
	"github.com/huaweicloud/golangsdk/openstack/identity/v3/agency"
	"github.com/huaweicloud/golangsdk/openstack/identity/v3/domains"
	sdkprojects "github.com/huaweicloud/golangsdk/openstack/identity/v3/projects"
	sdkroles "github.com/huaweicloud/golangsdk/openstack/identity/v3/roles"
)

func resourceIAMAgencyV3() *schema.Resource {
	return &schema.Resource{
		Create: resourceIAMAgencyV3Create,
		Read:   resourceIAMAgencyV3Read,
		Update: resourceIAMAgencyV3Update,
		Delete: resourceIAMAgencyV3Delete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"agency_domain_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"duration": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"expire_time": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"create_time": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"project_role": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"roles": {
							Type:     schema.TypeSet,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"project": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"domain_roles": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func agencyClient(d *schema.ResourceData, config *Config) (*golangsdk.ServiceClient, error) {
	c, err := config.loadIAMV3Client(GetRegion(d, config))
	if err != nil {
		return nil, err
	}
	c.Endpoint = "https://iam.myhwclouds.com:443/v3.0/"
	return c, nil
}

func listProjectsOfDomain(domainID string, client *golangsdk.ServiceClient) (map[string]string, error) {
	old := client.Endpoint
	defer func() { client.Endpoint = old }()
	client.Endpoint = "https://iam.myhwclouds.com:443/v3/"

	opts := sdkprojects.ListOpts{
		DomainID: domainID,
	}
	allPages, err := sdkprojects.List(client, &opts).AllPages()
	if err != nil {
		return nil, err
	}

	all, err := sdkprojects.ExtractProjects(allPages)
	if err != nil {
		return nil, err
	}

	r := make(map[string]string, len(all))
	for _, item := range all {
		r[item.Name] = item.ID
	}
	log.Printf("[DEBUG] projects = %#v\n", r)
	return r, nil
}

func listRolesOfDomain(domainID string, client *golangsdk.ServiceClient) (map[string]string, error) {
	old := client.Endpoint
	defer func() { client.Endpoint = old }()
	client.Endpoint = "https://iam.myhwclouds.com:443/v3/"

	opts := sdkroles.ListOpts{
		DomainID: domainID,
	}
	allPages, err := sdkroles.List(client, &opts).AllPages()
	if err != nil {
		return nil, err
	}

	all, err := sdkroles.ExtractRoles(allPages)
	if err != nil {
		return nil, err
	}
	log.Printf("[DEBUG] list roles = %#v, len=%d\n", all, len(all))
	if len(all) == 0 {
		return nil, nil
	}

	r := make(map[string]string, len(all))
	for _, item := range all {
		dn, ok := item.Extra["display_name"].(string)
		if ok {
			r[dn] = item.ID
		} else {
			log.Printf("[DEBUG] Can not retrieve role:%#v", item)
		}
	}
	log.Printf("[DEBUG] list roles = %#v, len=%d\n", r, len(r))
	return r, nil
}

func allRolesOfDomain(domainID string, client *golangsdk.ServiceClient) (map[string]string, error) {
	roles, err := listRolesOfDomain("", client)
	if err != nil {
		return nil, fmt.Errorf("Error listing global roles, err=%s", err)
	}
	if roles == nil {
		roles = map[string]string{}
	}

	customRoles, err := listRolesOfDomain(domainID, client)
	if err != nil {
		return nil, fmt.Errorf("Error listing domain's custom roles, err=%s", err)
	}
	if customRoles != nil {
		for k, v := range customRoles {
			roles[k] = v
		}
	}

	return roles, nil
}

func getDomainID(config *Config, client *golangsdk.ServiceClient) (string, error) {
	if config.DomainID != "" {
		return config.DomainID, nil
	}

	name := config.DomainName
	if name == "" {
		return "", fmt.Errorf("Error getting domain id, the domain name was required")
	}

	old := client.Endpoint
	defer func() { client.Endpoint = old }()
	client.Endpoint = "https://iam.myhwclouds.com:443/v3/auth/"

	opts := domains.ListOpts{
		Name: name,
	}
	allPages, err := domains.List(client, &opts).AllPages()
	if err != nil {
		return "", err
	}

	all, err := domains.ExtractDomains(allPages)
	if err != nil {
		return "", err
	}

	count := len(all)
	switch count {
	case 0:
		err := &golangsdk.ErrResourceNotFound{}
		err.ResourceType = "iam"
		err.Name = name
		return "", err
	case 1:
		return all[0].ID, nil
	default:
		err := &golangsdk.ErrMultipleResourcesFound{}
		err.ResourceType = "iam"
		err.Name = name
		err.Count = count
		return "", err
	}
}

func changeToPRPair(prs *schema.Set) (r map[string]bool) {
	r = make(map[string]bool)
	for _, v := range prs.List() {
		pr := v.(map[string]interface{})

		pn := pr["project"].(string)
		rs := pr["roles"].(*schema.Set)
		for _, role := range rs.List() {
			r[pn+"|"+role.(string)] = true
		}
	}
	return
}

func diffChangeOfProjectRole(old, newv *schema.Set) (delete, add []string) {
	delete = make([]string, 0)
	add = make([]string, 0)

	oldprs := changeToPRPair(old)
	newprs := changeToPRPair(newv)

	for k := range oldprs {
		if _, ok := newprs[k]; !ok {
			delete = append(delete, k)
		}

	}

	for k := range newprs {
		if _, ok := oldprs[k]; !ok {
			add = append(add, k)
		}
	}
	return
}

func resourceIAMAgencyV3Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	client, err := agencyClient(d, config)
	if err != nil {
		return fmt.Errorf("Error creating HuaweiCloud client: %s", err)
	}

	domainID, err := getDomainID(config, client)
	if err != nil {
		return fmt.Errorf("Error getting the domain id, err=%s", err)
	}

	opts := agency.CreateOpts{
		Name:             d.Get("name").(string),
		DomainID:         domainID,
		AgencyDomainName: d.Get("agency_domain_name").(string),
		Description:      d.Get("description").(string),
	}
	log.Printf("[DEBUG] Create IAM-Agency Options: %#v", opts)
	a, err := agency.Create(client, opts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating IAM-Agency: %s", err)
	}

	d.SetId(a.ID)

	projects, err := listProjectsOfDomain(domainID, client)
	if err != nil {
		return fmt.Errorf("Error querying the projects, err=%s", err)
	}

	roles, err := allRolesOfDomain(domainID, client)
	if err != nil {
		return fmt.Errorf("Error querying the roles, err=%s", err)
	}

	agencyID := a.ID
	prs := d.Get("project_role").(*schema.Set)
	for _, v := range prs.List() {
		pr := v.(map[string]interface{})
		pn := pr["project"].(string)
		pid, ok := projects[pn]
		if !ok {
			return fmt.Errorf("The project(%s) is not exist", pn)
		}

		rs := pr["roles"].(*schema.Set)
		for _, role := range rs.List() {
			r := role.(string)
			rid, ok := roles[r]
			if !ok {
				return fmt.Errorf("The role(%s) is not exist", r)
			}

			err = agency.AttachRoleByProject(client, agencyID, pid, rid).ExtractErr()
			if err != nil {
				return fmt.Errorf("Error attaching role(%s) by project{%s} to agency, err=%s",
					rid, pid, err)
			}
		}
	}

	/*
		drs := d.Get("domain_roles").([]interface{})
		//agencyDomainID := a.AgencyDomainID
		for _, v := range drs {
			err = agency.AttachRoleByDomain(client, agencyID, domainID, v.(string)).ExtractErr()
			if err != nil {
				return fmt.Errorf("Error attaching role(%s) by domain{%s}, err=%s",
					v.(string), domainID, err)
			}
		}*/
	return resourceIAMAgencyV3Read(d, meta)
}

func resourceIAMAgencyV3Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	client, err := agencyClient(d, config)
	if err != nil {
		return fmt.Errorf("Error creating HuaweiCloud client: %s", err)
	}

	a, err := agency.Get(client, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "IAM-Agency")
	}
	log.Printf("[DEBUG] Retrieved IAM-Agency %s: %#v", d.Id(), a)

	d.Set("region", GetRegion(d, config))
	d.Set("name", a.Name)
	d.Set("agency_domain_name", a.AgencyDomainName)
	d.Set("description", a.Description)
	d.Set("duration", a.Duration)
	d.Set("expire_time", a.ExpireTime)
	d.Set("create_time", a.CreateTime)

	projects, err := listProjectsOfDomain(a.DomainID, client)
	if err != nil {
		return fmt.Errorf("Error querying the projects, err=%s", err)
	}
	agencyID := d.Id()
	var prs []map[string]interface{}
	for pn, pid := range projects {
		roles, err := agency.ListRolesAttachedOnProject(client, agencyID, pid).ExtractRoles()
		if err != nil && !isResourceNotFound(err) {
			return fmt.Errorf("Error querying the roles attached on project(%s), err=%s", pn, err)
		}
		if len(roles) == 0 {
			continue
		}
		v := make([]string, len(roles))
		for i, role := range roles {
			v[i] = role.Extra["display_name"].(string)
		}
		prs = append(prs, map[string]interface{}{
			"project": pn,
			"roles":   v,
		})
	}
	d.Set("project_role", prs)
	return nil
}

func resourceIAMAgencyV3Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	client, err := agencyClient(d, config)
	if err != nil {
		return fmt.Errorf("Error creating HuaweiCloud client: %s", err)
	}

	aID := d.Id()

	if d.HasChange("agency_domain_name") || d.HasChange("description") {
		updateOpts := agency.UpdateOpts{
			AgencyDomainName: d.Get("agency_domain_name").(string),
			Description:      d.Get("description").(string),
		}
		log.Printf("[DEBUG] Updating IAM-Agency %s with options: %#v", aID, updateOpts)
		timeout := d.Timeout(schema.TimeoutUpdate)
		err = resource.Retry(timeout, func() *resource.RetryError {
			_, err := agency.Update(client, aID, updateOpts).Extract()
			if err != nil {
				return checkForRetryableError(err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("Error updating IAM-Agency %s: %s", aID, err)
		}
	}

	if d.HasChange("project_role") {
		domainID, err := getDomainID(config, client)
		if err != nil {
			return fmt.Errorf("Error getting the domain id, err=%s", err)
		}

		projects, err := listProjectsOfDomain(domainID, client)
		if err != nil {
			return fmt.Errorf("Error querying the projects, err=%s", err)
		}

		roles, err := allRolesOfDomain(domainID, client)
		if err != nil {
			return fmt.Errorf("Error querying the roles, err=%s", err)
		}

		o, n := d.GetChange("project_role")
		deleteprs, addprs := diffChangeOfProjectRole(o.(*schema.Set), n.(*schema.Set))
		for _, v := range deleteprs {
			pr := strings.Split(v, "|")
			pid, ok := projects[pr[0]]
			if !ok {
				return fmt.Errorf("The project(%s) is not exist", pr[0])
			}
			rid, ok := roles[pr[1]]
			if !ok {
				return fmt.Errorf("The role(%s) is not exist", pr[1])
			}

			err = agency.DetachRoleByProject(client, aID, pid, rid).ExtractErr()
			if err != nil {
				return fmt.Errorf("Error detaching role(%s) by project{%s} to agency(%s), err=%s",
					rid, pid, aID, err)
			}
		}

		for _, v := range addprs {
			pr := strings.Split(v, "|")
			pid, ok := projects[pr[0]]
			if !ok {
				return fmt.Errorf("The project(%s) is not exist", pr[0])
			}
			rid, ok := roles[pr[1]]
			if !ok {
				return fmt.Errorf("The role(%s) is not exist", pr[1])
			}

			err = agency.AttachRoleByProject(client, aID, pid, rid).ExtractErr()
			if err != nil {
				return fmt.Errorf("Error attaching role(%s) by project{%s} to agency(%s), err=%s",
					rid, pid, aID, err)
			}
		}
	}

	return resourceIAMAgencyV3Read(d, meta)
}

func resourceIAMAgencyV3Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	client, err := agencyClient(d, config)
	if err != nil {
		return fmt.Errorf("Error creating HuaweiCloud client: %s", err)
	}

	rID := d.Id()
	log.Printf("[DEBUG] Deleting IAM-Agency %s", rID)

	timeout := d.Timeout(schema.TimeoutDelete)
	err = resource.Retry(timeout, func() *resource.RetryError {
		err := agency.Delete(client, rID).ExtractErr()
		if err != nil {
			return checkForRetryableError(err)
		}
		return nil
	})
	if err != nil {
		if isResourceNotFound(err) {
			log.Printf("[INFO] deleting an unavailable IAM-Agency: %s", rID)
			return nil
		}
		return fmt.Errorf("Error deleting IAM-Agency %s: %s", rID, err)
	}

	return nil
}
