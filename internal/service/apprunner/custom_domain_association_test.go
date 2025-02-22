package apprunner_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/service/apprunner"
	"github.com/hashicorp/aws-sdk-go-base/v2/awsv1shim/v2/tfawserr"
	sdkacctest "github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tfapprunner "github.com/hashicorp/terraform-provider-aws/internal/service/apprunner"
)

func TestAccAppRunnerCustomDomainAssociation_basic(t *testing.T) {
	ctx := acctest.Context(t)
	domain := os.Getenv("APPRUNNER_CUSTOM_DOMAIN")
	if domain == "" {
		t.Skip("Environment variable APPRUNNER_CUSTOM_DOMAIN is not set")
	}

	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_apprunner_custom_domain_association.test"
	serviceResourceName := "aws_apprunner_service.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); testAccPreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, apprunner.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckCustomDomainAssociationDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccCustomDomainAssociationConfig_basic(rName, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomDomainAssociationExists(ctx, resourceName),
					resource.TestCheckResourceAttr(resourceName, "certificate_validation_records.#", "3"),
					resource.TestCheckResourceAttrSet(resourceName, "dns_target"),
					resource.TestCheckResourceAttr(resourceName, "domain_name", domain),
					resource.TestCheckResourceAttr(resourceName, "enable_www_subdomain", "true"),
					resource.TestCheckResourceAttr(resourceName, "status", tfapprunner.CustomDomainAssociationStatusPendingCertificateDNSValidation),
					resource.TestCheckResourceAttrPair(resourceName, "service_arn", serviceResourceName, "arn"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"dns_target"},
			},
		},
	})
}

func TestAccAppRunnerCustomDomainAssociation_disappears(t *testing.T) {
	ctx := acctest.Context(t)
	domain := os.Getenv("APPRUNNER_CUSTOM_DOMAIN")
	if domain == "" {
		t.Skip("Environment variable APPRUNNER_CUSTOM_DOMAIN is not set")
	}

	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_apprunner_custom_domain_association.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); testAccPreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, apprunner.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckCustomDomainAssociationDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccCustomDomainAssociationConfig_basic(rName, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomDomainAssociationExists(ctx, resourceName),
					acctest.CheckResourceDisappears(ctx, acctest.Provider, tfapprunner.ResourceCustomDomainAssociation(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckCustomDomainAssociationDestroy(ctx context.Context) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "aws_apprunner_connection" {
				continue
			}

			conn := acctest.Provider.Meta().(*conns.AWSClient).AppRunnerConn(ctx)

			domainName, serviceArn, err := tfapprunner.CustomDomainAssociationParseID(rs.Primary.ID)

			if err != nil {
				return err
			}

			customDomain, err := tfapprunner.FindCustomDomain(ctx, conn, domainName, serviceArn)

			if tfawserr.ErrCodeEquals(err, apprunner.ErrCodeResourceNotFoundException) {
				continue
			}

			if err != nil {
				return err
			}

			if customDomain != nil {
				return fmt.Errorf("App Runner Custom Domain Association (%s) still exists", rs.Primary.ID)
			}
		}

		return nil
	}
}

func testAccCheckCustomDomainAssociationExists(ctx context.Context, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No App Runner Custom Domain Association ID is set")
		}

		domainName, serviceArn, err := tfapprunner.CustomDomainAssociationParseID(rs.Primary.ID)

		if err != nil {
			return err
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).AppRunnerConn(ctx)

		customDomain, err := tfapprunner.FindCustomDomain(ctx, conn, domainName, serviceArn)

		if err != nil {
			return err
		}

		if customDomain == nil {
			return fmt.Errorf("App Runner Custom Domain Association (%s) not found", rs.Primary.ID)
		}

		return nil
	}
}

func testAccCustomDomainAssociationConfig_basic(rName, domain string) string {
	return fmt.Sprintf(`
resource "aws_apprunner_service" "test" {
  service_name = %[1]q

  source_configuration {
    auto_deployments_enabled = false
    image_repository {
      image_configuration {
        port = "80"
      }
      image_identifier      = "public.ecr.aws/nginx/nginx:latest"
      image_repository_type = "ECR_PUBLIC"
    }
  }
}

resource "aws_apprunner_custom_domain_association" "test" {
  domain_name = %[2]q
  service_arn = aws_apprunner_service.test.arn
}
`, rName, domain)
}
