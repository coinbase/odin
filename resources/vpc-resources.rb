env = environment('development') {
  region      ENV.fetch('AWS_REGION')
  account_id  ENV.fetch('AWS_ACCOUNT_ID')
}

vpc = env.resource('aws_vpc', "main") {
  cidr_block "10.0.0.0/16"
  tags {
    Name "test_vpc"
  }
}

public_subnet_a = env.resource("aws_subnet", "public_subnet_a") {
  vpc_id                  vpc.to_ref
  cidr_block              "10.0.10.0/24"
  availability_zone       "#{env.region}a"
  map_public_ip_on_launch false

  tags {
    Name "test_public_subnet_a"
  }
}

public_subnet_b = env.resource("aws_subnet", "public_subnet_b") {
  vpc_id                  vpc.to_ref
  cidr_block              "10.0.11.0/24"
  availability_zone       "#{env.region}b"
  map_public_ip_on_launch false

  tags {
    Name "test_public_subnet_b"
  }
}

private_subnet_a = env.resource("aws_subnet", "private_subnet_a") {
  vpc_id                  vpc.to_ref
  cidr_block              "10.0.20.0/24"
  availability_zone       "#{env.region}a"

  tags {
    Name "test_private_subnet_a"
    DeployWith "odin"
  }
}

private_subnet_b = env.resource("aws_subnet", "private_subnet_b") {
  vpc_id                  vpc.to_ref
  cidr_block              "10.0.21.0/24"
  availability_zone       "#{env.region}b"

  tags {
    Name "test_private_subnet_b"
    DeployWith "odin"
  }
}

ig = env.resource("aws_internet_gateway", "internet_gateway") {
  vpc_id vpc.to_ref
  tags {
    Name "test_internet_gateway"
  }
}

public_routetable = env.resource("aws_route_table", "public_routetable") {
  vpc_id vpc.to_ref

  route {
    cidr_block "0.0.0.0/0"
    gateway_id ig.to_ref
  }

  tags {
    Name "test_public_routetable"
  }
}

eip = env.resource("aws_eip", "eip_4_nat") {
  tags {
    Name "test_eip"
  }
}

nat = env.resource("aws_nat_gateway", "nat") {
  allocation_id eip.to_ref
  subnet_id     public_subnet_a.to_ref
  tags {
    Name "test_nat_gateway"
  }
}

private_routetable = env.resource("aws_route_table", "private_routetable") {
  vpc_id vpc.to_ref

  route {
    cidr_block     "0.0.0.0/0"
    nat_gateway_id nat.to_ref
  }

  tags {
    Name "test_private_routetable"
  }
}

env.resource("aws_route_table_association", "public_subnet_a") {
  subnet      public_subnet_a
  route_table public_routetable
}

env.resource("aws_route_table_association", "public_subnet_b") {
  subnet      public_subnet_b
  route_table public_routetable
}

env.resource("aws_route_table_association", "private_subnet_a") {
  subnet      private_subnet_a
  route_table private_routetable
}

env.resource("aws_route_table_association", "private_subnet_b") {
  subnet      private_subnet_b
  route_table private_routetable
}

env.vpc_id = vpc.to_ref
env.private_subnet_a_id = private_subnet_a.to_ref
env.public_subnet_a_id = public_subnet_a.to_ref
env.private_subnet_b_id = private_subnet_b.to_ref
env.public_subnet_b_id = public_subnet_b.to_ref

#####
# LIFECYCLE HOOK RESOURCES
#####

env.resource('aws_sns_topic', 'asg_lifecycle_hooks') {
  display_name 'asg_lifecycle_hooks'
  name         'asg_lifecycle_hooks'
}

life_cycle_role = env.resource('aws_iam_role', 'asg_lifecycle_hooks') {
  name 'asg_lifecycle_hooks'
  assume_role_policy '{
    "Version": "2012-10-17",
    "Statement": [
      {
        "Sid": "",
        "Effect": "Allow",
        "Principal": {
          "Service": "autoscaling.amazonaws.com"
        },
        "Action": "sts:AssumeRole"
      }
    ]
  }'
}

life_cycle_policy = env.resource('aws_iam_policy', 'asg_lifecycle_hooks') {
  name 'asg_lifecycle_hooks'
  policy '{
              "Version": "2012-10-17",
              "Statement": [{
                  "Effect": "Allow",
                  "Resource": "*",
                  "Action": [
                      "sqs:SendMessage",
                      "sqs:GetQueueUrl",
                      "sns:Publish"
                  ]
                }
              ]
          }'
}

env.resource('aws_iam_policy_attachment', 'asg_lifecycle_hooks') {
  name 'asg_lifecycle_hooks'
  _policy life_cycle_policy
  roles [life_cycle_role]
}
