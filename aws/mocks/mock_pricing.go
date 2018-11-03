package mocks

import (
	"encoding/json"

	"github.com/aws/aws-sdk-go/service/pricing"
	"github.com/coinbase/odin/aws"
)

var mockGetProductResponse = `
{
  "FormatVersion": "aws_v1",
  "NextToken": null,
  "PriceList": [
      {
          "product": {
              "attributes": {
                  "instanceType": "r4.large"
              }
          },
          "terms": {
              "OnDemand": {
                  "THISISARANDOMSTRING": {
                      "priceDimensions": {
                          "THISISARANDOMSTRING": {
                              "pricePerUnit": {
                                  "USD": "0.100000"
                              }
                          }
                      }
                  }
              },
              "Reserved": {
                  "THISISARANDOMSTRING": {
                      "priceDimensions": {
                          "THISISARANDOMSTRING": {
                              "pricePerUnit": {
                                  "USD": "0.0270000000"
                              }
                          }
                      }
                  }
              }
          },
          "version": "20181031070014"
      }
  ]
}
 `

// PricingClient struct
type PricingClient struct {
	aws.PricingAPI
}

// GetProducts returns
func (m *PricingClient) GetProducts(input *pricing.GetProductsInput) (*pricing.GetProductsOutput, error) {
	var pdo pricing.GetProductsOutput
	json.Unmarshal([]byte(mockGetProductResponse), &pdo)
	return &pdo, nil
}
