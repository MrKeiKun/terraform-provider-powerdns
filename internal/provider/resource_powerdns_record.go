package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var _ resource.Resource = &RecordResource{}

// RecordResource defines the resource implementation.
type RecordResource struct {
	client *Client
}

// RecordResourceModel describes the resource data model.
type RecordResourceModel struct {
	Zone    types.String `tfsdk:"zone"`
	Name    types.String `tfsdk:"name"`
	Type    types.String `tfsdk:"type"`
	TTL     types.Int64  `tfsdk:"ttl"`
	Records types.Set    `tfsdk:"records"`
	SetPtr  types.Bool   `tfsdk:"set_ptr"`
	ID      types.String `tfsdk:"id"`
}

func (r *RecordResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_record"
}

func (r *RecordResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"zone": schema.StringAttribute{
				MarkdownDescription: "The zone name",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The record name",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The record type",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ttl": schema.Int64Attribute{
				MarkdownDescription: "The record TTL",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"records": schema.SetAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of record values",
				Required:            true,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
				},
			},
			"set_ptr": schema.BoolAttribute{
				MarkdownDescription: "For A and AAAA records, if true, create corresponding PTR",
				Optional:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Record identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *RecordResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RecordResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate records
	if data.Records.IsNull() || len(data.Records.Elements()) == 0 {
		resp.Diagnostics.AddError("Invalid configuration", "'records' must not be empty")
		return
	}

	// Basic validation for records content
	for _, raw := range data.Records.Elements() {
		if str, ok := raw.(types.String); ok && strings.TrimSpace(str.ValueString()) == "" {
			tflog.Warn(ctx, "One or more values in 'records' are empty strings")
			break
		}
	}

	rrSet := ResourceRecordSet{
		Name: data.Name.ValueString(),
		Type: data.Type.ValueString(),
		TTL:  int(data.TTL.ValueInt64()),
	}

	records := make([]Record, 0, len(data.Records.Elements()))
	for _, rc := range data.Records.Elements() {
		if str, ok := rc.(types.String); ok {
			records = append(records, Record{
				Name:    rrSet.Name,
				Type:    rrSet.Type,
				TTL:     rrSet.TTL,
				Content: str.ValueString(),
				SetPtr:  data.SetPtr.ValueBool(),
			})
		}
	}
	rrSet.Records = records

	tflog.SetField(ctx, "zone", data.Zone.ValueString())
	tflog.SetField(ctx, "name", data.Name.ValueString())
	tflog.SetField(ctx, "type", data.Type.ValueString())
	tflog.Debug(ctx, "Creating PowerDNS record set")

	recID, err := r.client.ReplaceRecordSet(ctx, data.Zone.ValueString(), rrSet)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create record", fmt.Errorf("failed to create PowerDNS Record: %w", err).Error())
		return
	}

	data.ID = types.StringValue(recID)
	tflog.Info(ctx, "Created PowerDNS Record", map[string]any{"id": recID})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RecordResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.SetField(ctx, "zone", data.Zone.ValueString())
	tflog.SetField(ctx, "record_id", data.ID.ValueString())
	tflog.Debug(ctx, "Reading PowerDNS Record")

	records, err := r.client.ListRecordsByID(ctx, data.Zone.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read record", fmt.Errorf("couldn't fetch PowerDNS Record: %w", err).Error())
		return
	}

	if len(records) == 0 {
		// rrset no longer exists; clear state
		tflog.Warn(ctx, "PowerDNS Record not found; removing from state")
		resp.State.RemoveResource(ctx)
		return
	}

	var recs []types.String
	for _, record := range records {
		recs = append(recs, types.StringValue(record.Content))
	}

	data.Records, _ = types.SetValueFrom(ctx, types.StringType, recs)
	data.TTL = types.Int64Value(int64(records[0].TTL))
	data.Name = types.StringValue(records[0].Name)
	data.Type = types.StringValue(records[0].Type)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Records are immutable in PowerDNS - they use RequiresReplace() plan modifiers
	// So Update should not be called, but we need to implement it for the interface
	var data RecordResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Since records are immutable, just read the current state
	records, err := r.client.ListRecordsByID(ctx, data.Zone.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read record", fmt.Errorf("couldn't fetch PowerDNS Record: %w", err).Error())
		return
	}

	if len(records) == 0 {
		resp.Diagnostics.AddError("Record not found", "PowerDNS Record not found during update")
		return
	}

	var recs []types.String
	for _, record := range records {
		recs = append(recs, types.StringValue(record.Content))
	}

	data.Records, _ = types.SetValueFrom(ctx, types.StringType, recs)
	data.TTL = types.Int64Value(int64(records[0].TTL))
	data.Name = types.StringValue(records[0].Name)
	data.Type = types.StringValue(records[0].Type)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RecordResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.SetField(ctx, "zone", data.Zone.ValueString())
	tflog.SetField(ctx, "record_id", data.ID.ValueString())
	tflog.Debug(ctx, "Deleting PowerDNS Record")

	if err := r.client.DeleteRecordSetByID(ctx, data.Zone.ValueString(), data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete record", fmt.Errorf("error deleting PowerDNS Record: %w", err).Error())
		return
	}

	tflog.Info(ctx, "Deleted PowerDNS Record")
}

func (r *RecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Info(ctx, "Importing PowerDNS Record", map[string]any{"id": req.ID})

	var data map[string]string
	if err := json.Unmarshal([]byte(req.ID), &data); err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}

	zoneName, ok := data["zone"]
	if !ok {
		resp.Diagnostics.AddError("Missing zone name", "missing zone name in input data")
		return
	}
	recordID, ok := data["id"]
	if !ok {
		resp.Diagnostics.AddError("Missing record id", "missing record id in input data")
		return
	}

	tflog.Debug(ctx, "Fetching record for import", map[string]any{
		"zone": zoneName, "recordID": recordID,
	})

	records, err := r.client.ListRecordsByID(ctx, zoneName, recordID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch record", fmt.Errorf("couldn't fetch PowerDNS Record: %w", err).Error())
		return
	}
	if len(records) == 0 {
		resp.Diagnostics.AddError("No records found", "rrset has no records to import")
		return
	}

	var recs []types.String
	for _, record := range records {
		recs = append(recs, types.StringValue(record.Content))
	}

	var dataModel RecordResourceModel
	dataModel.Zone = types.StringValue(zoneName)
	dataModel.Name = types.StringValue(records[0].Name)
	dataModel.TTL = types.Int64Value(int64(records[0].TTL))
	dataModel.Type = types.StringValue(records[0].Type)
	dataModel.ID = types.StringValue(recordID)

	dataModel.Records, _ = types.SetValueFrom(ctx, types.StringType, recs)

	resp.Diagnostics.Append(resp.State.Set(ctx, &dataModel)...)
}

func NewRecordResource() resource.Resource {
	return &RecordResource{}
}
