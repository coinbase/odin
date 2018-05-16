package alarms

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/cloudwatch"
	fuzz "github.com/google/gofuzz"
)

func Test_SetAlarmDescription_Fuzz(t *testing.T) {
	// Making sure the descrption never panics
	for i := 0; i < 50; i++ {
		f := fuzz.New()
		aii := cloudwatch.PutMetricAlarmInput{}
		ai := AlarmInput{&aii}
		f.Fuzz(&aii)
		ai.SetAlarmDescription()
	}
}
