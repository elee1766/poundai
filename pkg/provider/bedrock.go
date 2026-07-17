package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"

	"github.com/elee1766/zsh_poundai/pkg/config"
)

// bedrock invokes models via Amazon Bedrock's Converse API, which works
// across model families. Credentials come from the standard AWS chain
// (env vars, shared config/credentials files, IMDS, SSO, ...).
type bedrock struct {
	svc    config.Service
	model  string
	client *bedrockruntime.Client
}

func newBedrock(svc config.Service) (*bedrock, error) {
	model := svc.Model
	if model == "" {
		return nil, fmt.Errorf("provider \"bedrock\": model is required (e.g. anthropic.claude-3-5-haiku-20241022-v1:0)")
	}
	var opts []func(*awsconfig.LoadOptions) error
	if svc.Region != "" {
		opts = append(opts, awsconfig.WithRegion(svc.Region))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}
	client := bedrockruntime.NewFromConfig(awsCfg)
	return &bedrock{svc: svc, model: model, client: client}, nil
}

func (p *bedrock) Complete(ctx context.Context, messages []Message) (string, error) {

	maxTokens := int32(p.svc.MaxTokens)
	if maxTokens == 0 {
		maxTokens = 1000
	}
	inferenceCfg := &types.InferenceConfiguration{MaxTokens: &maxTokens}
	if p.svc.Temperature != nil {
		t := float32(*p.svc.Temperature)
		inferenceCfg.Temperature = &t
	}

	system, rest := splitSystem(messages)
	var bedrockMsgs []types.Message
	for _, m := range rest {
		role := types.ConversationRoleUser
		if m.Role == "assistant" {
			role = types.ConversationRoleAssistant
		}
		bedrockMsgs = append(bedrockMsgs, types.Message{
			Role:    role,
			Content: []types.ContentBlock{&types.ContentBlockMemberText{Value: m.Content}},
		})
	}
	input := &bedrockruntime.ConverseInput{
		ModelId:         &p.model,
		Messages:        bedrockMsgs,
		InferenceConfig: inferenceCfg,
	}
	if system != "" {
		input.System = []types.SystemContentBlock{&types.SystemContentBlockMemberText{Value: system}}
	}
	out, err := p.client.Converse(ctx, input)
	if err != nil {
		return "", fmt.Errorf("bedrock converse: %w", err)
	}

	msg, ok := out.Output.(*types.ConverseOutputMemberMessage)
	if !ok {
		raw, _ := json.Marshal(out.Output)
		return "", fmt.Errorf("bedrock returned unexpected output type: %s", string(raw))
	}
	var sb strings.Builder
	for _, block := range msg.Value.Content {
		if text, ok := block.(*types.ContentBlockMemberText); ok {
			sb.WriteString(text.Value)
		}
	}
	if sb.Len() == 0 {
		return "", fmt.Errorf("bedrock returned no text content")
	}
	return sb.String(), nil
}
