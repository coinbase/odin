########################################
###           ENVIRONMENT            ###
########################################
require_relative './vpc-resources'
env = environment('development')

project = project('coinbase', 'deploy-test') {
  environments 'development'
  tags {
    ProjectName "coinbase/deploy-test"
    ConfigName  "development"
    ServiceName "web"
    self[:org] = "coinbase"
    self[:project] = "deploy-test"
  }
}

# SECURITY GROUPS
elb_sg = project.resource("aws_security_group", "elb-web-app") {
  name "coinbase-deploy-test-development-web-elb-sg"
  description "Security Group for Web ELBs"
  vpc_id       env.vpc_id

  ingress {
    from_port    8000
    to_port      8000
    protocol     "tcp"
    cidr_blocks  ["10.0.0.0/16"]
  }

  egress {
    from_port        0
    to_port          0
    protocol         "-1"
    cidr_blocks      ["0.0.0.0/0"]
  }

  tags {
    Name "elb::coinbase/deploy-test::development"
  }
}

project.resource("aws_security_group", "web-app") {
  name "coinbase-deploy-test-development-web-ec2-sg"
  description "Security Group for Web EC2 instances"
  vpc_id       env.vpc_id

  ingress {
    from_port    8000
    to_port      8000
    protocol     "tcp"
    security_groups  [elb_sg]
  }

  ingress {
    from_port    22
    to_port      22
    protocol     "tcp"
    cidr_blocks  ["10.0.0.0/16"]
  }

  egress {
    from_port        0
    to_port          0
    protocol         "-1"
    cidr_blocks      ["0.0.0.0/0"]
  }

  tags {
    Name "ec2::coinbase/deploy-test::development"
  }
}

project.resource("aws_security_group", "default") {
  name "coinbase-deploy-test-development-ec2-default"
  description "Default Security Group"
  vpc_id       env.vpc_id

  tags {
    Name "ec2::default"
    ProjectName "_all"
    ConfigName  "development"
    ServiceName "_all"
  }
}

# ELB
project.resource("aws_elb", "web-app") {
  name             "coinbase-deploy-test-web-elb"
  internal         true
  security_groups  [elb_sg]
  subnets          [env.public_subnet_a_id, env.public_subnet_b_id]
  listener {
    instance_port     8000
    instance_protocol "http"
    lb_port           80
    lb_protocol       "http"
  }

  health_check {
    target "HTTP:8000/"
    healthy_threshold 2
    unhealthy_threshold 5
    interval 30
    timeout 10
  }

  tags {
    Name "elb:coinbase-deploy-test"
  }
}

# ALB
alb = project.resource("aws_lb", "web-app") {
  name            "coinbase-deploy-test-web-alb"
  internal         true
  security_groups  [elb_sg]
  subnets          [env.public_subnet_a_id, env.public_subnet_b_id]

  lifecycle {
    ignore_changes ["enable_cross_zone_load_balancing", "enable_http2"]
  }

  tags {
    Name "alb:coinbase-deploy-test"
  }
}


target_group = project.resource('aws_alb_target_group', "alb_tg") {
  name "coinbase-deploy-test-web-tg"
  port 8000
  protocol 'HTTP'
  vpc_id env.vpc_id

  health_check {
    timeout               10
    unhealthy_threshold   5
    healthy_threshold     2
    interval              30
    path                  '/'
  }

  lifecycle {
    ignore_changes ["proxy_protocol_v2"]
  }

  tags {
    Name "alb:coinbase-deploy-test:tg"
  }
}

project.resource("aws_alb_listener", 'alb_ln') {
  _load_balancer_name alb.name
  load_balancer_arn alb.to_ref("arn")
  port 80
  protocol "HTTP"

  default_action {
    target_group_arn target_group.to_id_or_ref
    self["type"] = "forward"
  }
}

# Instance Roles
role = project.resource('aws_iam_role', 'coinbase-deploy-test-dns') {
  name 'coinbase-deploy-test'
  path '/odin/coinbase/deploy-test/development/web/'
  assume_role_policy(%({
    "Version": "2012-10-17",
    "Statement": [
      {
        "Effect": "Allow",
        "Principal": {
          "Service": "ec2.amazonaws.com"
        },
        "Action": "sts:AssumeRole"
      }
    ]
  }))
}

project.resource('aws_iam_instance_profile', 'coinbase-deploy-test') {
  name 'coinbase-deploy-test'
  path '/odin/coinbase/deploy-test/development/web/'
  role role
}


default_role = project.resource('aws_iam_role', 'default-profile') {
  name 'default-profile'
  path '/odin/_all/_all/_all/'
  assume_role_policy(%({
    "Version": "2012-10-17",
    "Statement": [
      {
        "Effect": "Allow",
        "Principal": {
          "Service": "ec2.amazonaws.com"
        },
        "Action": "sts:AssumeRole"
      }
    ]
  }))
}

project.resource('aws_iam_instance_profile', 'default-profile') {
  name 'default-profile'
  path '/odin/_all/_all/_all/'
  role default_role
}

policy = project.resource('aws_iam_policy', 'az-policy') {
  name 'coinbase-deploy-test'
  policy '{
              "Version": "2012-10-17",
              "Statement": [
                {
                  "Effect": "Allow",
                  "Action": [
                    "ec2:DescribeAvailabilityZones"
                  ],
                  "Resource": ["*"]
                }
              ]
            }'
}

project.resource('aws_iam_policy_attachment', 'default-profile') {
  name 'default-profile'
  _policy policy
  roles [role, default_role]
}
