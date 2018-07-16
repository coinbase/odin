package client

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/coinbase/odin/aws"
	"github.com/coinbase/odin/deployer/models"
	"github.com/coinbase/step/execution"
	"github.com/coinbase/step/utils/to"
)

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

		release := models.Release{}
		if err := json.Unmarshal([]byte(*sd.LastOutput), &release); err != nil {
			j, _ := to.PrettyJSON(*sd.LastOutput)
			fmt.Println(j)
			continue
		}

		cause := ""
		err_json := map[string]string{}

		err = json.Unmarshal([]byte(*release.Error.Cause), &err_json)
		if err != nil {
			fmt.Println(err)
			cause = *release.Error.Cause
		} else {
			cause = err_json["errorMessage"]
		}

		fmt.Println(fmt.Printf("%v -- %v", *e.Name, cause))
	}

	return nil
}
