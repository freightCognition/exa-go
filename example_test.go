package exa_test

import (
	"context"
	"fmt"
	"log"

	"github.com/freightCognition/exa-go"
)

func ExampleClient_Search() {
	c, err := exa.New() // reads EXA_API_KEY
	if err != nil {
		log.Fatal(err)
	}
	resp, err := c.Search(context.Background(), &exa.SearchRequest{
		Query:      "freight brokerage software",
		NumResults: exa.Ptr(5),
		Contents:   &exa.ContentsOptions{Highlights: &exa.HighlightsOptions{}},
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, r := range resp.Results {
		fmt.Println(r.Title, r.URL)
	}
}
