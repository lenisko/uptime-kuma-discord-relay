package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"github.com/pelletier/go-toml/v2"
	"net/http"
	"os"
	"strconv"
)

type webhookPayload struct {
	Heartbeat struct {
		MonitorID int    `json:"monitorID"`
		Status    int    `json:"status"`
		Time      string `json:"time"`
		Msg       string `json:"msg"`
		Important bool   `json:"important"`
		Duration  int    `json:"duration"`
	} `json:"heartbeat"`
	Monitor struct {
		ID                     int               `json:"id"`
		Name                   string            `json:"name"`
		Description            string            `json:"description"`
		URL                    string            `json:"url"`
		Method                 string            `json:"method"`
		Hostname               string            `json:"hostname"`
		Port                   string            `json:"port"`
		MaxRetries             int               `json:"maxretries"`
		Weight                 int               `json:"weight"`
		Active                 bool              `json:"active"`
		Type                   string            `json:"type"`
		Interval               int               `json:"interval"`
		RetryInterval          int               `json:"retryInterval"`
		ResendInterval         int               `json:"resendInterval"`
		Keyword                string            `json:"keyword"`
		ExpiryNotification     bool              `json:"expiryNotification"`
		IgnoreTls              bool              `json:"ignoreTls"`
		UpsideDown             bool              `json:"upsideDown"`
		PacketSize             int               `json:"packetSize"`
		MaxRedirects           int               `json:"maxredirects"`
		AcceptedStatusCodes    []string          `json:"accepted_statuscodes"`
		DnsResolveType         string            `json:"dns_resolve_type"`
		DnsResolveServer       string            `json:"dns_resolve_server"`
		DnsLastResult          string            `json:"dns_last_result"`
		DockerContainer        string            `json:"docker_container"`
		DockerHost             string            `json:"docker_host"`
		ProxyID                string            `json:"proxyId"`
		NotificationIDList     map[string]bool   `json:"notificationIDList"`
		Tags                   []string          `json:"tags"`
		Maintenance            bool              `json:"maintenance"`
		MqttTopic              string            `json:"mqttTopic"`
		MqttSuccessMessage     string            `json:"mqttSuccessMessage"`
		DatabaseQuery          string            `json:"databaseQuery"`
		AuthMethod             map[string]string `json:"authMethod"`
		GrpcUrl                string            `json:"grpcUrl"`
		GrpcProtobuf           string            `json:"grpcProtobuf"`
		GrpcMethod             string            `json:"grpcMethod"`
		GrpcServiceName        string            `json:"grpcServiceName"`
		GrpcEnableTls          bool              `json:"grpcEnableTls"`
		RadiusCalledStationId  string            `json:"radiusCalledStationId"`
		RadiusCallingStationId string            `json:"radiusCallingStationId"`
		Game                   string            `json:"game"`
	} `json:"monitor"`
	Msg string `json:"msg"`
}

type DiscordStatus struct {
	color  int
	status string
}

type Config struct {
	WebhookURL  string `toml:"webhook_url"`
	BearerToken string `toml:"bearer_token"`
	UptimeURL   string `toml:"uptime_url"`
	Prod        bool   `toml:"prod"`
	HttpPort    int    `toml:"http_port"`
}

func LoadConfig() (*Config, error) {
	config := &Config{}

	data, err := os.ReadFile("config.toml")
	if err != nil {
		return nil, err
	}

	err = toml.Unmarshal(data, &config)

	return config, nil
}

func main() {
	config, err := LoadConfig()
	if err != nil {
		panic("Can't load config.yml file.")
	}

	if config.Prod {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.Default()
	router.POST("/webhook", func(c *gin.Context) {
		// check auth
		authHeader := c.GetHeader("Authorization")
		if authHeader != "Bearer "+config.BearerToken {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		// Parse the uptime-kuma webhook payload
		var payload webhookPayload
		if err = c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Veryify payload monitor name before processing
		if payload.Monitor.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing monitor name"})
			return
		}

		discordStatus := &DiscordStatus{
			color:  65280,
			status: "Up",
		}

		if payload.Heartbeat.Status == 0 {
			discordStatus = &DiscordStatus{
				color:  16711680,
				status: "Down",
			}
		}

		client := resty.New()
		_, err = client.R().
			SetHeader("Content-Type", "application/json").
			SetBody(map[string]interface{}{
				"content": "",
				"embeds": []map[string]interface{}{
					{
						"type":        "rich",
						"title":       fmt.Sprintf("%s is %s", payload.Monitor.Name, discordStatus.status),
						"description": payload.Monitor.Description,
						"color":       discordStatus.color,
						"url":         config.UptimeURL,
					},
				},
			}).
			Post(config.WebhookURL)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Success"})
	})

	err = router.Run(":" + strconv.Itoa(config.HttpPort))
	if err != nil {
		panic("Can't start HTTP server")
	}
}
