package rabbitmq

type Config struct {
	URL      string
	Exchange string
	Queue    string
	DLQ      string
}
