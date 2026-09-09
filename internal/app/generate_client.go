package app

import (
	"context"

	"protocollens/internal/generator"
	"protocollens/internal/generator/curl"
	"protocollens/internal/generator/golang"
	"protocollens/internal/generator/python"
)

type GenerateClient struct {
	getTemplate *GetReplayTemplate
}

func NewGenerateClient(getTemplate *GetReplayTemplate) *GenerateClient {
	return &GenerateClient{getTemplate: getTemplate}
}

func (uc *GenerateClient) Execute(ctx context.Context, analysisID, requestID string, target generator.Target) (generator.Output, error) {
	template, err := uc.getTemplate.Execute(ctx, analysisID, requestID)
	if err != nil {
		return generator.Output{}, err
	}

	in := generator.Input{
		Method:      template.Method,
		URL:         template.URL,
		Query:       template.Query,
		Headers:     template.Headers,
		Body:        template.Body,
		BodyStruct:  template.BodyStruct,
		ContentType: template.ContentType,
	}

	switch target {
	case generator.TargetCurl:
		return curl.Generate(in)
	case generator.TargetPython:
		return python.Generate(in)
	case generator.TargetGo:
		return golang.Generate(in)
	default:
		return generator.Output{}, generator.ErrUnsupportedTarget
	}
}
