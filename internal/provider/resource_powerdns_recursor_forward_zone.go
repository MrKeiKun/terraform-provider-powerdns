package provider

import (
	"context"
	"errors"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var _ resource.Resource = &RecursorForwardZoneResource{}

// RecursorForwardZoneResource defines the resource implementation.
type RecursorForwardZoneResource struct {
	client *Client
}

// RecursorForwardZoneResourceModel describes the resource data model.
type RecursorForwardZoneResourceModel struct {
	Zone             types.String `tfsdk:"zone"`
	Servers          types.List   `tfsdk:"servers"`
	RecursionDesired types.Bool   `tfsdk:"recursion_desired"`
	NotifyAllowed    types.Bool   `tfsdk:"notify_allowed"`
	ID               types.String `tfsdk:"id"`
}

func (r *RecursorForwardZoneResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_recursor_forward_zone"
}

func (r *RecursorForwardZoneResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages PowerDNS recursor forward zones. Forward zones allow queries for specific domains to be sent to designated DNS servers.",
		Attributes: map[string]schema.Attribute{
			"zone": schema.StringAttribute{
				MarkdownDescription: "The zone name to forward. Must be a valid DNS zone name ending with a dot.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"servers": schema.ListAttribute{
				MarkdownDescription: "List of DNS servers to forward queries to. Each server must be a valid IP address or hostname.",
				Required:            true,
				ElementType:         types.StringType,
			},
			"recursion_desired": schema.BoolAttribute{
				MarkdownDescription: "Whether the RD (Recursion Desired) bit is set. When true, the recursor will set the RD bit on outgoing queries. Default is true.",
				Optional:            true,
				Computed:            true,
			},
			"notify_allowed": schema.BoolAttribute{
				MarkdownDescription: "Whether or not to permit incoming NOTIFY to wipe cache for the domain. For zones of type \"Forwarded\".",
				Optional:            true,
				Computed:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Zone identifier. This is automatically generated and corresponds to the zone name.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *RecursorForwardZoneResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", "Expected *Client")
		return
	}
	r.client = client
}

func (r *RecursorForwardZoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RecursorForwardZoneResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneName := data.Zone.ValueString()

	// Ensure zone name ends with a dot for DNS standards
	if zoneName != "" && zoneName[len(zoneName)-1] != '.' {
		zoneName = zoneName + "."
	}

	var servers []string
	if !data.Servers.IsNull() {
		for _, s := range data.Servers.Elements() {
			if str, ok := s.(types.String); ok {
				servers = append(servers, str.ValueString())
			}
		}
	}

	// Use false as default to match PowerDNS API behavior
	recursionDesired := false
	if !data.RecursionDesired.IsNull() {
		recursionDesired = data.RecursionDesired.ValueBool()
	}

	// Use false as default to match PowerDNS API behavior
	notifyAllowed := false
	if !data.NotifyAllowed.IsNull() {
		notifyAllowed = data.NotifyAllowed.ValueBool()
	}

	tflog.SetField(ctx, "zone", zoneName)
	tflog.Debug(ctx, "Creating recursor forward zone")

	// Create the zone
	recursorZone := RecursorZone{
		Name:             zoneName,
		Kind:             "Forwarded",
		Servers:          servers,
		RecursionDesired: recursionDesired,
		NotifyAllowed:    notifyAllowed,
	}

	createdZone, err := r.client.CreateRecursorZone(ctx, recursorZone)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create recursor forward zone", err.Error())
		return
	}

	data.ID = types.StringValue(createdZone.Name)

	// Update computed fields - use original servers to avoid API normalization
	var serversList []types.String
	for _, s := range servers {
		serversList = append(serversList, types.StringValue(s))
	}
	data.Servers, _ = types.ListValueFrom(ctx, types.StringType, serversList)
	data.RecursionDesired = types.BoolValue(createdZone.RecursionDesired)
	data.NotifyAllowed = types.BoolValue(createdZone.NotifyAllowed)

	tflog.Info(ctx, "Created recursor forward zone", map[string]any{"id": createdZone.Name})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RecursorForwardZoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RecursorForwardZoneResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneName := data.ID.ValueString()

	tflog.SetField(ctx, "zone", zoneName)
	tflog.Debug(ctx, "Reading recursor forward zone")

	zone, err := r.client.GetRecursorZone(ctx, zoneName)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			tflog.Warn(ctx, "Recursor forward zone not found; removing from state")
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to get recursor forward zone", err.Error())
		return
	}

	// Check if it's a forwarded zone
	if zone.Kind != "Forwarded" {
		tflog.Warn(ctx, "Zone is not a forward zone; removing from state")
		resp.State.RemoveResource(ctx)
		return
	}

	data.Zone = types.StringValue(zone.Name)
	// For Read, we need to normalize the servers from API to match what user expects
	var serversList []types.String
	for _, s := range zone.Servers {
		// Remove default port :53 if present to match user input
		s = strings.TrimSuffix(s, ":53")
		serversList = append(serversList, types.StringValue(s))
	}
	data.Servers, _ = types.ListValueFrom(ctx, types.StringType, serversList)
	data.RecursionDesired = types.BoolValue(zone.RecursionDesired)
	data.NotifyAllowed = types.BoolValue(zone.NotifyAllowed)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RecursorForwardZoneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RecursorForwardZoneResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneName := data.Zone.ValueString()

	// Ensure zone name ends with a dot
	if zoneName != "" && zoneName[len(zoneName)-1] != '.' {
		zoneName = zoneName + "."
	}

	var servers []string
	if !data.Servers.IsNull() {
		for _, s := range data.Servers.Elements() {
			if str, ok := s.(types.String); ok {
				servers = append(servers, str.ValueString())
			}
		}
	}

	// Use false as default to match PowerDNS API behavior
	recursionDesired := false
	if !data.RecursionDesired.IsNull() {
		recursionDesired = data.RecursionDesired.ValueBool()
	}

	notifyAllowed := false
	if !data.NotifyAllowed.IsNull() {
		notifyAllowed = data.NotifyAllowed.ValueBool()
	}

	tflog.SetField(ctx, "zone", zoneName)
	tflog.Debug(ctx, "Updating recursor forward zone")

	// For updates, we need to delete and recreate the zone
	err := r.client.DeleteRecursorZone(ctx, zoneName)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update recursor forward zone", err.Error())
		return
	}

	// Recreate the zone with updated settings
	updateData := RecursorZone{
		Name:             zoneName,
		Kind:             "Forwarded",
		Servers:          servers,
		RecursionDesired: recursionDesired,
		NotifyAllowed:    notifyAllowed,
	}

	_, err = r.client.CreateRecursorZone(ctx, updateData)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update recursor forward zone", err.Error())
		return
	}

	// Update the state with the normalized zone name
	data.Zone = types.StringValue(zoneName)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RecursorForwardZoneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RecursorForwardZoneResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneName := data.Zone.ValueString()

	tflog.SetField(ctx, "zone", zoneName)
	tflog.Debug(ctx, "Deleting recursor forward zone")

	err := r.client.DeleteRecursorZone(ctx, zoneName)
	if err != nil {
		// If the zone doesn't exist, that's actually what we want
		// Return success to allow Terraform to clean up the state
		if strings.Contains(err.Error(), "Could not find domain") {
			tflog.Info(ctx, "Recursor forward zone already deleted", map[string]any{"zone": zoneName})
			return
		}
		// For other errors, report them
		resp.Diagnostics.AddError("Error deleting recursor forward zone", err.Error())
		return
	}

	tflog.Info(ctx, "Successfully deleted recursor forward zone")
}

func (r *RecursorForwardZoneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func NewRecursorForwardZoneResource() resource.Resource {
	return &RecursorForwardZoneResource{}
}
