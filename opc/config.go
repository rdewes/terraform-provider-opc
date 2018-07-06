package opc

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/go-oracle-terraform/lbaas"
	"github.com/hashicorp/go-oracle-terraform/opc"
	"github.com/hashicorp/go-oracle-terraform/storage"
	"github.com/hashicorp/terraform/helper/logging"
	"github.com/hashicorp/terraform/terraform"
)

// Config represents the provider configuarion attributes
type Config struct {
	User             string
	Password         string
	IdentityDomain   string
	Endpoint         string
	MaxRetries       int
	Insecure         bool
	StorageEndpoint  string
	StorageServiceID string
	LBaaSEndpoint    string
}

// Client holder for the OPC (OCI Classic) API Clients
type Client struct {
	computeClient *compute.Client
	storageClient *storage.Client
	lbaasClient   *lbaas.Client
}

// Client gets the OPC (OCI Classic) API Clients
func (c *Config) Client() (*Client, error) {

	userAgentString := fmt.Sprintf("HashiCorp-Terraform-v%s", terraform.VersionString())

	config := opc.Config{
		IdentityDomain: &c.IdentityDomain,
		Username:       &c.User,
		Password:       &c.Password,
		MaxRetries:     &c.MaxRetries,
		UserAgent:      &userAgentString,
	}

	if logging.IsDebugOrHigher() {
		config.LogLevel = opc.LogDebug
		config.Logger = opcLogger{}
	}

	// Setup HTTP Client based on insecure
	httpClient := cleanhttp.DefaultClient()
	if c.Insecure {
		transport := cleanhttp.DefaultTransport()
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
		httpClient.Transport = transport
	}

	config.HTTPClient = httpClient

	client := &Client{}

	if c.Endpoint != "" {
		computeEndpoint, err := url.ParseRequestURI(c.Endpoint)
		if err != nil {
			return nil, fmt.Errorf("Invalid Compute Endpoint URI: %s", err)
		}
		config.APIEndpoint = computeEndpoint
		computeClient, err := compute.NewComputeClient(&config)
		if err != nil {
			return nil, err
		}
		client.computeClient = computeClient
		log.Print("[DEBUG] Authenticated with Compute Client")

	}

	if c.StorageEndpoint != "" {
		storageEndpoint, err := url.ParseRequestURI(c.StorageEndpoint)
		if err != nil {
			return nil, fmt.Errorf("Invalid Storage Endpoint URI: %+v", err)
		}
		config.APIEndpoint = storageEndpoint
		if (c.StorageServiceID) != "" {
			config.IdentityDomain = &c.StorageServiceID
		}
		storageClient, err := storage.NewStorageClient(&config)
		if err != nil {
			return nil, err
		}
		client.storageClient = storageClient
		log.Print("[DEBUG] Authenticated with Storage Client")

	}

	if c.LBaaSEndpoint != "" {
		lbaasEndpoint, err := url.ParseRequestURI(c.LBaaSEndpoint)
		if err != nil {
			return nil, fmt.Errorf("Invalid LBaaS Endpoint URI: %+v", err)
		}
		config.APIEndpoint = lbaasEndpoint
		lbaasClient, err := lbaas.NewClient(&config)
		if err != nil {
			return nil, err
		}
		client.lbaasClient = lbaasClient
		log.Print("[DEBUG] Authenticated with Load Balancer Client")
	}

	return client, nil
}

type opcLogger struct{}

func (l opcLogger) Log(args ...interface{}) {
	tokens := make([]string, 0, len(args))
	for _, arg := range args {
		if token, ok := arg.(string); ok {
			tokens = append(tokens, token)
		}
	}
	log.SetFlags(0)
	log.Print(fmt.Sprintf("go-oracle-terraform: %s", strings.Join(tokens, " ")))
}

func (c *Client) getLBaaSClient() (*lbaas.Client, error) {
	if c.lbaasClient == nil {
		return nil, fmt.Errorf("Load Balancer API client has not been initialized. Ensure the `lbaas_endpoint` for the Load Balancer Classic REST API Endpoint has been declared in the provider configuration.")
	}
	return c.lbaasClient, nil
}
