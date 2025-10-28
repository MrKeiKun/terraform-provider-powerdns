// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// getConfigValueWithEnvFallback returns the config value or falls back to the environment variable.
func getConfigValueWithEnvFallback(configValue string, envVar string) string {
	if configValue == "" {
		return os.Getenv(envVar)
	}
	return configValue
}

// getConfigBoolWithEnvFallback returns the config bool or falls back to the environment variable.
func getConfigBoolWithEnvFallback(configValue bool, isNull bool, isUnknown bool, envVar string) bool {
	if isNull || isUnknown {
		if env := os.Getenv(envVar); env != "" {
			if parsed, err := strconv.ParseBool(env); err == nil {
				return parsed
			}
		}
	}
	return configValue
}

// getConfigIntWithEnvFallback returns the config int or falls back to the environment variable.
func getConfigIntWithEnvFallback(configValue int, isNull bool, isUnknown bool, envVar string) int {
	if isNull || isUnknown {
		if env := os.Getenv(envVar); env != "" {
			if parsed, err := strconv.Atoi(env); err == nil {
				return parsed
			}
		}
	}
	return configValue
}

// Ensure PowerDNSProvider satisfies various provider interfaces.
var _ provider.Provider = &PowerDNSProvider{}

// PowerDNSProvider defines the provider implementation.
type PowerDNSProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// PowerDNSProviderModel describes the provider data model.
type PowerDNSProviderModel struct {
	APIKey            types.String `tfsdk:"api_key"`
	ClientCertFile    types.String `tfsdk:"client_cert_file"`
	ClientCertKeyFile types.String `tfsdk:"client_cert_key_file"`
	ServerURL         types.String `tfsdk:"server_url"`
	RecursorServerURL types.String `tfsdk:"recursor_server_url"`
	InsecureHTTPS     types.Bool   `tfsdk:"insecure_https"`
	CACertificate     types.String `tfsdk:"ca_certificate"`
	CacheRequests     types.Bool   `tfsdk:"cache_requests"`
	CacheMemSize      types.String `tfsdk:"cache_mem_size"`
	CacheTTL          types.Int64  `tfsdk:"cache_ttl"`
}

func (p *PowerDNSProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "powerdns"
	resp.Version = p.version
}

func (p *PowerDNSProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				MarkdownDescription: "REST API authentication API key. Can also be set via PDNS_API_KEY.",
				Optional:            true,
				Sensitive:           true,
			},
			"client_cert_file": schema.StringAttribute{
				MarkdownDescription: "REST API authentication client certificate file path (.crt). Can also be set via PDNS_CLIENT_CERT_FILE.",
				Optional:            true,
			},
			"client_cert_key_file": schema.StringAttribute{
				MarkdownDescription: "REST API authentication client certificate key file path (.key). Can also be set via PDNS_CLIENT_CERT_KEY_FILE.",
				Optional:            true,
			},
			"server_url": schema.StringAttribute{
				MarkdownDescription: "Base URL of the PowerDNS server (e.g., https://pdns.example.com). Can also be set via PDNS_SERVER_URL.",
				Optional:            true,
			},
			"insecure_https": schema.BoolAttribute{
				MarkdownDescription: "Disable verification of the PowerDNS server's TLS certificate. Also via PDNS_INSECURE_HTTPS.",
				Optional:            true,
			},
			"ca_certificate": schema.StringAttribute{
				MarkdownDescription: "Content or path of a Root CA to verify the server certificate. Also via PDNS_CACERT.",
				Optional:            true,
			},
			"cache_requests": schema.BoolAttribute{
				MarkdownDescription: "Enable caching of REST API requests. Also via PDNS_CACHE_REQUESTS.",
				Optional:            true,
			},
			"cache_mem_size": schema.StringAttribute{
				MarkdownDescription: "Cache memory size in MB. Also via PDNS_CACHE_MEM_SIZE.",
				Optional:            true,
			},
			"cache_ttl": schema.Int64Attribute{
				MarkdownDescription: "Cache TTL in seconds. Also via PDNS_CACHE_TTL.",
				Optional:            true,
			},
			"recursor_server_url": schema.StringAttribute{
				MarkdownDescription: "Base URL of the PowerDNS recursor server. Also via PDNS_RECURSOR_SERVER_URL.",
				Optional:            true,
			},
		},
	}
}

func (p *PowerDNSProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data PowerDNSProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Create the client
	config := Config{
		APIKey:            getConfigValueWithEnvFallback(data.APIKey.ValueString(), "PDNS_API_KEY"),
		ClientCertFile:    getConfigValueWithEnvFallback(data.ClientCertFile.ValueString(), "PDNS_CLIENT_CERT_FILE"),
		ClientCertKeyFile: getConfigValueWithEnvFallback(data.ClientCertKeyFile.ValueString(), "PDNS_CLIENT_CERT_KEY_FILE"),
		ServerURL:         getConfigValueWithEnvFallback(data.ServerURL.ValueString(), "PDNS_SERVER_URL"),
		RecursorServerURL: getConfigValueWithEnvFallback(data.RecursorServerURL.ValueString(), "PDNS_RECURSOR_SERVER_URL"),
		InsecureHTTPS:     getConfigBoolWithEnvFallback(data.InsecureHTTPS.ValueBool(), data.InsecureHTTPS.IsNull(), data.InsecureHTTPS.IsUnknown(), "PDNS_INSECURE_HTTPS"),
		CACertificate:     getConfigValueWithEnvFallback(data.CACertificate.ValueString(), "PDNS_CACERT"),
		CacheEnable:       getConfigBoolWithEnvFallback(data.CacheRequests.ValueBool(), data.CacheRequests.IsNull(), data.CacheRequests.IsUnknown(), "PDNS_CACHE_REQUESTS"),
		CacheMemorySize:   getConfigValueWithEnvFallback(data.CacheMemSize.ValueString(), "PDNS_CACHE_MEM_SIZE"),
		CacheTTL:          getConfigIntWithEnvFallback(int(data.CacheTTL.ValueInt64()), data.CacheTTL.IsNull(), data.CacheTTL.IsUnknown(), "PDNS_CACHE_TTL"),
	}

	client, err := config.Client(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create PowerDNS client", err.Error())
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *PowerDNSProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewZoneResource,
		NewRecordResource,
		NewPTRRecordResource,
		NewReverseZoneResource,
		NewRecursorConfigResource,
		NewRecursorForwardZoneResource,
	}
}

func (p *PowerDNSProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewReverseZoneDataSource,
		NewZoneDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &PowerDNSProvider{
			version: version,
		}
	}
}
