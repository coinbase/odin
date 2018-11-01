package cost

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/coinbase/step/utils/is"

	"github.com/aws/aws-sdk-go/service/pricing"
	"github.com/coinbase/odin/aws"
	"github.com/coinbase/step/utils/to"
)

var REGIONS = map[string]string{
	"us-east-2":      "US East (Ohio)",
	"us-east-1":      "US East (N. Virginia)",
	"us-west-1":      "US West (N. California)",
	"us-west-2":      "US West (Oregon)",
	"ap-south-1":     "Asia Pacific (Mumbai)",
	"ap-northeast-3": "Asia Pacific (Osaka-Local)",
	"ap-northeast-2": "Asia Pacific (Seoul)",
	"ap-southeast-1": "Asia Pacific (Singapore)",
	"ap-southeast-2": "Asia Pacific (Sydney)",
	"ap-northeast-1": "Asia Pacific (Tokyo)",
	"ca-central-1":   "Canada (Central)",
	"cn-north-1":     "China (Beijing)",
	"cn-northwest-1": "China (Ningxia)",
	"eu-central-1":   "EU (Frankfurt)",
	"eu-west-1":      "EU (Ireland)",
	"eu-west-2":      "EU (London)",
	"eu-west-3":      "EU (Paris)",
	"sa-east-1":      "South America (São Paulo)",
}

type PriceDimension struct {
	PricePerUnit struct {
		USD *string `json:"USD"`
	} `json:"pricePerUnit"`
}

type Term struct {
	PriceDimensions map[string]PriceDimension `json:"priceDimensions"`
}

type Terms struct {
	OnDemand map[string]Term `json:"OnDemand"`
}

// Cost returns the cost of an instance
// This is a major PITA because the pricing API is very difficult to navigate
func Cost(pricec aws.PricingAPI, region *string, instanceType *string) (*string, error) {
	input := &pricing.GetProductsInput{
		ServiceCode: to.Strp("AmazonEC2"),
		Filters: []*pricing.Filter{
			&pricing.Filter{
				Field: to.Strp("operatingSystem"),
				Type:  to.Strp("TERM_MATCH"),
				Value: to.Strp("Linux"),
			},
			&pricing.Filter{
				Field: to.Strp("operation"),
				Type:  to.Strp("TERM_MATCH"),
				Value: to.Strp("RunInstances"),
			},
			&pricing.Filter{
				Field: to.Strp("capacitystatus"),
				Type:  to.Strp("TERM_MATCH"),
				Value: to.Strp("Used"),
			},
			&pricing.Filter{
				Field: to.Strp("tenancy"),
				Type:  to.Strp("TERM_MATCH"),
				Value: to.Strp("Shared"),
			},
			&pricing.Filter{
				Field: to.Strp("instanceType"),
				Type:  to.Strp("TERM_MATCH"),
				Value: instanceType,
			},
			&pricing.Filter{
				Field: to.Strp("location"),
				Type:  to.Strp("TERM_MATCH"),
				Value: to.Strp(REGIONS[*region]),
			},
		},
	}

	p, err := pricec.GetProducts(input)
	if err != nil {
		return nil, err
	}

	// expect only one price, otherwise could over/under price
	if len(p.PriceList) != 1 {
		return nil, fmt.Errorf("Princing Error: Expected len(PriceList) to eq 1 but was %v", len(p.PriceList))
	}

	termInput := p.PriceList[0]["terms"]

	if termInput == nil {
		return nil, fmt.Errorf("Princing Error: No terms")
	}

	termJSON, err := json.Marshal(termInput)
	if err != nil {
		return nil, err
	}

	var terms Terms
	json.Unmarshal([]byte(termJSON), &terms)

	if len(terms.OnDemand) != 1 {
		return nil, fmt.Errorf("Princing Error: Expected len(OnDemand) to eq 1 but was %v", len(terms.OnDemand))
	}

	var term Term
	for _, v := range terms.OnDemand {
		term = v
	}

	if len(term.PriceDimensions) != 1 {
		return nil, fmt.Errorf("Princing Error: Expected len(PriceDimension) to eq 1 but was %v", len(term.PriceDimensions))
	}

	var pd PriceDimension
	for _, v := range term.PriceDimensions {
		pd = v
	}

	if is.EmptyStr(pd.PricePerUnit.USD) {
		return nil, fmt.Errorf("Princing Error: PricePerUnit.USD not found")
	}

	return pd.PricePerUnit.USD, nil
}

// SmartBidPrice returns a smart bid price for a Spot instance
func SmartBidPrice(pricec aws.PricingAPI, region *string, instanceType *string) (*string, error) {
	price, err := Cost(pricec, region, instanceType)
	if err != nil {
		return nil, err
	}

	pricef, err := strconv.ParseFloat(*price, 64)
	if err != nil {
		return nil, fmt.Errorf("Princing Error: Parsing Price %v", price)
	}

	// To ensure we keep the instances we increase bid by %1
	pricef *= 1.01

	return to.Strp(fmt.Sprintf("%f", pricef)), nil
}
