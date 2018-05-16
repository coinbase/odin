package asg

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/autoscaling"
	fuzz "github.com/google/gofuzz"
)

func Test_Defaults(t *testing.T) {
	for i := 0; i < 50; i++ {
		f := fuzz.New()
		aii := autoscaling.CreateAutoScalingGroupInput{}
		ai := Input{&aii}
		f.Fuzz(&aii)
		ai.SetDefaults()
	}
}
