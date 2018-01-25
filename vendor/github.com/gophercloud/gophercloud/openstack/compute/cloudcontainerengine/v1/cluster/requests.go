package cluster

import (
	"log"

	"github.com/gophercloud/gophercloud"
)

type CreateOpts struct {
	Kind       string         `json:"kind" required:"true"`
	Apiversion string         `json:"apiVersion" required:"true"`
	Metadata   MetaDataOpts   `json:"metadata" required:"true"`
	Spec       SpecCreateOpts `json:"spec" required:"true"`
}

type MetaDataOpts struct {
	Name string `json:"name" required:"true"`
}

type SpecCreateOpts struct {
	Description     string `json:"description,omitempty"`
	Vpc             string `json:"vpc" required:"true"`
	Subnet          string `json:"subnet" required:"true"`
	Region          string `json:"region" required:"true"`
	Az              string `json:"az,omitempty"`
	SecurityGroupID string `json:"security_group_id,omitempty"`
}

type CreateOptsBuilder interface {
	ToClusterCreateMap() (map[string]interface{}, error)
}

func (opts CreateOpts) ToClusterCreateMap() (map[string]interface{}, error) {
	return gophercloud.BuildRequestBody(opts, "")
}

func Create(c *gophercloud.ServiceClient, opts CreateOptsBuilder) (r CreateResult) {
	b, err := opts.ToClusterCreateMap()
	if err != nil {
		r.Err = err
		return
	}
	log.Printf("[DEBUG] create url:%q, body=%#v", createURL(c), b)
	reqOpt := &gophercloud.RequestOpts{OkCodes: []int{201}}
	_, r.Err = c.Post(rootURL(c), b, &r.Body, reqOpt)
	return
}

type SpecUpdateOpts struct {
	Description string `json:"description" required:"true"`
}

type UpdateOpts struct {
	Kind       string         `json:"kind" required:"true"`
	Apiversion string         `json:"apiVersion" required:"true"`
	Spec       SpecUpdateOpts `json:"spec" required:"true"`
}

type UpdateOptsBuilder interface {
	ToClusterUpdateMap() (map[string]interface{}, error)
}

func (opts UpdateOpts) ToClusterUpdateMap() (map[string]interface{}, error) {
	return gophercloud.BuildRequestBody(opts, "")
}

func Update(c *gophercloud.ServiceClient, id string, opts UpdateOptsBuilder) (r UpdateResult) {
	b, err := opts.ToClusterUpdateMap()
	if err != nil {
		r.Err = err
		return
	}
	log.Printf("[DEBUG] update url:%q, body=%#v", updateURL(c, id), b)
	reqOpt := &gophercloud.RequestOpts{OkCodes: []int{200}}
	_, r.Err = c.PUT(updateURL(c, id), b, &r.Body, reqOpt)
	return
}
