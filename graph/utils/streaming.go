package utils

import (
	"context"
	"strings"

	"github.com/tmc/langchaingo/llms"
)

// StreamingOptions configures streaming behavior
type StreamingOptions struct {
	NodeID   string
	Callback func(nodeId string, chunk string)
	Logger   *Logger
}

// StreamingHelper provides common streaming functionality
type StreamingHelper struct {
	llm llms.Model
}

// NewStreamingHelper creates a new streaming helper
func NewStreamingHelper(llm llms.Model) *StreamingHelper {
	return &StreamingHelper{llm: llm}
}

// GenerateWithStreaming generates content with streaming support
func (h *StreamingHelper) GenerateWithStreaming(
	ctx context.Context,
	prompt string,
	opts StreamingOptions,
) (string, error) {
	var response strings.Builder
	
	// Log the operation start
	if opts.Logger != nil {
		opts.Logger.Debug("Starting streaming generation for node: %s", opts.NodeID)
	}
	
	_, err := llms.GenerateFromSinglePrompt(ctx, h.llm, prompt, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
		chunkStr := string(chunk)
		response.Write(chunk)
		
		// Call the streaming callback if provided
		if opts.Callback != nil {
			opts.Callback(opts.NodeID, chunkStr)
		}
		
		// Debug log chunks
		if opts.Logger != nil && len(chunkStr) > 0 {
			opts.Logger.Debug("Streaming chunk for %s: %q (length: %d)", opts.NodeID, chunkStr, len(chunkStr))
		}
		
		return nil
	}))
	
	if err != nil {
		if opts.Logger != nil {
			opts.Logger.Error("Streaming generation failed for %s: %v", opts.NodeID, err)
		}
		return "", err
	}
	
	result := response.String()
	if opts.Logger != nil {
		opts.Logger.Info("Streaming generation completed for %s: %d characters", opts.NodeID, len(result))
	}
	
	return result, nil
}

// GenerateWithoutStreaming generates content without streaming (for JSON responses)
func (h *StreamingHelper) GenerateWithoutStreaming(
	ctx context.Context,
	prompt string,
	opts StreamingOptions,
) (string, error) {
	if opts.Logger != nil {
		opts.Logger.Debug("Starting non-streaming generation for node: %s", opts.NodeID)
	}
	
	response, err := llms.GenerateFromSinglePrompt(ctx, h.llm, prompt)
	if err != nil {
		if opts.Logger != nil {
			opts.Logger.Error("Non-streaming generation failed for %s: %v", opts.NodeID, err)
		}
		return "", err
	}
	
	if opts.Logger != nil {
		opts.Logger.Info("Non-streaming generation completed for %s: %d characters", opts.NodeID, len(response))
	}
	
	return response, nil
}