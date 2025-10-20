package root

import (
	"context"
	"fmt"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/henriquemarlon/city.fun/simulator/configs"
	"github.com/henriquemarlon/city.fun/simulator/internal/infra/repository/factory"
	"github.com/henriquemarlon/city.fun/simulator/internal/infra/service/simulation"
	"github.com/henriquemarlon/city.fun/simulator/internal/infra/version"
	"github.com/henriquemarlon/city.fun/simulator/pkg/events"
	"github.com/henriquemarlon/city.fun/simulator/pkg/service"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const serviceName = "simulator"

var (
	logColor            bool
	logLevel            string
	sensorServerAddress string
	telemetryAddress    string
	pushInterval        int
	hivemqUrl           string
	hivemqMqttTopic     string
	hivemqUsername      string
	hivemqUsernameFile  string
	hivemqPassword      string
	hivemqPasswordFile  string
	databaseUrl         string
	databaseUrlFile     string
	databaseName        string
	databaseCollection  string
	cfg                 *configs.SimulatorConfig
)

var Cmd = &cobra.Command{
	Use:     "congo-" + serviceName,
	Short:   "Runs congo-" + serviceName,
	Long:    "Runs congo-" + serviceName + " in standalone mode",
	Run:     run,
	Version: version.BuildVersion,
}

func init() {
	Cmd.Flags().BoolVar(&logColor, "log-color", true, "Tint the logs (colored output)")
	cobra.CheckErr(viper.BindPFlag(configs.LOG_COLOR, Cmd.Flags().Lookup("log-color")))
	Cmd.Flags().StringVar(&logLevel, "log-level", "info", "Log level: debug, info, warn or error")
	cobra.CheckErr(viper.BindPFlag(configs.LOG_LEVEL, Cmd.Flags().Lookup("log-level")))

	Cmd.Flags().StringVar(&sensorServerAddress, "sensor-server-address", ":10000", "Sensor server address and port")
	cobra.CheckErr(viper.BindPFlag(configs.SENSOR_SERVER_ADDRESS, Cmd.Flags().Lookup("sensor-server-address")))
	Cmd.Flags().StringVar(&telemetryAddress, "telemetry-address", ":10001", "Health check and metrics address and port")
	cobra.CheckErr(viper.BindPFlag(configs.TELEMETRY_ADDRESS, Cmd.Flags().Lookup("telemetry-address")))
	Cmd.Flags().IntVar(&pushInterval, "push-interval", 1, "Push interval in seconds")
	cobra.CheckErr(viper.BindPFlag(configs.PUSH_INTERVAL, Cmd.Flags().Lookup("push-interval")))

	Cmd.Flags().StringVar(&hivemqUrl, "hivemq-url", "", "HiveMQ URL")
	cobra.CheckErr(viper.BindPFlag(configs.HIVEMQ_URL, Cmd.Flags().Lookup("hivemq-url")))

	Cmd.Flags().StringVar(&hivemqUsername, "hivemq-username", "", "HiveMQ Username")
	cobra.CheckErr(viper.BindPFlag(configs.HIVEMQ_USERNAME, Cmd.Flags().Lookup("hivemq-username")))
	Cmd.Flags().StringVar(&hivemqUsernameFile, "hivemq-username-file", "", "Path to file containing HiveMQ Username")
	cobra.CheckErr(viper.BindPFlag(configs.HIVEMQ_USERNAME_FILE, Cmd.Flags().Lookup("hivemq-username-file")))

	Cmd.Flags().StringVar(&hivemqPassword, "hivemq-password", "", "HiveMQ Password")
	cobra.CheckErr(viper.BindPFlag(configs.HIVEMQ_PASSWORD, Cmd.Flags().Lookup("hivemq-password")))
	Cmd.Flags().StringVar(&hivemqPasswordFile, "hivemq-password-file", "", "Path to file containing HiveMQ Password")
	cobra.CheckErr(viper.BindPFlag(configs.HIVEMQ_PASSWORD_FILE, Cmd.Flags().Lookup("hivemq-password-file")))

	Cmd.Flags().StringVar(&hivemqMqttTopic, "hivemq-mqtt-topic", "sensors/data", "MQTT topic for publishing sensor data")
	cobra.CheckErr(viper.BindPFlag(configs.HIVEMQ_MQTT_TOPIC, Cmd.Flags().Lookup("hivemq-mqtt-topic")))

	Cmd.Flags().StringVar(&databaseUrl, "database-url", "", "Database URL")
	cobra.CheckErr(viper.BindPFlag(configs.DATABASE_URL, Cmd.Flags().Lookup("database-url")))
	Cmd.Flags().StringVar(&databaseUrlFile, "database-url-file", "", "Path to file containing Database URL")
	cobra.CheckErr(viper.BindPFlag(configs.DATABASE_URL_FILE, Cmd.Flags().Lookup("database-url-file")))

	Cmd.Flags().StringVar(&databaseName, "database-name", "", "Database Name")
	cobra.CheckErr(viper.BindPFlag(configs.DATABASE_NAME, Cmd.Flags().Lookup("database-name")))
	Cmd.Flags().StringVar(&databaseCollection, "database-collection", "", "Database Collection")
	cobra.CheckErr(viper.BindPFlag(configs.DATABASE_COLLECTION, Cmd.Flags().Lookup("database-collection")))

	Cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = configs.LoadSimulatorConfig()
		if err != nil {
			return err
		}
		return nil
	}
}

func run(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.MaxStartupTime)
	defer cancel()

	createInfo := simulation.CreateInfo{
		CreateInfo: service.CreateInfo{
			Name:                 serviceName,
			LogLevel:             cfg.LogLevel,
			LogColor:             cfg.LogColor,
			EnableSignalHandling: true,
			TelemetryCreate:      true,
			TelemetryAddress:     cfg.TelemetryAddress,
		},
		Config: *cfg,
	}

	var err error
	createInfo.Repository, err = factory.NewRepositoryFromConnectionString(
		ctx,
		cfg.DatabaseUrl.String(),
		cfg.DatabaseName,
		cfg.DatabaseCollection,
	)
	cobra.CheckErr(err)

	defer createInfo.Repository.Close()

	createInfo.EventDispatcher = events.NewEventDispatcher()

	mqttClientOptions := MQTT.NewClientOptions()
	mqttClientOptions.AddBroker(cfg.HivemqUrl)
	mqttClientOptions.SetUsername(cfg.HivemqUsername)
	mqttClientOptions.SetPassword(cfg.HivemqPassword.Value)
	mqttClientOptions.SetClientID("simulator")
	mqttClientOptions.SetAutoReconnect(false)
	mqttClientOptions.SetConnectTimeout(5 * time.Second)
	mqttClientOptions.SetOrderMatters(false)

	createInfo.MqttClient = MQTT.NewClient(mqttClientOptions)
	if token := createInfo.MqttClient.Connect(); token.Wait() && token.Error() != nil {
		cobra.CheckErr(fmt.Errorf("failed to connect to MQTT broker: %w", token.Error()))
	}

	if !createInfo.MqttClient.IsConnected() {
		cobra.CheckErr(fmt.Errorf("failed to establish MQTT connection"))
	}

	simulationService, err := simulation.Create(ctx, &createInfo)
	cobra.CheckErr(err)

	cobra.CheckErr(simulationService.Serve())
}
