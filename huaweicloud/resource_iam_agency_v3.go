package huaweicloud

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/huaweicloud/golangsdk"
	"github.com/huaweicloud/golangsdk/openstack/identity/v3/agency"
	"github.com/huaweicloud/golangsdk/openstack/identity/v3/domains"
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
		},
	}
}

func chooseIAMV3Client(d *schema.ResourceData, config *Config) (*golangsdk.ServiceClient, error) {
	c, err := config.loadIAMV3Client(GetRegion(d, config))
	if err != nil {
		return nil, err
	}
	c.Endpoint = "https://iam.myhwclouds.com:443/v3.0/"
	return c, nil
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

func resourceIAMAgencyV3Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	client, err := chooseIAMV3Client(d, config)
	if err != nil {
		return fmt.Errorf("Error creating HuaweiCloud client: %s", err)
	}

	domainID, err := getDomainID(config, client)
	if err != nil {
		return err
	}
	opts := agency.CreateOpts{
		Name:             d.Get("name").(string),
		DomainID:         domainID,
		AgencyDomainName: d.Get("agency_domain_name").(string),
		Description:      d.Get("description").(string),
	}
	log.Printf("[DEBUG] Create IAM-Agency Options: %#v", opts)

	r, err := agency.Create(client, opts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating IAM-Agency: %s", err)
	}

	d.SetId(r.ID)

	return resourceIAMAgencyV3Read(d, meta)
}

func resourceIAMAgencyV3Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	client, err := chooseIAMV3Client(d, config)
	if err != nil {
		return fmt.Errorf("Error creating HuaweiCloud client: %s", err)
	}

	r, err := agency.Get(client, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "IAM-Agency")
	}
	log.Printf("[DEBUG] Retrieved IAM-Agency %s: %#v", d.Id(), r)

	d.Set("region", GetRegion(d, config))
	d.Set("name", r.Name)
	d.Set("agency_domain_name", r.AgencyDomainName)
	d.Set("description", r.Description)
	d.Set("duration", r.Duration)
	d.Set("expire_time", r.ExpireTime)
	d.Set("create_time", r.CreateTime)

	return nil
}

func resourceIAMAgencyV3Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	client, err := chooseIAMV3Client(d, config)
	if err != nil {
		return fmt.Errorf("Error creating HuaweiCloud client: %s", err)
	}

	rID := d.Id()

	updateOpts := agency.UpdateOpts{
		AgencyDomainName: d.Get("agency_domain_name").(string),
		Description:      d.Get("description").(string),
	}
	log.Printf("[DEBUG] Updating IAM-Agency %s with options: %#v", rID, updateOpts)
	timeout := d.Timeout(schema.TimeoutUpdate)
	err = resource.Retry(timeout, func() *resource.RetryError {
		_, err := agency.Update(client, rID, updateOpts).Extract()
		if err != nil {
			return checkForRetryableError(err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("Error updating IAM-Agency %s: %s", rID, err)
	}

	return resourceIAMAgencyV3Read(d, meta)
}

func resourceIAMAgencyV3Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	client, err := chooseIAMV3Client(d, config)
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
