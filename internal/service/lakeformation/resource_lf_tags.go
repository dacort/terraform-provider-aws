package lakeformation

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/lakeformation"
	"github.com/hashicorp/aws-sdk-go-base/v2/awsv1shim/v2/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/create"
	"github.com/hashicorp/terraform-provider-aws/internal/errs"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
	"github.com/hashicorp/terraform-provider-aws/names"
)

const (
	ResNameLFTags = "Resource LF Tags"
)

// @SDKResource("aws_lakeformation_resource_lf_tags")
func ResourceResourceLFTags() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceResourceLFTagsCreate,
		ReadWithoutTimeout:   resourceResourceLFTagsRead,
		DeleteWithoutTimeout: resourceResourceLFTagsDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(20 * time.Minute),
			Delete: schema.DefaultTimeout(20 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"catalog_id": {
				Type:         schema.TypeString,
				Computed:     true,
				ForceNew:     true,
				Optional:     true,
				ValidateFunc: verify.ValidAccountID,
			},
			"database": {
				Type:     schema.TypeList,
				Computed: true,
				ForceNew: true,
				MaxItems: 1,
				Optional: true,
				ExactlyOneOf: []string{
					"database",
					"table",
					"table_with_columns",
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"catalog_id": {
							Type:         schema.TypeString,
							Computed:     true,
							ForceNew:     true,
							Optional:     true,
							ValidateFunc: verify.ValidAccountID,
						},
						"name": {
							Type:     schema.TypeString,
							ForceNew: true,
							Required: true,
						},
					},
				},
			},
			"lf_tag": {
				Type:     schema.TypeSet,
				Required: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"catalog_id": {
							Type:     schema.TypeString,
							ForceNew: true,
							Optional: true,
							Computed: true,
						},
						"key": {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							ValidateFunc: validation.StringLenBetween(1, 128),
						},
						"value": {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							ValidateFunc: validateLFTagValues(),
						},
					},
				},
				Set: lfTagsHash,
			},
			"table": {
				Type:     schema.TypeList,
				Computed: true,
				ForceNew: true,
				MaxItems: 1,
				Optional: true,
				ExactlyOneOf: []string{
					"database",
					"table",
					"table_with_columns",
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"catalog_id": {
							Type:         schema.TypeString,
							Computed:     true,
							ForceNew:     true,
							Optional:     true,
							ValidateFunc: verify.ValidAccountID,
						},
						"database_name": {
							Type:     schema.TypeString,
							ForceNew: true,
							Required: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
							ForceNew: true,
							Optional: true,
							AtLeastOneOf: []string{
								"table.0.name",
								"table.0.wildcard",
							},
						},
						"wildcard": {
							Type:     schema.TypeBool,
							Default:  false,
							ForceNew: true,
							Optional: true,
							AtLeastOneOf: []string{
								"table.0.name",
								"table.0.wildcard",
							},
						},
					},
				},
			},
			"table_with_columns": {
				Type:     schema.TypeList,
				Computed: true,
				ForceNew: true,
				MaxItems: 1,
				Optional: true,
				ExactlyOneOf: []string{
					"database",
					"table",
					"table_with_columns",
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"catalog_id": {
							Type:         schema.TypeString,
							Computed:     true,
							ForceNew:     true,
							Optional:     true,
							ValidateFunc: verify.ValidAccountID,
						},
						"column_names": {
							Type:     schema.TypeSet,
							ForceNew: true,
							Optional: true,
							Set:      schema.HashString,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validation.NoZeroValues,
							},
							AtLeastOneOf: []string{
								"table_with_columns.0.column_names",
								"table_with_columns.0.wildcard",
							},
						},
						"database_name": {
							Type:     schema.TypeString,
							ForceNew: true,
							Required: true,
						},
						"excluded_column_names": {
							Type:     schema.TypeSet,
							ForceNew: true,
							Optional: true,
							Set:      schema.HashString,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validation.NoZeroValues,
							},
						},
						"name": {
							Type:     schema.TypeString,
							ForceNew: true,
							Required: true,
						},
						"wildcard": {
							Type:     schema.TypeBool,
							Default:  false,
							ForceNew: true,
							Optional: true,
							AtLeastOneOf: []string{
								"table_with_columns.0.column_names",
								"table_with_columns.0.wildcard",
							},
						},
					},
				},
			},
		},
	}
}

func resourceResourceLFTagsCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	conn := meta.(*conns.AWSClient).LakeFormationConn(ctx)

	input := &lakeformation.AddLFTagsToResourceInput{}

	if v, ok := d.GetOk("catalog_id"); ok {
		input.CatalogId = aws.String(v.(string))
	}

	if v, ok := d.GetOk("lf_tag"); ok && v.(*schema.Set).Len() > 0 {
		input.LFTags = expandLFTagPairs(v.(*schema.Set).List())
	}

	tagger, ds := lfTagsTagger(d)
	diags = append(diags, ds...)
	if diags.HasError() {
		return diags
	}

	input.Resource = tagger.ExpandResource(d)

	var output *lakeformation.AddLFTagsToResourceOutput
	err := retry.RetryContext(ctx, IAMPropagationTimeout, func() *retry.RetryError {
		var err error
		output, err = conn.AddLFTagsToResourceWithContext(ctx, input)
		if err != nil {
			if tfawserr.ErrCodeEquals(err, lakeformation.ErrCodeConcurrentModificationException) {
				return retry.RetryableError(err)
			}
			if tfawserr.ErrMessageContains(err, "AccessDeniedException", "is not authorized") {
				return retry.RetryableError(err)
			}

			return retry.NonRetryableError(err)
		}
		return nil
	})

	if tfresource.TimedOut(err) {
		output, err = conn.AddLFTagsToResourceWithContext(ctx, input)
	}

	if err != nil {
		return create.AddError(diags, names.LakeFormation, create.ErrActionCreating, ResNameLFTags, input.String(), err)
	}

	if output != nil && len(output.Failures) > 0 {
		for _, v := range output.Failures {
			if v.LFTag == nil || v.Error == nil {
				continue
			}

			diags = create.AddError(diags,
				names.LakeFormation,
				create.ErrActionCreating,
				ResNameLFTags,
				fmt.Sprintf("catalog id:%s, tag key:%s, values:%+v", aws.StringValue(v.LFTag.CatalogId), aws.StringValue(v.LFTag.TagKey), aws.StringValueSlice(v.LFTag.TagValues)),
				awserr.New(aws.StringValue(v.Error.ErrorCode), aws.StringValue(v.Error.ErrorMessage), nil),
			)
		}
	}
	if diags.HasError() {
		return diags
	}

	d.SetId(fmt.Sprintf("%d", create.StringHashcode(input.String())))

	return append(diags, resourceResourceLFTagsRead(ctx, d, meta)...)
}

func resourceResourceLFTagsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	conn := meta.(*conns.AWSClient).LakeFormationConn(ctx)

	input := &lakeformation.GetResourceLFTagsInput{
		ShowAssignedLFTags: aws.Bool(true),
	}

	if v, ok := d.GetOk("catalog_id"); ok {
		input.CatalogId = aws.String(v.(string))
	}

	tagger, ds := lfTagsTagger(d)
	diags = append(diags, ds...)
	if diags.HasError() {
		return diags
	}

	input.Resource = tagger.ExpandResource(d)

	output, err := conn.GetResourceLFTagsWithContext(ctx, input)

	if err != nil {
		return create.AddError(diags, names.LakeFormation, create.ErrActionReading, ResNameLFTags, d.Id(), err)
	}

	if err := d.Set("lf_tag", tagger.FlattenTags(output)); err != nil {
		return create.AddError(diags, names.LakeFormation, create.ErrActionSetting, ResNameLFTags, d.Id(), err)
	}

	return diags
}

func resourceResourceLFTagsDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	conn := meta.(*conns.AWSClient).LakeFormationConn(ctx)

	input := &lakeformation.RemoveLFTagsFromResourceInput{}

	if v, ok := d.GetOk("catalog_id"); ok {
		input.CatalogId = aws.String(v.(string))
	}

	if v, ok := d.GetOk("lf_tag"); ok && v.(*schema.Set).Len() > 0 {
		input.LFTags = expandLFTagPairs(v.(*schema.Set).List())
	}

	tagger, ds := lfTagsTagger(d)
	diags = append(diags, ds...)
	if diags.HasError() {
		return diags
	}

	input.Resource = tagger.ExpandResource(d)

	if input.Resource == nil || reflect.DeepEqual(input.Resource, &lakeformation.Resource{}) || len(input.LFTags) == 0 {
		// if resource is empty, don't delete = it won't delete anything since this is the predicate
		return create.AddWarningMessage(diags, names.LakeFormation, create.ErrActionSetting, ResNameLFTags, d.Id(), "no LF-Tags to remove")
	}

	err := retry.RetryContext(ctx, d.Timeout(schema.TimeoutDelete), func() *retry.RetryError {
		var err error
		_, err = conn.RemoveLFTagsFromResourceWithContext(ctx, input)
		if err != nil {
			if tfawserr.ErrCodeEquals(err, lakeformation.ErrCodeConcurrentModificationException) {
				return retry.RetryableError(err)
			}
			if tfawserr.ErrMessageContains(err, "AccessDeniedException", "is not authorized") {
				return retry.RetryableError(err)
			}

			return retry.NonRetryableError(fmt.Errorf("removing Lake Formation LF-Tags: %w", err))
		}
		return nil
	})

	if tfresource.TimedOut(err) {
		_, err = conn.RemoveLFTagsFromResourceWithContext(ctx, input)
	}

	if err != nil {
		return create.AddError(diags, names.LakeFormation, create.ErrActionDeleting, ResNameLFTags, d.Id(), err)
	}

	return diags
}

func lfTagsTagger(d *schema.ResourceData) (tagger, diag.Diagnostics) {
	var diags diag.Diagnostics
	if v, ok := d.GetOk("database"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		return &databaseTagger{}, diags
	} else if v, ok := d.GetOk("table"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		return &tableTagger{}, diags
	} else if v, ok := d.GetOk("table_with_columns"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		return &columnTagger{}, diags
	} else {
		diags = append(diags, errs.NewErrorDiagnostic(
			"Invalid Lake Formation Resource Type",
			"An unexpected error occurred while resolving the Lake Formation Resource type. "+
				"This is always an error in the provider. "+
				"Please report the following to the provider developer:\n\n"+
				"No Lake Formation Resource defined.",
		))
		return nil, diags
	}
}

type tagger interface {
	ExpandResource(*schema.ResourceData) *lakeformation.Resource
	FlattenTags(*lakeformation.GetResourceLFTagsOutput) []any
}

type databaseTagger struct{}

func (t *databaseTagger) ExpandResource(d *schema.ResourceData) *lakeformation.Resource {
	v := d.Get("database").([]any)[0].(map[string]any)
	return &lakeformation.Resource{
		Database: ExpandDatabaseResource(v),
	}
}

func (t *databaseTagger) FlattenTags(output *lakeformation.GetResourceLFTagsOutput) []any {
	return flattenLFTagPairs(output.LFTagOnDatabase)
}

type tableTagger struct{}

func (t *tableTagger) ExpandResource(d *schema.ResourceData) *lakeformation.Resource {
	v := d.Get("table").([]any)[0].(map[string]any)
	return &lakeformation.Resource{
		Table: ExpandTableResource(v),
	}
}

func (t *tableTagger) FlattenTags(output *lakeformation.GetResourceLFTagsOutput) []any {
	return flattenLFTagPairs(output.LFTagsOnTable)
}

type columnTagger struct{}

func (t *columnTagger) ExpandResource(d *schema.ResourceData) *lakeformation.Resource {
	v := d.Get("table_with_columns").([]any)[0].(map[string]any)
	return &lakeformation.Resource{
		TableWithColumns: expandTableColumnsResource(v),
	}
}

func (t *columnTagger) FlattenTags(output *lakeformation.GetResourceLFTagsOutput) []any {
	if len(output.LFTagsOnColumns) == 0 {
		return []any{}
	}

	tags := output.LFTagsOnColumns[0]
	if tags == nil {
		return []any{}
	}

	return flattenLFTagPairs(tags.LFTags)
}

func lfTagsHash(v interface{}) int {
	m, ok := v.(map[string]interface{})

	if !ok {
		return 0
	}

	var buf bytes.Buffer
	buf.WriteString(m["key"].(string))
	buf.WriteString(m["value"].(string))
	buf.WriteString(m["catalog_id"].(string))

	return create.StringHashcode(buf.String())
}

func expandLFTagPair(tfMap map[string]interface{}) *lakeformation.LFTagPair {
	if tfMap == nil {
		return nil
	}

	apiObject := &lakeformation.LFTagPair{}

	if v, ok := tfMap["catalog_id"].(string); ok && v != "" {
		apiObject.CatalogId = aws.String(v)
	}

	if v, ok := tfMap["key"].(string); ok && v != "" {
		apiObject.TagKey = aws.String(v)
	}

	if v, ok := tfMap["value"].(string); ok && v != "" {
		apiObject.TagValues = aws.StringSlice([]string{v})
	}

	return apiObject
}

func expandLFTagPairs(tfList []interface{}) []*lakeformation.LFTagPair {
	if len(tfList) == 0 {
		return nil
	}

	var apiObjects []*lakeformation.LFTagPair

	for _, tfMapRaw := range tfList {
		tfMap, ok := tfMapRaw.(map[string]interface{})

		if !ok {
			continue
		}

		apiObject := expandLFTagPair(tfMap)

		if apiObject == nil {
			continue
		}

		apiObjects = append(apiObjects, apiObject)
	}

	return apiObjects
}

func flattenLFTagPair(apiObject *lakeformation.LFTagPair) map[string]interface{} {
	if apiObject == nil {
		return nil
	}

	tfMap := map[string]interface{}{}

	if v := apiObject.CatalogId; v != nil {
		tfMap["catalog_id"] = aws.StringValue(v)
	}

	if v := apiObject.TagKey; v != nil {
		tfMap["key"] = aws.StringValue(v)
	}

	if v := apiObject.TagValues; len(v) > 0 {
		tfMap["value"] = aws.StringValue(apiObject.TagValues[0])
	}

	return tfMap
}

func flattenLFTagPairs(apiObjects []*lakeformation.LFTagPair) []interface{} {
	if len(apiObjects) == 0 {
		return nil
	}

	var tfList []interface{}

	for _, apiObject := range apiObjects {
		if apiObject == nil {
			continue
		}

		tfList = append(tfList, flattenLFTagPair(apiObject))
	}

	return tfList
}
