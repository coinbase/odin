package client

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/sfn"
	"github.com/coinbase/odin/aws/mocks"
	"github.com/coinbase/step/utils/to"
	"github.com/stretchr/testify/assert"
)

func Test_Halt(t *testing.T) {
	awsc := mocks.MockAWS()
	r := minimalRelease(t)

	r.Release.SetDefaults(to.Strp("region"), to.Strp("accountid"), "")

	awsc.SFN.ListExecutionsResp = &sfn.ListExecutionsOutput{
		Executions: []*sfn.ExecutionListItem{
			&sfn.ExecutionListItem{
				Name:         r.ExecutionName(),
				ExecutionArn: to.Strp("arn"),
				StartDate:    to.Timep(time.Now()),
			},
		},
	}

	err := halt(awsc, r, to.Strp("deployerARN"))
	assert.NoError(t, err)
}
