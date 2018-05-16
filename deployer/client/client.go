package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"sort"
	"strings"

	"github.com/coinbase/odin/deployer/models"
	"github.com/coinbase/step/execution"
	"github.com/coinbase/step/utils/is"
	"github.com/coinbase/step/utils/to"
)

// executionPrefix returns
func executionPrefix(release *models.Release) string {
	pn := strings.Replace(*release.ProjectName, "/", "-", -1)
	return fmt.Sprintf("deploy-%v-%v-", pn, *release.ConfigName)
}

// executionName returns
func executionName(release *models.Release) *string {
	return to.TimeUUID(executionPrefix(release))
}

// validateClientAttributes returns
func validateClientAttributes(release *models.Release) error {
	if release == nil {
		// Extra paranoid
		return fmt.Errorf("Release is nil")
	}

	if is.EmptyStr(release.ProjectName) {
		return fmt.Errorf("ProjectName must be defined")
	}

	if is.EmptyStr(release.ConfigName) {
		return fmt.Errorf("ConfigName must be defined")
	}

	if is.EmptyStr(release.Bucket) {
		return fmt.Errorf("Bucket must be defined")
	}

	return nil
}

func fileOrJSON(fileOrJSON *string) (*string, error) {
	if fileOrJSON == nil {
		return nil, fmt.Errorf("No file or JSON")
	}

	raw := strings.Trim(*fileOrJSON, " \n")
	// Return if starts with open bracket
	if raw[0] == '{' {
		return fileOrJSON, nil
	}

	buf, err := ioutil.ReadFile(*fileOrJSON)
	if err != nil {
		return nil, err
	}

	return to.Strp(string(buf)), nil
}

func releaseFromFileOrJSON(releaseFileOrJSON *string, region *string, accountID *string) (*models.Release, error) {
	jsonRaw, err := fileOrJSON(releaseFileOrJSON)
	if err != nil {
		return nil, err
	}

	var release models.Release
	if err := json.Unmarshal([]byte(*jsonRaw), &release); err != nil {
		return nil, err
	}

	// replace user_data_file with .userdata
	if release.UserData != nil && *release.UserData == "{{USER_DATA_FILE}}" {
		buf, err := ioutil.ReadFile(fmt.Sprintf("%v.userdata", *releaseFileOrJSON))
		if err != nil {
			return nil, err
		}
		release.UserData = to.Strp(string(buf))
	}

	release.SetDefaultRegionAccount(region, accountID)

	if err := validateClientAttributes(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

func stateName(sd *execution.StateDetails) string {
	stateName := ""
	if sd.LastTaskName != nil {
		stateName = *sd.LastTaskName
	} else if sd.LastStateName != nil {
		stateName = *sd.LastStateName
	}
	return stateName
}

func waiter(ed *execution.ExecutionDetails, sd *execution.StateDetails, err error) error {
	if err != nil {
		return fmt.Errorf("Unexpected Error %v", err.Error())
	}

	spinnerCounter++

	ws, err := waiterStr(ed.Status, sd)
	if err != nil {
		return err
	}

	fmt.Printf("\r%v               ", ws)

	return nil
}

func waiterStr(status *string, sd *execution.StateDetails) (string, error) {
	newLine := fmt.Sprintf("%s(%s)", *status, stateName(sd))

	var release models.Release
	if sd.LastOutput != nil {
		if err := json.Unmarshal([]byte(*sd.LastOutput), &release); err != nil {
			return "", err
		}
	}
	// Checks it has correctly unmarshalled
	if release.ProjectName != nil {
		if release.Error != nil {
			newLine = fmt.Sprintf("%v Error %v(%v)", newLine, *release.Error.Error, *release.Error.Cause)
		} else {
			sh := []string{}
			for name, service := range release.Services {
				st := serviceStr(name, service)
				if st != "" {
					sh = append(sh, st)
				}
			}
			if len(sh) > 0 {
				sort.Strings(sh)
				newLine = fmt.Sprintf("%v %v", newLine, strings.Join(sh, "  "))
			}
		}
	}

	return fmt.Sprintf("%v%v", spinner(), newLine), nil
}

var spinnerCounter = 0
var spinnerChar = "/-\\|"

func spinner() string {
	return string(spinnerChar[int(math.Mod(float64(spinnerCounter), 4))])
}

func serviceStr(name string, service *models.Service) string {
	RED := "\x1b[0;31m"
	GRAY := "\x1b[1;37m"
	GREEN := "\x1b[0;32m"
	YELLOW := "\x1b[1;33m"
	NC := "\x1b[0m" // No Color

	if service.HealthReport != nil {
		dots := []string{}
		barAt := *service.HealthReport.TargetHealthy
		// There might have been a termination now number of instances are above desired capacity
		numberOfDots := int(math.Max(float64(*service.HealthReport.TargetLaunched), float64(*service.HealthReport.Launching)))

		numberOfGreenDots := *service.HealthReport.Healthy
		numberOfRedDots := *service.HealthReport.Terminating
		numberOfYellowDots := *service.HealthReport.Launching - numberOfGreenDots - numberOfRedDots

		for i := 0; i < numberOfDots; i++ {
			if i == barAt {
				dots = append(dots, fmt.Sprintf("%v|%v", GRAY, NC))
			}
			if i < numberOfGreenDots {
				dots = append(dots, fmt.Sprintf("%v.%v", GREEN, NC))
			} else if i < (numberOfGreenDots + numberOfYellowDots) {
				dots = append(dots, fmt.Sprintf("%v.%v", YELLOW, NC))
			} else if i < (numberOfGreenDots + numberOfYellowDots + numberOfRedDots) {
				dots = append(dots, fmt.Sprintf("%v.%v", RED, NC))
			} else {
				dots = append(dots, fmt.Sprintf("%v.%v", GRAY, NC))
			}
		}
		return fmt.Sprintf("%s: %v", name, strings.Join(dots, ""))
	}

	return ""
}
