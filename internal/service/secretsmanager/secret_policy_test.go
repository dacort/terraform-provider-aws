package secretsmanager_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/hashicorp/aws-sdk-go-base/v2/awsv1shim/v2/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	sdkacctest "github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tfsecretsmanager "github.com/hashicorp/terraform-provider-aws/internal/service/secretsmanager"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
)

func TestAccSecretsManagerSecretPolicy_basic(t *testing.T) {
	ctx := acctest.Context(t)
	var policy secretsmanager.GetResourcePolicyOutput
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_secretsmanager_secret_policy.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); testAccPreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, secretsmanager.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckSecretPolicyDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccSecretPolicyConfig_basic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSecretPolicyExists(ctx, resourceName, &policy),
					resource.TestMatchResourceAttr(resourceName, "policy",
						regexp.MustCompile(`{"Action":"secretsmanager:GetSecretValue".+`)),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"block_public_policy"},
			},
			{
				Config: testAccSecretPolicyConfig_updated(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSecretPolicyExists(ctx, resourceName, &policy),
					resource.TestMatchResourceAttr(resourceName, "policy",
						regexp.MustCompile(`{"Action":"secretsmanager:\*".+`)),
				),
			},
		},
	})
}

func TestAccSecretsManagerSecretPolicy_blockPublicPolicy(t *testing.T) {
	ctx := acctest.Context(t)
	var policy secretsmanager.GetResourcePolicyOutput
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_secretsmanager_secret_policy.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); testAccPreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, secretsmanager.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckSecretPolicyDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccSecretPolicyConfig_block(rName, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSecretPolicyExists(ctx, resourceName, &policy),
					resource.TestCheckResourceAttr(resourceName, "block_public_policy", "true"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"block_public_policy"},
			},
			{
				Config: testAccSecretPolicyConfig_block(rName, false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSecretPolicyExists(ctx, resourceName, &policy),
					resource.TestCheckResourceAttr(resourceName, "block_public_policy", "false"),
				),
			},
			{
				Config: testAccSecretPolicyConfig_block(rName, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSecretPolicyExists(ctx, resourceName, &policy),
					resource.TestCheckResourceAttr(resourceName, "block_public_policy", "true"),
				),
			},
		},
	})
}

func TestAccSecretsManagerSecretPolicy_disappears(t *testing.T) {
	ctx := acctest.Context(t)
	var policy secretsmanager.GetResourcePolicyOutput
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_secretsmanager_secret_policy.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); testAccPreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, secretsmanager.EndpointsID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckSecretPolicyDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccSecretPolicyConfig_basic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSecretPolicyExists(ctx, resourceName, &policy),
					acctest.CheckResourceDisappears(ctx, acctest.Provider, tfsecretsmanager.ResourceSecretPolicy(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckSecretPolicyDestroy(ctx context.Context) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acctest.Provider.Meta().(*conns.AWSClient).SecretsManagerConn(ctx)

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "aws_secretsmanager_secret_policy" {
				continue
			}

			secretInput := &secretsmanager.DescribeSecretInput{
				SecretId: aws.String(rs.Primary.ID),
			}

			var output *secretsmanager.DescribeSecretOutput

			err := retry.RetryContext(ctx, tfsecretsmanager.PropagationTimeout, func() *retry.RetryError {
				var err error
				output, err = conn.DescribeSecretWithContext(ctx, secretInput)

				if err != nil {
					return retry.NonRetryableError(err)
				}

				if output != nil && output.DeletedDate == nil {
					return retry.RetryableError(fmt.Errorf("Secret %q still exists", rs.Primary.ID))
				}

				return nil
			})

			if tfresource.TimedOut(err) {
				output, err = conn.DescribeSecretWithContext(ctx, secretInput)
			}

			if tfawserr.ErrCodeEquals(err, secretsmanager.ErrCodeResourceNotFoundException) {
				continue
			}

			if err != nil {
				return err
			}

			if output != nil && output.DeletedDate == nil {
				return fmt.Errorf("Secret %q still exists", rs.Primary.ID)
			}

			input := &secretsmanager.GetResourcePolicyInput{
				SecretId: aws.String(rs.Primary.ID),
			}

			_, err = conn.GetResourcePolicyWithContext(ctx, input)

			if tfawserr.ErrCodeEquals(err, secretsmanager.ErrCodeResourceNotFoundException) ||
				tfawserr.ErrMessageContains(err, secretsmanager.ErrCodeInvalidRequestException,
					"You can't perform this operation on the secret because it was marked for deletion.") {
				continue
			}

			if err != nil {
				return err
			}
		}

		return nil
	}
}

func testAccCheckSecretPolicyExists(ctx context.Context, resourceName string, policy *secretsmanager.GetResourcePolicyOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).SecretsManagerConn(ctx)
		input := &secretsmanager.GetResourcePolicyInput{
			SecretId: aws.String(rs.Primary.ID),
		}

		output, err := conn.GetResourcePolicyWithContext(ctx, input)

		if err != nil {
			return err
		}

		if output == nil {
			return fmt.Errorf("Secret Policy %q does not exist", rs.Primary.ID)
		}

		*policy = *output

		return nil
	}
}

func testAccSecretPolicyConfig_basic(rName string) string {
	return fmt.Sprintf(`
resource "aws_iam_role" "test" {
  name               = %[1]q
  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_secretsmanager_secret" "test" {
  name = %[1]q
}

resource "aws_secretsmanager_secret_policy" "test" {
  secret_arn = aws_secretsmanager_secret.test.arn

  policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
	{
	  "Sid": "EnableAllPermissions",
	  "Effect": "Allow",
	  "Principal": {
		"AWS": "${aws_iam_role.test.arn}"
	  },
	  "Action": "secretsmanager:GetSecretValue",
	  "Resource": "*"
	}
  ]
}
POLICY
}
`, rName)
}

func testAccSecretPolicyConfig_updated(rName string) string {
	return fmt.Sprintf(`
resource "aws_secretsmanager_secret" "test" {
  name = %[1]q
}

resource "aws_secretsmanager_secret_policy" "test" {
  secret_arn = aws_secretsmanager_secret.test.arn

  policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
	{
	  "Sid": "EnableAllPermissions",
	  "Effect": "Allow",
	  "Principal": {
		"AWS": "*"
	  },
	  "Action": "secretsmanager:*",
	  "Resource": "*"
	}
  ]
}
POLICY
}
`, rName)
}

func testAccSecretPolicyConfig_block(rName string, block bool) string {
	return fmt.Sprintf(`
resource "aws_iam_role" "test" {
  name               = %[1]q
  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_secretsmanager_secret" "test" {
  name = %[1]q
}

resource "aws_secretsmanager_secret_policy" "test" {
  secret_arn          = aws_secretsmanager_secret.test.arn
  block_public_policy = %[2]t

  policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
	{
	  "Sid": "EnableAllPermissions",
	  "Effect": "Allow",
	  "Principal": {
		"AWS": "${aws_iam_role.test.arn}"
	  },
	  "Action": "secretsmanager:GetSecretValue",
	  "Resource": "*"
	}
  ]
}
POLICY
}
`, rName, block)
}
