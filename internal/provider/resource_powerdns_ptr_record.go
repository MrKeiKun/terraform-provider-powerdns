package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var _ resource.Resource = &PTRRecordResource{}

// PTRRecordResource defines the resource implementation.
type PTRRecordResource struct {
	client *Client
}

// PTRRecordResourceModel describes the resource data model.
type PTRRecordResourceModel struct {
	IPAddress   types.String `tfsdk:"ip_address"`
	Hostname    types.String `tfsdk:"hostname"`
	TTL         types.Int64  `tfsdk:"ttl"`
	ReverseZone types.String `tfsdk:"reverse_zone"`
	ID          types.String `tfsdk:"id"`
}

func (r *PTRRecordResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ptr_record"
}

func (r *PTRRecordResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"ip_address": schema.StringAttribute{
				MarkdownDescription: "The IP address to create a PTR record for (IPv4 or IPv6)",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"hostname": schema.StringAttribute{
				MarkdownDescription: "The hostname to point to",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ttl": schema.Int64Attribute{
				MarkdownDescription: "The TTL of the PTR record",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"reverse_zone": schema.StringAttribute{
				MarkdownDescription: "The name of the reverse zone",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "PTR record identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *PTRRecordResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PTRRecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PTRRecordResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ipAddress := data.IPAddress.ValueString()
	hostname := data.Hostname.ValueString()
	ttl := int(data.TTL.ValueInt64())
	reverseZone := data.ReverseZone.ValueString()

	tflog.SetField(ctx, "ip_address", ipAddress)
	tflog.SetField(ctx, "reverse_zone", reverseZone)
	tflog.Debug(ctx, "Creating PTR record")

	// Get the PTR record name
	ptrName, err := GetPTRRecordName(ipAddress)
	if err != nil {
		resp.Diagnostics.AddError("Failed to determine PTR record name", fmt.Errorf("failed to determine PTR record name: %w", err).Error())
		return
	}

	// Determine the correct suffix based on IP version
	suffix := ".in-addr.arpa."
	if net.ParseIP(ipAddress).To4() == nil {
		suffix = ".ip6.arpa."
	}

	// Create the PTR record with full FQDN
	rrSet := ResourceRecordSet{
		Name:       ptrName + suffix,
		Type:       "PTR",
		TTL:        ttl,
		ChangeType: "REPLACE",
		Records: []Record{
			{
				Content: hostname,
				TTL:     ttl,
			},
		},
	}

	// Ensure reverse zone exists before creating PTR record
	exists, err := r.client.ZoneExists(ctx, reverseZone)
	if err != nil {
		resp.Diagnostics.AddError("Failed to verify zone existence", fmt.Errorf("error checking zone existence: %w", err).Error())
		return
	}

	if !exists {
		resp.Diagnostics.AddError("Zone not found", fmt.Sprintf("reverse zone %s does not exist", reverseZone))
		return
	}

	recID, err := r.client.ReplaceRecordSet(ctx, reverseZone, rrSet)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create PTR record", fmt.Errorf("failed to create PTR record: %w", err).Error())
		return
	}

	data.ID = types.StringValue(recID)
	tflog.Info(ctx, "Created PTR record", map[string]any{
		"id":          recID,
		"ptr_name":    rrSet.Name,
		"reverseZone": reverseZone,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PTRRecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PTRRecordResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ipAddress := data.IPAddress.ValueString()
	reverseZone := data.ReverseZone.ValueString()

	tflog.SetField(ctx, "ip_address", ipAddress)
	tflog.SetField(ctx, "reverse_zone", reverseZone)
	tflog.Debug(ctx, "Reading PTR record")

	// Get the PTR record name
	ptrName, err := GetPTRRecordName(ipAddress)
	if err != nil {
		resp.Diagnostics.AddError("Failed to determine PTR record name", fmt.Errorf("failed to determine PTR record name: %w", err).Error())
		return
	}

	// Determine the correct suffix based on IP version
	suffix := ".in-addr.arpa."
	if net.ParseIP(ipAddress).To4() == nil {
		suffix = ".ip6.arpa."
	}

	records, err := r.client.ListRecordsInRRSet(ctx, reverseZone, ptrName+suffix, "PTR")
	if err != nil {
		resp.Diagnostics.AddError("Failed to read PTR record", fmt.Errorf("couldn't fetch PTR record: %w", err).Error())
		return
	}

	if len(records) == 0 {
		tflog.Warn(ctx, "PTR record not found; removing from state", map[string]any{
			"ptr_name": ptrName + suffix,
		})
		resp.State.RemoveResource(ctx)
		return
	}

	tflog.Debug(ctx, "Found PTR record", map[string]any{
		"ptr_name": ptrName + suffix,
		"content":  records[0].Content,
	})

	data.Hostname = types.StringValue(records[0].Content)
	data.TTL = types.Int64Value(int64(records[0].TTL))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PTRRecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PTRRecordResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ipAddress := data.IPAddress.ValueString()
	reverseZone := data.ReverseZone.ValueString()

	tflog.SetField(ctx, "ip_address", ipAddress)
	tflog.SetField(ctx, "reverse_zone", reverseZone)
	tflog.Debug(ctx, "Deleting PTR record")

	// Get the PTR record name
	ptrName, err := GetPTRRecordName(ipAddress)
	if err != nil {
		resp.Diagnostics.AddError("Failed to determine PTR record name", fmt.Errorf("failed to determine PTR record name: %w", err).Error())
		return
	}

	// Determine the correct suffix based on IP version
	suffix := ".in-addr.arpa."
	if net.ParseIP(ipAddress).To4() == nil {
		suffix = ".ip6.arpa."
	}

	if err := r.client.DeleteRecordSet(ctx, reverseZone, ptrName+suffix, "PTR"); err != nil {
		// Check if this is a backend limitation error (common with LMDB)
		if strings.Contains(err.Error(), "Hosting backend does not support editing records") ||
			strings.Contains(err.Error(), "Attempt to abort a transaction while there isn't one open") {
			tflog.Warn(ctx, "Backend does not support record deletion via API, removing from state only", map[string]any{
				"error": err.Error(),
				"zone":  reverseZone,
				"ptr":   ptrName + suffix,
			})
			// Don't return error - let the resource be removed from state
			return
		}
		resp.Diagnostics.AddError("Failed to delete PTR record", fmt.Errorf("error deleting PTR record: %w", err).Error())
		return
	}

	tflog.Info(ctx, "Successfully deleted PTR record", map[string]any{
		"ptr_name": ptrName + suffix,
	})
}

func (r *PTRRecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Info(ctx, "Importing PTR record", map[string]any{"id": req.ID})

	var data map[string]string
	if err := json.Unmarshal([]byte(req.ID), &data); err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}

	zone, ok := data["zone"]
	if !ok {
		resp.Diagnostics.AddError("Missing zone", "missing zone in import data")
		return
	}

	recordID, ok := data["id"]
	if !ok {
		resp.Diagnostics.AddError("Missing record id", "missing id in import data")
		return
	}

	tflog.Debug(ctx, "Fetching PTR record for import", map[string]any{
		"zone":     zone,
		"recordID": recordID,
	})

	records, err := r.client.ListRecordsByID(ctx, zone, recordID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch PTR record", fmt.Errorf("couldn't fetch PTR record: %w", err).Error())
		return
	}

	if len(records) == 0 {
		resp.Diagnostics.AddError("PTR record not found", "PTR record not found")
		return
	}

	tflog.Debug(ctx, "Found PTR record during import", map[string]any{
		"recordID": recordID,
		"content":  records[0].Content,
	})

	// Extract IP address from PTR record name
	parts := strings.Split(recordID, ":::")
	ip, err := ParsePTRRecordName(parts[0])
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse PTR record name", err.Error())
		return
	}

	var dataModel PTRRecordResourceModel
	dataModel.ReverseZone = types.StringValue(zone)
	dataModel.Hostname = types.StringValue(records[0].Content)
	dataModel.TTL = types.Int64Value(int64(records[0].TTL))
	dataModel.IPAddress = types.StringValue(ip.String())
	dataModel.ID = types.StringValue(recordID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &dataModel)...)
}

func (r *PTRRecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// PTR records are immutable - they use RequiresReplace() plan modifiers
	// So Update should not be called, but we need to implement it for the interface
	var data PTRRecordResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Since PTR records are immutable, just read the current state
	ipAddress := data.IPAddress.ValueString()
	reverseZone := data.ReverseZone.ValueString()

	// Get the PTR record name
	ptrName, err := GetPTRRecordName(ipAddress)
	if err != nil {
		resp.Diagnostics.AddError("Failed to determine PTR record name", fmt.Errorf("failed to determine PTR record name: %w", err).Error())
		return
	}

	// Determine the correct suffix based on IP version
	suffix := ".in-addr.arpa."
	if net.ParseIP(ipAddress).To4() == nil {
		suffix = ".ip6.arpa."
	}

	records, err := r.client.ListRecordsInRRSet(ctx, reverseZone, ptrName+suffix, "PTR")
	if err != nil {
		resp.Diagnostics.AddError("Failed to read PTR record", fmt.Errorf("couldn't fetch PTR record: %w", err).Error())
		return
	}

	if len(records) == 0 {
		resp.Diagnostics.AddError("PTR record not found", "PTR record not found during update")
		return
	}

	data.Hostname = types.StringValue(records[0].Content)
	data.TTL = types.Int64Value(int64(records[0].TTL))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func NewPTRRecordResource() resource.Resource {
	return &PTRRecordResource{}
}
