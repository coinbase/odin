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
									"CGJXHFUSGE546RV6.JRTCKXETXF": {
											"priceDimensions": {
													"CGJXHFUSGE546RV6.JRTCKXETXF.6YS6EN2CT7": {
															"pricePerUnit": {
																	"USD": "0.100000"
															}
													}
											}
									}
							},
							"Reserved": {
									"CGJXHFUSGE546RV6.38NPMPTW36": {
											"priceDimensions": {
													"CGJXHFUSGE546RV6.38NPMPTW36.6YS6EN2CT7": {
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

// CWClient struct
type PricingClient struct {
	aws.PricingAPI
}

// GetProducts returns
func (m *PricingClient) GetProducts(input *pricing.GetProductsInput) (*pricing.GetProductsOutput, error) {
	var pdo pricing.GetProductsOutput
	json.Unmarshal([]byte(mockGetProductResponse), &pdo)
	return &pdo, nil
}
