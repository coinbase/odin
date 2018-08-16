# GeoEngineer Resources For Step Function Deployer
# GEO_ENV=development bundle exec geo apply resources/odin.rb

########################################
###           ENVIRONMENT            ###
########################################

env = environment('development') {
  region      ENV.fetch('AWS_REGION')
  account_id  ENV.fetch('AWS_ACCOUNT_ID')
}

########################################
###            PROJECT               ###
########################################
project = project('coinbase', 'odin') {
  environments 'development'
  tags {
    ProjectName "coinbase/odin"
    ConfigName "development"
    DeployWith "step-deployer"
    self[:org] = "coinbase"
    self[:project] = "odin"
  }
}

context = {
  assumed_role_name: "coinbase-odin-assumed",
  assumable_from: [ ENV['AWS_ACCOUNT_ID'] ],
  assumed_policy_file: "#{__dir__}/odin_assumed_policy.json.erb"
}

project.from_template('bifrost_deployer', 'odin', {
  lambda_policy_file: "#{__dir__}/odin_lambda_policy.json.erb",
  lambda_policy_context: context
})

# The assumed role exists in all environments
project.from_template('step_assumed', 'coinbase-odin-assumed', context)
