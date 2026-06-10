package email

import "fmt"

type SimpleMessageBuilder struct {
	baseURL string
}

func NewSimpleMessageBuilder(baseURL string) *SimpleMessageBuilder {
	return &SimpleMessageBuilder{
		baseURL: baseURL,
	}
}

func (b *SimpleMessageBuilder) BuildConfirmationMessage(token string) (string, string) {
	subject := "Confirm your subscription"
	link := fmt.Sprintf("%s/api/v1/confirm/%s", b.baseURL, token)
	body := fmt.Sprintf("Please confirm your subscription by clicking here: %s", link)
	return subject, body
}

func (b *SimpleMessageBuilder) BuildReleaseMessage(repo, tag, token string) (string, string) {
	subject := fmt.Sprintf("New release for %s", repo)
	link := fmt.Sprintf("%s/api/v1/unsubscribe/%s", b.baseURL, token)
	body := fmt.Sprintf("A new release %s is available for %s!\r\n\r\nTo unsubscribe click here: %s", tag, repo, link)
	return subject, body
}
