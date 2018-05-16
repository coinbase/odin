########################################
###           ENVIRONMENT            ###
########################################

env = environment('step') {
  region      ENV.fetch('AWS_REGION')
  account_id  ENV.fetch('AWS_ACCOUNT_ID')
}

########################################
###            PROJECT               ###
########################################
config = "development"
project = project('coinbase', 'odin') {
  environments 'step'
  tags {
    ProjectName "coinbase/odin"
    ConfigName config
    DeployWith "step-deployer"
    self[:org] = "coinbase"
    self[:project] = "odin"
  }
}

step_name = "#{project.org}-#{project.name}"
step_role_name =  "#{step_name}-step-function-role"
lambda_role_name = "#{step_name}-lambda-role"
lambda_assumed_role_name = "#{step_name}-assumed"
s3_bucket_name = "#{step_name}-#{env.account_id}"

########################################
###               S3                 ###
########################################

# Bucket for deploy artifacts
project.resource('aws_s3_bucket', 'odin') {
  bucket s3_bucket_name
  acl("private")
}

########################################
###         Step Function            ###
########################################

step_role = project.resource('aws_iam_role', step_role_name) {
  name step_role_name
  path "/step/#{project.full_name}/#{config}/"
  assume_role_policy(
    {
      "Version": "2012-10-17",
      "Statement": [
        {
          "Effect": "Allow",
          "Principal": {
            "Service": "states.us-east-1.amazonaws.com"
          },
          "Action": "sts:AssumeRole"
        }
      ]
    }.to_json
  )
}

project.resource('aws_iam_role_policy', step_role_name) {
  depends_on [step_role.terraform_name]
  name step_role_name
  role step_role_name
  policy(
    {
      "Version": "2012-10-17",
      "Statement": [
        {
          "Effect": "Allow",
          "Action": [
            "lambda:InvokeFunction"
          ],
          "Resource": "*"
        }
    ]
    }.to_json
  )
}

project.resource("aws_sfn_state_machine", "sfn_state_machine") {
  depends_on [step_role.terraform_name]
  name       step_name
  role_arn   step_role.to_ref("arn")
  definition '{
    "StartAt": "Noop",
    "States": {
      "Noop": {
        "Type": "Pass",
        "End": true
      }
    }
  }'

  lifecycle {
    ignore_changes ["definition"] # Ignore changes here
  }
}

########################################
###         IAM  Role                ###
########################################

lambda_role = project.resource('aws_iam_role', lambda_role_name) {
  name lambda_role_name
  path "/"
  assume_role_policy(
    {
      "Version": "2012-10-17",
      "Statement": [
        {
          "Sid": "",
          "Effect": "Allow",
          "Principal": {
            "Service": "lambda.amazonaws.com"
          },
          "Action": "sts:AssumeRole"
        }
      ]
    }.to_json
  )
}

project.resource('aws_iam_role_policy', lambda_role_name) {
  depends_on [lambda_role.terraform_name]
  name lambda_role_name
  role lambda_role_name
  policy({
    "Version": "2012-10-17",
    "Statement": [
      {
          "Sid": "",
          "Effect": "Allow",
          "Resource": "arn:aws:iam::*:role/#{lambda_assumed_role_name}",
          "Action": "sts:AssumeRole"
        },
        {
          "Effect": "Allow",
          "Action": [
            "logs:CreateLogGroup",
            "logs:CreateLogStream",
            "logs:PutLogEvents"
          ],
          "Resource": "arn:aws:logs:*:*:log-group:/aws/lambda/*"
        },
        # GET OBJECTS FROM EXACTLY ONE S3 BUCKET
        {
          "Effect": "Allow",
          "Action": [
            "s3:GetObject*",
            "s3:PutObject*",
            "s3:List*",
            "s3:DeleteObject*"
          ],
          "Resource": [
            "arn:aws:s3:::#{s3_bucket_name}/*",
            "arn:aws:s3:::#{s3_bucket_name}"
          ]
        },
        {
          "Effect": "Deny",
          "Action": [
            "s3:*",
          ],
          "NotResource": [
            "arn:aws:s3:::#{s3_bucket_name}/*",
            "arn:aws:s3:::#{s3_bucket_name}"
          ]
        }
    ]
  }.to_json)
}

# Assumed Role Policy: if you put this role in any account then you will be able to deploy there
lambda_assumed_role = project.resource('aws_iam_role', lambda_assumed_role_name) {
  depends_on [lambda_role.terraform_name]
  name lambda_assumed_role_name
  path "/"
  assume_role_policy(
    {
      "Version": "2012-10-17",
      "Statement": [
        {
          "Effect": "Allow",
          "Principal": {
            "AWS": "arn:aws:iam::#{env.account_id}:role/#{lambda_role_name}"
          },
          "Action": "sts:AssumeRole"
        }
      ]
    }.to_json
  )
}

project.resource('aws_iam_role_policy', lambda_assumed_role_name) {
  depends_on [lambda_role.terraform_name, lambda_assumed_role.terraform_name]
  name lambda_assumed_role_name
  role lambda_assumed_role_name
  policy(
    {
      "Version": "2012-10-17",
      "Statement": [
        # READ/DEPLOY Methods
        {
          "Effect": "Allow",
          "Action": [
            "iam:GetRole",
            "iam:PassRole",
            "iam:GetInstanceProfile",

            "ec2:DescribeImages",
            "ec2:RunInstances",
            "ec2:DescribeSubnets",
            "ec2:DescribeSecurityGroups",

            "elasticloadbalancing:DescribeLoadBalancerAttributes",
            "elasticloadbalancing:DescribeLoadBalancers",
            "elasticloadbalancing:DescribeTargetGroupAttributes",
            "elasticloadbalancing:DescribeTags",
            "elasticloadbalancing:DescribeTargetHealth",
            "elasticloadbalancing:DescribeTargetGroups",
            "elasticloadbalancing:DescribeLoadBalancerPolicies",
            "elasticloadbalancing:DescribeLoadBalancerPolicyTypes",
            "elasticloadbalancing:DescribeInstanceHealth",

            "cloudwatch:PutMetricAlarm",
            "cloudwatch:DeleteAlarms",
            "cloudwatch:DescribeAlarms",

            "sns:GetTopicAttributes",

            "autoscaling:*"
          ],
          "Resource": "*",
          "Condition": {
            "Bool": {
              "aws:SecureTransport": "true"
            }
          }
        }
      ]
    }.to_json
  )
}

########################################
###            Lambda                ###
########################################

lambda_function = project.resource("aws_lambda_function", step_name) {
  function_name step_name
  description step_name

  role lambda_role.to_ref('arn')

  lifecycle {
    ignore_changes ["environment", "filename", "source_code_hash"]
  }

  filename File.expand_path(File.dirname(__FILE__)) + '/lambda.zip'
  handler "lambda"
  memory_size 128
  runtime "go1.x"
  timeout "300"
  publish "true"
}

