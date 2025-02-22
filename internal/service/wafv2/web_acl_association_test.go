package wafv2_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/service/wafv2"
	sdkacctest "github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/service/apigateway"
	tfwafv2 "github.com/hashicorp/terraform-provider-aws/internal/service/wafv2"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
)

func TestAccWAFV2WebACLAssociation_basic(t *testing.T) {
	ctx := acctest.Context(t)
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_wafv2_web_acl_association.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			acctest.PreCheckAPIGatewayTypeEDGE(t)
			testAccPreCheckScopeRegional(ctx, t)
		},
		ErrorCheck:               acctest.ErrorCheck(t, wafv2.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckWebACLAssociationDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccWebACLAssociationConfig_basic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckWebACLAssociationExists(ctx, resourceName),
					acctest.MatchResourceAttrRegionalARNNoAccount(resourceName, "resource_arn", "apigateway", regexp.MustCompile(fmt.Sprintf("/restapis/.*/stages/%s", rName))),
					acctest.MatchResourceAttrRegionalARN(resourceName, "web_acl_arn", "wafv2", regexp.MustCompile(fmt.Sprintf("regional/webacl/%s/.*", rName))),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccWAFV2WebACLAssociation_disappears(t *testing.T) {
	ctx := acctest.Context(t)
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_wafv2_web_acl_association.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			acctest.PreCheckAPIGatewayTypeEDGE(t)
			testAccPreCheckScopeRegional(ctx, t)
		},
		ErrorCheck:               acctest.ErrorCheck(t, wafv2.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckWebACLAssociationDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccWebACLAssociationConfig_basic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckWebACLAssociationExists(ctx, resourceName),
					acctest.CheckResourceDisappears(ctx, acctest.Provider, tfwafv2.ResourceWebACLAssociation(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccWebACLAssociationConfig_basic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckWebACLAssociationExists(ctx, resourceName),
					acctest.CheckResourceDisappears(ctx, acctest.Provider, apigateway.ResourceStage(), "aws_api_gateway_stage.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckWebACLAssociationDestroy(ctx context.Context) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "aws_wafv2_web_acl_association" {
				continue
			}

			_, resourceARN, err := tfwafv2.WebACLAssociationParseResourceID(rs.Primary.ID)

			if err != nil {
				return err
			}

			conn := acctest.Provider.Meta().(*conns.AWSClient).WAFV2Conn(ctx)

			_, err = tfwafv2.FindWebACLByResourceARN(ctx, conn, resourceARN)

			if tfresource.NotFound(err) {
				continue
			}

			if err != nil {
				return err
			}

			return fmt.Errorf("WAFv2 WebACL Association %s still exists", rs.Primary.ID)
		}

		return nil
	}
}

func testAccCheckWebACLAssociationExists(ctx context.Context, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No WAFv2 WebACL Association ID is set")
		}

		_, resourceARN, err := tfwafv2.WebACLAssociationParseResourceID(rs.Primary.ID)

		if err != nil {
			return err
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).WAFV2Conn(ctx)

		_, err = tfwafv2.FindWebACLByResourceARN(ctx, conn, resourceARN)

		return err
	}
}

func testAccWebACLAssociationConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "aws_api_gateway_stage" "test" {
  stage_name    = %[1]q
  rest_api_id   = aws_api_gateway_rest_api.test.id
  deployment_id = aws_api_gateway_deployment.test.id
}

resource "aws_api_gateway_rest_api" "test" {
  name = %[1]q
}

resource "aws_api_gateway_deployment" "test" {
  rest_api_id = aws_api_gateway_rest_api.test.id
  depends_on  = [aws_api_gateway_integration.test]
}

resource "aws_api_gateway_integration" "test" {
  rest_api_id = aws_api_gateway_rest_api.test.id
  resource_id = aws_api_gateway_resource.test.id
  http_method = aws_api_gateway_method.test.http_method
  type        = "MOCK"
}

resource "aws_api_gateway_resource" "test" {
  rest_api_id = aws_api_gateway_rest_api.test.id
  parent_id   = aws_api_gateway_rest_api.test.root_resource_id
  path_part   = "mytestresource"
}

resource "aws_api_gateway_method" "test" {
  rest_api_id   = aws_api_gateway_rest_api.test.id
  resource_id   = aws_api_gateway_resource.test.id
  http_method   = "GET"
  authorization = "NONE"
}

resource "aws_wafv2_web_acl" "test" {
  name  = %[1]q
  scope = "REGIONAL"

  default_action {
    allow {}
  }

  visibility_config {
    cloudwatch_metrics_enabled = false
    metric_name                = "friendly-metric-name"
    sampled_requests_enabled   = false
  }
}

resource "aws_wafv2_web_acl_association" "test" {
  resource_arn = aws_api_gateway_stage.test.arn
  web_acl_arn  = aws_wafv2_web_acl.test.arn
}
`, name)
}
