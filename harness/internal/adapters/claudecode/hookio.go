// Package claudecode speaks the Claude Code hook protocol: a tool call as JSON
// on stdin, a permission decision as JSON on stdout.
package claudecode

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/Wihrt/mise/harness/internal/domain"
)

type input struct {
	ToolName  string `json:"tool_name"`
	Cwd       string `json:"cwd"`
	ToolInput struct {
		Command      string `json:"command"`
		Pattern      string `json:"pattern"`
		FilePath     string `json:"file_path"`
		NotebookPath string `json:"notebook_path"`
		SubagentType string `json:"subagent_type"`
		Model        string `json:"model"`
		Description  string `json:"description"`
	} `json:"tool_input"`
}

// Read decodes a PreToolUse hook input.
func Read(r io.Reader) (domain.Request, error) {
	var in input
	if err := json.NewDecoder(r).Decode(&in); err != nil {
		return domain.Request{}, fmt.Errorf("decode hook input: %w", err)
	}
	path := in.ToolInput.FilePath
	if path == "" {
		path = in.ToolInput.NotebookPath
	}
	return domain.Request{
		Tool:         in.ToolName,
		Command:      in.ToolInput.Command,
		Pattern:      in.ToolInput.Pattern,
		Path:         path,
		Cwd:          in.Cwd,
		SubagentType: in.ToolInput.SubagentType,
		Model:        in.ToolInput.Model,
		Description:  in.ToolInput.Description,
	}, nil
}

type output struct {
	HookSpecificOutput struct {
		HookEventName            string `json:"hookEventName"`
		PermissionDecision       string `json:"permissionDecision,omitempty"`
		PermissionDecisionReason string `json:"permissionDecisionReason,omitempty"`
		AdditionalContext        string `json:"additionalContext,omitempty"`
	} `json:"hookSpecificOutput"`
}

// Write encodes a decision; allowing a call writes nothing.
func Write(w io.Writer, d domain.Decision) error {
	if d.Verdict == domain.Allow {
		return nil
	}
	var out output
	out.HookSpecificOutput.HookEventName = "PreToolUse"
	if d.Verdict == domain.Hint {
		// No permission decision: the call follows the normal permission flow.
		out.HookSpecificOutput.AdditionalContext = d.Reason
		if err := json.NewEncoder(w).Encode(out); err != nil {
			return fmt.Errorf("encode hook hint: %w", err)
		}
		return nil
	}
	out.HookSpecificOutput.PermissionDecision = "deny"
	if d.Verdict == domain.Ask {
		out.HookSpecificOutput.PermissionDecision = "ask"
	}
	out.HookSpecificOutput.PermissionDecisionReason = d.Reason
	if err := json.NewEncoder(w).Encode(out); err != nil {
		return fmt.Errorf("encode hook decision: %w", err)
	}
	return nil
}
