package acm_test

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
)

func TestAccACMCertificateValidation_basic(t *testing.T) {
	rootDomain := acctest.ACMCertificateDomainFromEnv(t)
	domain := acctest.ACMCertificateRandomSubDomain(rootDomain)
	certificateResourceName := "aws_acm_certificate.test"
	resourceName := "aws_acm_certificate_validation.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, acm.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAcmCertificateDestroy,
		Steps: []resource.TestStep{
			// Test that validation succeeds
			{
				Config: testAccAcmCertificateValidation_basic(rootDomain, domain),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(resourceName, "certificate_arn", certificateResourceName, "arn"),
				),
			},
		},
	})
}

func TestAccACMCertificateValidation_timeout(t *testing.T) {
	rootDomain := acctest.ACMCertificateDomainFromEnv(t)
	domain := acctest.ACMCertificateRandomSubDomain(rootDomain)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, acm.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAcmCertificateDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccAcmCertificateValidation_timeout(domain),
				ExpectError: regexp.MustCompile("Expected certificate to be issued but was in state PENDING_VALIDATION"),
			},
		},
	})
}

func TestAccACMCertificateValidation_validationRecordFQDNS(t *testing.T) {
	rootDomain := acctest.ACMCertificateDomainFromEnv(t)
	domain := acctest.ACMCertificateRandomSubDomain(rootDomain)
	certificateResourceName := "aws_acm_certificate.test"
	resourceName := "aws_acm_certificate_validation.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, acm.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAcmCertificateDestroy,
		Steps: []resource.TestStep{
			// Test that validation fails if given validation_fqdns don't match
			{
				Config:      testAccAcmCertificateValidation_validationRecordFqdnsWrongFqdn(domain),
				ExpectError: regexp.MustCompile("missing .+ DNS validation record: .+"),
			},
			// Test that validation succeeds with validation
			{
				Config: testAccAcmCertificateValidation_validationRecordFqdnsOneRoute53Record(rootDomain, domain),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(resourceName, "certificate_arn", certificateResourceName, "arn"),
				),
			},
		},
	})
}

func TestAccACMCertificateValidation_validationRecordFQDNSEmail(t *testing.T) {
	rootDomain := acctest.ACMCertificateDomainFromEnv(t)
	domain := acctest.ACMCertificateRandomSubDomain(rootDomain)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, acm.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAcmCertificateDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccAcmCertificateValidation_validationRecordFqdnsEmailValidation(domain),
				ExpectError: regexp.MustCompile("validation_record_fqdns is only valid for DNS validation"),
			},
		},
	})
}

func TestAccACMCertificateValidation_validationRecordFQDNSRoot(t *testing.T) {
	rootDomain := acctest.ACMCertificateDomainFromEnv(t)
	certificateResourceName := "aws_acm_certificate.test"
	resourceName := "aws_acm_certificate_validation.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, acm.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAcmCertificateDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAcmCertificateValidation_validationRecordFqdnsOneRoute53Record(rootDomain, rootDomain),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(resourceName, "certificate_arn", certificateResourceName, "arn"),
				),
			},
		},
	})
}

func TestAccACMCertificateValidation_validationRecordFQDNSRootAndWildcard(t *testing.T) {
	rootDomain := acctest.ACMCertificateDomainFromEnv(t)
	wildcardDomain := fmt.Sprintf("*.%s", rootDomain)
	certificateResourceName := "aws_acm_certificate.test"
	resourceName := "aws_acm_certificate_validation.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, acm.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAcmCertificateDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAcmCertificateValidation_validationRecordFqdnsTwoRoute53Records(rootDomain, rootDomain, strconv.Quote(wildcardDomain)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(resourceName, "certificate_arn", certificateResourceName, "arn"),
				),
			},
		},
	})
}

func TestAccACMCertificateValidation_validationRecordFQDNSSan(t *testing.T) {
	rootDomain := acctest.ACMCertificateDomainFromEnv(t)
	domain := acctest.ACMCertificateRandomSubDomain(rootDomain)
	sanDomain := acctest.ACMCertificateRandomSubDomain(rootDomain)
	certificateResourceName := "aws_acm_certificate.test"
	resourceName := "aws_acm_certificate_validation.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, acm.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAcmCertificateDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAcmCertificateValidation_validationRecordFqdnsTwoRoute53Records(rootDomain, domain, strconv.Quote(sanDomain)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(resourceName, "certificate_arn", certificateResourceName, "arn"),
				),
			},
		},
	})
}

func TestAccACMCertificateValidation_validationRecordFQDNSWildcard(t *testing.T) {
	rootDomain := acctest.ACMCertificateDomainFromEnv(t)
	wildcardDomain := fmt.Sprintf("*.%s", rootDomain)
	certificateResourceName := "aws_acm_certificate.test"
	resourceName := "aws_acm_certificate_validation.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, acm.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAcmCertificateDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAcmCertificateValidation_validationRecordFqdnsOneRoute53Record(rootDomain, wildcardDomain),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(resourceName, "certificate_arn", certificateResourceName, "arn"),
				),
			},
		},
	})
}

func TestAccACMCertificateValidation_validationRecordFQDNSWildcardAndRoot(t *testing.T) {
	rootDomain := acctest.ACMCertificateDomainFromEnv(t)
	wildcardDomain := fmt.Sprintf("*.%s", rootDomain)
	certificateResourceName := "aws_acm_certificate.test"
	resourceName := "aws_acm_certificate_validation.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		ErrorCheck:   acctest.ErrorCheck(t, acm.EndpointsID),
		Providers:    acctest.Providers,
		CheckDestroy: testAccCheckAcmCertificateDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAcmCertificateValidation_validationRecordFqdnsTwoRoute53Records(rootDomain, wildcardDomain, strconv.Quote(rootDomain)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(resourceName, "certificate_arn", certificateResourceName, "arn"),
				),
			},
		},
	})
}

func testAccAcmCertificateValidation_basic(rootZoneDomain, domainName string) string {
	return fmt.Sprintf(`
resource "aws_acm_certificate" "test" {
  domain_name       = %[1]q
  validation_method = "DNS"
}

data "aws_route53_zone" "test" {
  name         = %[2]q
  private_zone = false
}

#
# for_each acceptance testing requires SDKv2
#
# resource "aws_route53_record" "test" {
#   for_each = {
#     for dvo in aws_acm_certificate.test.domain_validation_options: dvo.domain_name => {
#       name   = dvo.resource_record_name
#       record = dvo.resource_record_value
#       type   = dvo.resource_record_type
#     }
#   }

#   allow_overwrite = true
#   name            = each.value.name
#   records         = [each.value.record]
#   ttl             = 60
#   type            = each.value.type
#   zone_id         = data.aws_route53_zone.test.zone_id
# }

resource "aws_route53_record" "test" {
  allow_overwrite = true
  name            = tolist(aws_acm_certificate.test.domain_validation_options)[0].resource_record_name
  records         = [tolist(aws_acm_certificate.test.domain_validation_options)[0].resource_record_value]
  ttl             = 60
  type            = tolist(aws_acm_certificate.test.domain_validation_options)[0].resource_record_type
  zone_id         = data.aws_route53_zone.test.zone_id
}

resource "aws_acm_certificate_validation" "test" {
  depends_on = [aws_route53_record.test]

  certificate_arn = aws_acm_certificate.test.arn
}
`, domainName, rootZoneDomain)
}

func testAccAcmCertificateValidation_timeout(domainName string) string {
	return fmt.Sprintf(`
resource "aws_acm_certificate" "test" {
  domain_name       = %[1]q
  validation_method = "DNS"
}

resource "aws_acm_certificate_validation" "test" {
  certificate_arn = aws_acm_certificate.test.arn

  timeouts {
    create = "5s"
  }
}
`, domainName)
}

func testAccAcmCertificateValidation_validationRecordFqdnsEmailValidation(domainName string) string {
	return fmt.Sprintf(`
resource "aws_acm_certificate" "test" {
  domain_name       = %[1]q
  validation_method = "EMAIL"
}

resource "aws_acm_certificate_validation" "test" {
  certificate_arn         = aws_acm_certificate.test.arn
  validation_record_fqdns = ["wrong-validation-fqdn.example.com"]
}
`, domainName)
}

func testAccAcmCertificateValidation_validationRecordFqdnsOneRoute53Record(rootZoneDomain, domainName string) string {
	return fmt.Sprintf(`
resource "aws_acm_certificate" "test" {
  domain_name       = %[1]q
  validation_method = "DNS"
}

data "aws_route53_zone" "test" {
  name         = %[2]q
  private_zone = false
}

#
# for_each acceptance testing requires SDKv2
#
# resource "aws_route53_record" "test" {
#   for_each = {
#     for dvo in aws_acm_certificate.test.domain_validation_options: dvo.domain_name => {
#       name   = dvo.resource_record_name
#       record = dvo.resource_record_value
#       type   = dvo.resource_record_type
#     }
#   }

#   allow_overwrite = true
#   name            = each.value.name
#   records         = [each.value.record]
#   ttl             = 60
#   type            = each.value.type
#   zone_id         = data.aws_route53_zone.test.zone_id
# }

# resource "aws_acm_certificate_validation" "test" {
#   certificate_arn         = aws_acm_certificate.test.arn
#   validation_record_fqdns = [for record in aws_route53_record.test: record.fqdn]
# }

resource "aws_route53_record" "test" {
  allow_overwrite = true
  name            = tolist(aws_acm_certificate.test.domain_validation_options)[0].resource_record_name
  records         = [tolist(aws_acm_certificate.test.domain_validation_options)[0].resource_record_value]
  ttl             = 60
  type            = tolist(aws_acm_certificate.test.domain_validation_options)[0].resource_record_type
  zone_id         = data.aws_route53_zone.test.zone_id
}

resource "aws_acm_certificate_validation" "test" {
  certificate_arn         = aws_acm_certificate.test.arn
  validation_record_fqdns = [aws_route53_record.test.fqdn]
}
`, domainName, rootZoneDomain)
}

func testAccAcmCertificateValidation_validationRecordFqdnsTwoRoute53Records(rootZoneDomain, domainName, subjectAlternativeNames string) string {
	return fmt.Sprintf(`
resource "aws_acm_certificate" "test" {
  domain_name               = %[1]q
  subject_alternative_names = [%[2]s]
  validation_method         = "DNS"
}

data "aws_route53_zone" "test" {
  name         = %[3]q
  private_zone = false
}

#
# for_each acceptance testing requires SDKv2
#
# resource "aws_route53_record" "test" {
#   for_each = {
#     for dvo in aws_acm_certificate.test.domain_validation_options: dvo.domain_name => {
#       name   = dvo.resource_record_name
#       record = dvo.resource_record_value
#       type   = dvo.resource_record_type
#     }
#   }

#   allow_overwrite = true
#   name            = each.value.name
#   records         = [each.value.record]
#   ttl             = 60
#   type            = each.value.type
#   zone_id         = data.aws_route53_zone.test.zone_id
# }

# resource "aws_acm_certificate_validation" "test" {
#   certificate_arn         = aws_acm_certificate.test.arn
#   validation_record_fqdns = [for record in aws_route53_record.test: record.fqdn]
# }

resource "aws_route53_record" "test" {
  allow_overwrite = true
  name            = tolist(aws_acm_certificate.test.domain_validation_options)[0].resource_record_name
  records         = [tolist(aws_acm_certificate.test.domain_validation_options)[0].resource_record_value]
  ttl             = 60
  type            = tolist(aws_acm_certificate.test.domain_validation_options)[0].resource_record_type
  zone_id         = data.aws_route53_zone.test.zone_id
}

resource "aws_route53_record" "test2" {
  allow_overwrite = true
  name            = tolist(aws_acm_certificate.test.domain_validation_options)[1].resource_record_name
  records         = [tolist(aws_acm_certificate.test.domain_validation_options)[1].resource_record_value]
  ttl             = 60
  type            = tolist(aws_acm_certificate.test.domain_validation_options)[1].resource_record_type
  zone_id         = data.aws_route53_zone.test.zone_id
}

resource "aws_acm_certificate_validation" "test" {
  certificate_arn         = aws_acm_certificate.test.arn
  validation_record_fqdns = [aws_route53_record.test.fqdn, aws_route53_record.test2.fqdn]
}
`, domainName, subjectAlternativeNames, rootZoneDomain)
}

func testAccAcmCertificateValidation_validationRecordFqdnsWrongFqdn(domainName string) string {
	return fmt.Sprintf(`
resource "aws_acm_certificate" "test" {
  domain_name       = %[1]q
  validation_method = "DNS"
}

resource "aws_acm_certificate_validation" "test" {
  certificate_arn         = aws_acm_certificate.test.arn
  validation_record_fqdns = ["wrong-validation-fqdn.example.com"]
}
`, domainName)
}
