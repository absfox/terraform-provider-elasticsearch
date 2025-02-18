package es

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	elastic7 "github.com/olivere/elastic/v7"
	elastic5 "gopkg.in/olivere/elastic.v5"
	elastic6 "gopkg.in/olivere/elastic.v6"
)

func resourceElasticsearchXpackUser() *schema.Resource {
	return &schema.Resource{
		Description: "Provides an Elasticsearch XPack user resource. See the upstream [docs](https://www.elastic.co/guide/en/elasticsearch/reference/current/security-api.html) for more details.",
		Create:      resourceElasticsearchXpackUserCreate,
		Read:        resourceElasticsearchXpackUserRead,
		Update:      resourceElasticsearchXpackUserUpdate,
		Delete:      resourceElasticsearchXpackUserDelete,

		Schema: map[string]*schema.Schema{
			"username": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "An identifier for the user. \n\n Usernames must be at least 1 and no more than 1024 characters. They can contain alphanumeric characters (a-z, A-Z, 0-9), spaces, punctuation, and printable symbols in the Basic Latin (ASCII) block. Leading or trailing whitespace is not allowed.",
			},
			"fullname": {
				Type:        schema.TypeString,
				Optional:    true,
				Required:    false,
				Description: "The full name of the user",
			},
			"email": {
				Type:        schema.TypeString,
				Optional:    true,
				Required:    false,
				Description: "The email of the user",
			},
			"enabled": {
				Type:        schema.TypeBool,
				Default:     true,
				Optional:    true,
				Required:    false,
				Description: "Specifies whether the user is enabled, defaults to true.",
			},
			"password": {
				Type:        schema.TypeString,
				Sensitive:   true,
				Required:    false,
				Optional:    true,
				StateFunc:   hashSum,
				Description: "The user’s password. Passwords must be at least 6 characters long. Mutually exclusive with `password_hash`, one of which must be provided at creation.",
			},
			"password_hash": {
				Type:        schema.TypeString,
				Required:    false,
				Sensitive:   true,
				Optional:    true,
				StateFunc:   hashSum,
				Description: "A hash of the user’s password. This must be produced using the same hashing algorithm as has been configured for password storage. Mutually exclusive with `password`, one of which must be provided at creation.",
			},
			"roles": {
				Type:     schema.TypeSet,
				Optional: false,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "A set of roles the user has. The roles determine the user’s access permissions",
			},
			"metadata": {
				Type:             schema.TypeString,
				Default:          "{}",
				Optional:         true,
				DiffSuppressFunc: suppressEquivalentJson,
				Description:      "Arbitrary metadata that you want to associate with the user",
			},
		},
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func resourceElasticsearchXpackUserCreate(d *schema.ResourceData, m interface{}) error {
	name := d.Get("username").(string)

	reqBody, err := buildPutUserBody(d, m)
	if err != nil {
		return err
	}
	err = xpackPutUser(d, m, name, reqBody)
	if err != nil {
		return err
	}
	d.SetId(name)
	return resourceElasticsearchXpackUserRead(d, m)
}

func resourceElasticsearchXpackUserRead(d *schema.ResourceData, m interface{}) error {

	user, err := xpackGetUser(d, m, d.Id())
	if err != nil {
		fmt.Println("Error during read")
		if elasticErr, ok := err.(*elastic7.Error); ok && elastic7.IsNotFound(elasticErr) {
			fmt.Printf("[WARN] User %s not found. Removing from state\n", d.Id())
			d.SetId("")
			return nil
		}
		if elasticErr, ok := err.(*elastic6.Error); ok && elastic6.IsNotFound(elasticErr) {
			fmt.Printf("[WARN] User %s not found. Removing from state\n", d.Id())
			d.SetId("")
			return nil
		}
		if elasticErr, ok := err.(*elastic5.Error); ok && elastic5.IsNotFound(elasticErr) {
			fmt.Printf("[WARN] User %s not found. Removing from state\n", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	ds := &resourceDataSetter{d: d}
	ds.set("username", user.Username)
	ds.set("roles", user.Roles)
	ds.set("fullname", user.Fullname)
	ds.set("email", user.Email)
	ds.set("metadata", user.Metadata)
	ds.set("enabled", user.Enabled)
	return ds.err
}

func resourceElasticsearchXpackUserUpdate(d *schema.ResourceData, m interface{}) error {
	name := d.Get("username").(string)

	reqBody, err := buildPutUserBody(d, m)
	if err != nil {
		return err
	}
	err = xpackPutUser(d, m, name, reqBody)
	if err != nil {
		return err
	}
	return resourceElasticsearchXpackUserRead(d, m)
}

func resourceElasticsearchXpackUserDelete(d *schema.ResourceData, m interface{}) error {

	err := xpackDeleteUser(d, m, d.Id())
	if err != nil {
		fmt.Println("Error during destroy")
		if elasticErr, ok := err.(*elastic7.Error); ok && elastic7.IsNotFound(elasticErr) {
			fmt.Printf("[WARN] User %s not found. Resource removed from state\n", d.Id())
			d.SetId("")
			return nil
		}
		if elasticErr, ok := err.(*elastic6.Error); ok && elastic6.IsNotFound(elasticErr) {
			fmt.Printf("[WARN] User %s not found. Resource removed from state\n", d.Id())
			d.SetId("")
			return nil
		}
		if elasticErr, ok := err.(*elastic5.Error); ok && elastic5.IsNotFound(elasticErr) {
			fmt.Printf("[WARN] User %s not found. Resource removed from state\n", d.Id())
			d.SetId("")
			return nil
		}
	}
	d.SetId("")
	return nil
}

func buildPutUserBody(d *schema.ResourceData, m interface{}) (string, error) {
	roles := expandStringList(d.Get("roles").(*schema.Set).List())
	username := d.Get("username").(string)
	fullname := d.Get("fullname").(string)
	password := d.Get("password").(string)
	passwordHash := d.Get("password_hash").(string)
	email := d.Get("email").(string)
	enabled := d.Get("enabled").(bool)
	metadata := d.Get("metadata").(string)

	user := XPackSecurityUser{
		Username: username,
		Roles:    roles,
		Fullname: fullname,
		Email:    email,
		Enabled:  enabled,
		Metadata: optionalInterfaceJson(metadata),
	}

	if d.HasChange("password") {
		user.Password = password
	}
	if d.HasChange("password_hash") {
		user.PasswordHash = passwordHash
	}

	body, err := json.Marshal(user)
	if err != nil {
		fmt.Printf("Body : %s", body)
		err = fmt.Errorf("Body Error : %s", body)
	}
	log.Printf("[INFO] put body: %+v", user)
	return string(body[:]), err
}

func xpackPutUser(d *schema.ResourceData, m interface{}, name string, body string) error {
	esClient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return err
	}
	switch client := esClient.(type) {
	case *elastic7.Client:
		return elastic7PutUser(client, name, body)
	case *elastic6.Client:
		return elastic6PutUser(client, name, body)
	case *elastic5.Client:
		return elastic5PutUser(client, name, body)
	default:
		return errors.New("unhandled client type")
	}
}

func xpackGetUser(d *schema.ResourceData, m interface{}, name string) (XPackSecurityUser, error) {
	esClient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return XPackSecurityUser{}, err
	}
	switch client := esClient.(type) {
	case *elastic7.Client:
		return elastic7GetUser(client, name)
	case *elastic6.Client:
		return elastic6GetUser(client, name)
	case *elastic5.Client:
		return elastic5GetUser(client, name)
	default:
		return XPackSecurityUser{}, errors.New("unhandled client type")
	}
}

func xpackDeleteUser(d *schema.ResourceData, m interface{}, name string) error {
	esClient, err := getClient(m.(*ProviderConf))
	if err != nil {
		return err
	}
	switch client := esClient.(type) {
	case *elastic7.Client:
		return elastic7DeleteUser(client, name)
	case *elastic6.Client:
		return elastic6DeleteUser(client, name)
	case *elastic5.Client:
		return elastic5DeleteUser(client, name)
	default:
		return errors.New("unhandled client type")
	}
}

func elastic5PutUser(client *elastic5.Client, name string, body string) error {
	return errors.New("unsupported in elasticv5 client")
}

func elastic6PutUser(client *elastic6.Client, name string, body string) error {
	_, err := client.XPackSecurityPutUser(name).Body(body).Do(context.Background())
	log.Printf("[INFO] put error: %+v", err)
	return err
}

func elastic7PutUser(client *elastic7.Client, name string, body string) error {
	_, err := client.XPackSecurityPutUser(name).Body(body).Do(context.Background())
	log.Printf("[INFO] put error: %+v", err)
	return err
}

func elastic5GetUser(client *elastic5.Client, name string) (XPackSecurityUser, error) {
	err := errors.New("unsupported in elasticv5 client")
	return XPackSecurityUser{}, err
}

func elastic6GetUser(client *elastic6.Client, name string) (XPackSecurityUser, error) {
	res, err := client.XPackSecurityGetUser(name).Do(context.Background())
	if err != nil {
		return XPackSecurityUser{}, err
	}
	obj := (*res)[name]
	user := XPackSecurityUser{}
	user.Username = name
	user.Roles = obj.Roles
	user.Fullname = obj.Fullname
	user.Email = obj.Email
	user.Enabled = obj.Enabled
	if metadata, err := json.Marshal(obj.Metadata); err != nil {
		return user, err
	} else {
		user.Metadata = string(metadata)
	}
	return user, err
}

func elastic7GetUser(client *elastic7.Client, name string) (XPackSecurityUser, error) {
	res, err := client.XPackSecurityGetUser(name).Do(context.Background())
	if err != nil {
		return XPackSecurityUser{}, err
	}
	obj := (*res)[name]
	user := XPackSecurityUser{}
	user.Username = name
	user.Roles = obj.Roles
	user.Fullname = obj.Fullname
	user.Email = obj.Email
	user.Enabled = obj.Enabled
	if metadata, err := json.Marshal(obj.Metadata); err != nil {
		return user, err
	} else {
		user.Metadata = string(metadata)
	}
	return user, err
}

func elastic5DeleteUser(client *elastic5.Client, name string) error {
	err := errors.New("unsupported in elasticv5 client")
	return err
}

func elastic6DeleteUser(client *elastic6.Client, name string) error {
	_, err := client.XPackSecurityDeleteUser(name).Do(context.Background())
	return err
}

func elastic7DeleteUser(client *elastic7.Client, name string) error {
	_, err := client.XPackSecurityDeleteUser(name).Do(context.Background())
	return err
}

// XPackSecurityUser is the user object.
//
// we want to define a new struct as the one from elastic has metadata as
// a map[string]interface{} but we want to manage string only
type XPackSecurityUser struct {
	Username     string      `json:"username"`
	Roles        []string    `json:"roles"`
	Fullname     string      `json:"full_name,omitempty"`
	Email        string      `json:"email,omitempty"`
	Metadata     interface{} `json:"metadata,omitempty"`
	Enabled      bool        `json:"enabled,omitempty"`
	Password     string      `json:"password,omitempty"`
	PasswordHash string      `json:"password_hash,omitempty"`
}
