package client

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/coinbase/odin/aws"
	"github.com/coinbase/step/bifrost"
	"github.com/coinbase/step/execution"
	"github.com/coinbase/step/utils/to"
)

// Release is the Data Structure passed between Client to Deployer
type FailedRelease struct {
	// Useful information from AWS
	AwsAccountID *string `json:"aws_account_id,omitempty"`
	AwsRegion    *string `json:"aws_region,omitempty"`

	ProjectName *string `json:"project_name,omitempty"`
	ConfigName  *string `json:"config_name,omitempty"`

	// Where the previous Catch Error should be located
	Error *bifrost.ReleaseError `json:"error,omitempty"`
}

// List the recent failures and their causes
func Failures(step_fn *string) error {
	region, accountID := to.RegionAccount()

	deployerARN := to.StepArn(region, accountID, step_fn)

	awsc := &aws.ClientsStr{}

	return failures(awsc.SFNClient(nil, nil, nil), deployerARN)
}

func failures(sfnc aws.SFNAPI, arn *string) error {
	execs, err := execution.ExecutionsAfter(sfnc, arn, to.Strp("FAILED"), time.Now().Add((-3*24)*time.Hour))

	if err != nil {
		return err
	}

	for _, e := range execs {
		sd, err := e.GetStateDetails(sfnc)
		if err != nil {
			return err
		}

		release := FailedRelease{}
		if err := json.Unmarshal([]byte(*sd.LastOutput), &release); err != nil {
			j, _ := to.PrettyJSON(*sd.LastOutput)
			fmt.Println(j)
			continue
		}

		cause := ""
		err_json := map[string]string{}

		if release.Error != nil {
			err = json.Unmarshal([]byte(*release.Error.Cause), &err_json)

			if err != nil {
				fmt.Println(err)
				cause = *release.Error.Cause
			} else {
				cause = err_json["errorMessage"]
			}
		}

		fmt.Println(fmt.Printf("%v -- %v -- %q", *sd.LastStateName, *e.Name, cause))
	}

	return nil
}
