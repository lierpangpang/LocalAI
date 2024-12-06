package model

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mudler/LocalAI/core/config"
	"github.com/mudler/LocalAI/core/schema"
	"github.com/mudler/LocalAI/pkg/functions"
	"github.com/mudler/LocalAI/pkg/templates"
	"github.com/rs/zerolog/log"
)

// Rather than pass an interface{} to the prompt template:
// These are the definitions of all possible variables LocalAI will currently populate for use in a prompt template file
// Please note: Not all of these are populated on every endpoint - your template should either be tested for each endpoint you map it to, or tolerant of zero values.
type PromptTemplateData struct {
	SystemPrompt         string
	SuppressSystemPrompt bool // used by chat specifically to indicate that SystemPrompt above should be _ignored_
	Input                string
	Instruction          string
	Functions            []functions.Function
	MessageIndex         int
}

type ChatMessageTemplateData struct {
	SystemPrompt string
	Role         string
	RoleName     string
	FunctionName string
	Content      string
	MessageIndex int
	Function     bool
	FunctionCall interface{}
	LastMessage  bool
}

const (
	ChatPromptTemplate templates.TemplateType = iota
	ChatMessageTemplate
	CompletionPromptTemplate
	EditPromptTemplate
	FunctionsPromptTemplate
)

func (ml *ModelLoader) EvaluateTemplateForPrompt(templateType templates.TemplateType, config config.BackendConfig, in PromptTemplateData) (string, error) {
	template := ""

	// A model can have a "file.bin.tmpl" file associated with a prompt template prefix
	if ml.ExistsInModelPath(fmt.Sprintf("%s.tmpl", config.Model)) {
		template = config.Model
	}

	switch templateType {
	case CompletionPromptTemplate:
		if config.TemplateConfig.Completion != "" {
			template = config.TemplateConfig.Completion
		}
	case EditPromptTemplate:
		if config.TemplateConfig.Edit != "" {
			template = config.TemplateConfig.Edit
		}
	case ChatPromptTemplate:
		if config.TemplateConfig.Chat != "" {
			template = config.TemplateConfig.Chat
		}
	case FunctionsPromptTemplate:
		if config.TemplateConfig.Functions != "" {
			template = config.TemplateConfig.Functions
		}
	}

	if template == "" {
		return in.Input, nil
	}

	if config.TemplateConfig.JinjaTemplate {
		return ml.EvaluateJinjaTemplateForPrompt(templateType, template, in)
	}

	return ml.templates.EvaluateTemplate(templateType, template, in)
}

func (ml *ModelLoader) EvaluateTemplateForChatMessage(templateName string, messageData ChatMessageTemplateData) (string, error) {
	return ml.templates.EvaluateTemplate(ChatMessageTemplate, templateName, messageData)
}

func (ml *ModelLoader) templateJinjaChat(templateName string, messageData []ChatMessageTemplateData) (string, error) {

	conversation := make(map[string]interface{})
	messages := make([]map[string]interface{}, len(messageData))

	for _, message := range messageData {
		// TODO: this is not correct, we have to map jinja tokenizer template from transformers to our own

		messages = append(messages, map[string]interface{}{
			"Role":         message.Role,
			"RoleName":     message.RoleName,
			"Content":      message.Content,
			"FunctionCall": message.FunctionCall,
			"FunctionName": message.FunctionName,
			"LastMessage":  message.LastMessage,
			"Function":     message.Function,
			"MessageIndex": message.MessageIndex,
		})
	}

	conversation["messages"] = messages

	return ml.templates.EvaluateJinjaTemplate(ChatMessageTemplate, templateName, conversation)
}

func (ml *ModelLoader) EvaluateJinjaTemplateForPrompt(templateType templates.TemplateType, templateName string, in PromptTemplateData) (string, error) {

	conversation := make(map[string]interface{})

	conversation["system_prompt"] = in.SystemPrompt
	conversation["content"] = in.Input

	return ml.templates.EvaluateJinjaTemplate(templateType, templateName, conversation)
}

func (ml *ModelLoader) TemplateMessages(messages []schema.Message, config *config.BackendConfig, funcs []functions.Function, shouldUseFn bool) string {

	if config.TemplateConfig.JinjaTemplate {
		var messageData []ChatMessageTemplateData
		for messageIndex, i := range messages {
			fcall := i.FunctionCall
			if len(i.ToolCalls) > 0 {
				fcall = i.ToolCalls
			}
			messageData = append(messageData, ChatMessageTemplateData{
				SystemPrompt: config.SystemPrompt,
				Role:         config.Roles[i.Role],
				RoleName:     i.Role,
				Content:      i.StringContent,
				FunctionCall: fcall,
				FunctionName: i.Name,
				LastMessage:  messageIndex == (len(messages) - 1),
				Function:     config.Grammar != "" && (messageIndex == (len(messages) - 1)),
				MessageIndex: messageIndex,
			})
		}

		templatedInput, err := ml.templateJinjaChat(config.TemplateConfig.ChatMessage, messageData)
		if err == nil {
			return templatedInput
		}
	}

	var predInput string
	suppressConfigSystemPrompt := false
	mess := []string{}
	for messageIndex, i := range messages {
		var content string
		role := i.Role

		// if function call, we might want to customize the role so we can display better that the "assistant called a json action"
		// if an "assistant_function_call" role is defined, we use it, otherwise we use the role that is passed by in the request
		if (i.FunctionCall != nil || i.ToolCalls != nil) && i.Role == "assistant" {
			roleFn := "assistant_function_call"
			r := config.Roles[roleFn]
			if r != "" {
				role = roleFn
			}
		}
		r := config.Roles[role]
		contentExists := i.Content != nil && i.StringContent != ""

		fcall := i.FunctionCall
		if len(i.ToolCalls) > 0 {
			fcall = i.ToolCalls
		}

		// First attempt to populate content via a chat message specific template
		if config.TemplateConfig.ChatMessage != "" {
			chatMessageData := ChatMessageTemplateData{
				SystemPrompt: config.SystemPrompt,
				Role:         r,
				RoleName:     role,
				Content:      i.StringContent,
				FunctionCall: fcall,
				FunctionName: i.Name,
				LastMessage:  messageIndex == (len(messages) - 1),
				Function:     config.Grammar != "" && (messageIndex == (len(messages) - 1)),
				MessageIndex: messageIndex,
			}
			templatedChatMessage, err := ml.EvaluateTemplateForChatMessage(config.TemplateConfig.ChatMessage, chatMessageData)
			if err != nil {
				log.Error().Err(err).Interface("message", chatMessageData).Str("template", config.TemplateConfig.ChatMessage).Msg("error processing message with template, skipping")
			} else {
				if templatedChatMessage == "" {
					log.Warn().Msgf("template \"%s\" produced blank output for %+v. Skipping!", config.TemplateConfig.ChatMessage, chatMessageData)
					continue // TODO: This continue is here intentionally to skip over the line `mess = append(mess, content)` below, and to prevent the sprintf
				}
				log.Debug().Msgf("templated message for chat: %s", templatedChatMessage)
				content = templatedChatMessage
			}
		}

		marshalAnyRole := func(f any) {
			j, err := json.Marshal(f)
			if err == nil {
				if contentExists {
					content += "\n" + fmt.Sprint(r, " ", string(j))
				} else {
					content = fmt.Sprint(r, " ", string(j))
				}
			}
		}
		marshalAny := func(f any) {
			j, err := json.Marshal(f)
			if err == nil {
				if contentExists {
					content += "\n" + string(j)
				} else {
					content = string(j)
				}
			}
		}
		// If this model doesn't have such a template, or if that template fails to return a value, template at the message level.
		if content == "" {
			if r != "" {
				if contentExists {
					content = fmt.Sprint(r, i.StringContent)
				}

				if i.FunctionCall != nil {
					marshalAnyRole(i.FunctionCall)
				}
				if i.ToolCalls != nil {
					marshalAnyRole(i.ToolCalls)
				}
			} else {
				if contentExists {
					content = fmt.Sprint(i.StringContent)
				}
				if i.FunctionCall != nil {
					marshalAny(i.FunctionCall)
				}
				if i.ToolCalls != nil {
					marshalAny(i.ToolCalls)
				}
			}
			// Special Handling: System. We care if it was printed at all, not the r branch, so check seperately
			if contentExists && role == "system" {
				suppressConfigSystemPrompt = true
			}
		}

		mess = append(mess, content)
	}

	joinCharacter := "\n"
	if config.TemplateConfig.JoinChatMessagesByCharacter != nil {
		joinCharacter = *config.TemplateConfig.JoinChatMessagesByCharacter
	}

	predInput = strings.Join(mess, joinCharacter)
	log.Debug().Msgf("Prompt (before templating): %s", predInput)

	promptTemplate := ChatPromptTemplate

	if config.TemplateConfig.Functions != "" && shouldUseFn {
		promptTemplate = FunctionsPromptTemplate
	}

	templatedInput, err := ml.EvaluateTemplateForPrompt(promptTemplate, *config, PromptTemplateData{
		SystemPrompt:         config.SystemPrompt,
		SuppressSystemPrompt: suppressConfigSystemPrompt,
		Input:                predInput,
		Functions:            funcs,
	})
	if err == nil {
		predInput = templatedInput
		log.Debug().Msgf("Template found, input modified to: %s", predInput)
	} else {
		log.Debug().Msgf("Template failed loading: %s", err.Error())
	}

	return predInput
}
