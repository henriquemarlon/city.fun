package root

import (
	"context"

	ckafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/hashicorp/go-retryablehttp"

	"github.com/henriquemarlon/city.fun/relayer/configs"
	"github.com/henriquemarlon/city.fun/relayer/internal/infra/repository/factory"
	"github.com/henriquemarlon/city.fun/relayer/internal/infra/service/relayer"
	"github.com/henriquemarlon/city.fun/relayer/internal/infra/version"
	"github.com/henriquemarlon/city.fun/relayer/pkg/kafka"
	"github.com/henriquemarlon/city.fun/relayer/pkg/service"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const serviceName = "relayer"

var (
	logColor               bool
	logLevel               string
	authKind               string
	authPrivateKey         string
	authPrivateKeyFile     string
	authMnemonic           string
	authMnemonicFile       string
	authMnemonicIndex      uint32
	kafkaBroker            string
	kafkaTopics            string
	databaseUrl            string
	databaseUrlFile        string
	databaseName           string
	databaseCollection     string
	maxStartupTime         int
	telemetryAddress       string
	blockchainId           uint64
	blockchainHttpEndpoint string
	rewardToken            string
	cfg                    *configs.RelayerConfig
)

var Cmd = &cobra.Command{
	Use:     "city-fun-" + serviceName,
	Short:   "Runs City.fun Relayer",
	Long:    `A Kafka-based relayer service for processing and distributing rewards on city.fun platform`,
	Run:     run,
	Version: version.BuildVersion,
}

func init() {
	// Logging flags
	Cmd.Flags().BoolVar(&logColor, "log-color", true, "Tint the logs (colored output)")
	cobra.CheckErr(viper.BindPFlag(configs.LOG_COLOR, Cmd.Flags().Lookup("log-color")))
	Cmd.Flags().StringVar(&logLevel, "log-level", "info", "Log level: debug, info, warn or error")
	cobra.CheckErr(viper.BindPFlag(configs.LOG_LEVEL, Cmd.Flags().Lookup("log-level")))

	// Blockchain flags
	Cmd.Flags().Uint64Var(&blockchainId, "blockchain-id", 31337, "Blockchain ID")
	cobra.CheckErr(viper.BindPFlag(configs.BLOCKCHAIN_ID, Cmd.Flags().Lookup("blockchain-id")))
	Cmd.Flags().StringVar(&blockchainHttpEndpoint, "blockchain-http-endpoint", "", "Blockchain HTTP endpoint")
	cobra.CheckErr(viper.BindPFlag(configs.BLOCKCHAIN_HTTP_ENDPOINT, Cmd.Flags().Lookup("blockchain-http-endpoint")))

	// Auth flags
	Cmd.Flags().StringVar(&authKind, "auth-kind", "private_key", "Auth kind: private_key, private_key_file, mnemonic, mnemonic_file")
	cobra.CheckErr(viper.BindPFlag(configs.AUTH_KIND, Cmd.Flags().Lookup("auth-kind")))
	Cmd.Flags().StringVar(&authPrivateKey, "auth-private-key", "", "Private key for signing transactions")
	cobra.CheckErr(viper.BindPFlag(configs.AUTH_PRIVATE_KEY, Cmd.Flags().Lookup("auth-private-key")))
	Cmd.Flags().StringVar(&authPrivateKeyFile, "auth-private-key-file", "", "Path to file containing private key")
	cobra.CheckErr(viper.BindPFlag(configs.AUTH_PRIVATE_KEY_FILE, Cmd.Flags().Lookup("auth-private-key-file")))
	Cmd.Flags().StringVar(&authMnemonic, "auth-mnemonic", "", "Mnemonic for signing transactions")
	cobra.CheckErr(viper.BindPFlag(configs.AUTH_MNEMONIC, Cmd.Flags().Lookup("auth-mnemonic")))
	Cmd.Flags().StringVar(&authMnemonicFile, "auth-mnemonic-file", "", "Path to file containing mnemonic")
	cobra.CheckErr(viper.BindPFlag(configs.AUTH_MNEMONIC_FILE, Cmd.Flags().Lookup("auth-mnemonic-file")))
	Cmd.Flags().Uint32Var(&authMnemonicIndex, "auth-mnemonic-index", 0, "Account index for mnemonic-based auth")
	cobra.CheckErr(viper.BindPFlag(configs.AUTH_MNEMONIC_ACCOUNT_INDEX, Cmd.Flags().Lookup("auth-mnemonic-index")))

	// Kafka flags
	Cmd.Flags().StringVar(&kafkaBroker, "kafka-broker", "localhost:9092", "Kafka broker URL")
	cobra.CheckErr(viper.BindPFlag(configs.KAFKA_BROKER, Cmd.Flags().Lookup("kafka-broker")))
	Cmd.Flags().StringVar(&kafkaTopics, "kafka-topics", "reward_granted", "Comma-separated list of Kafka topics to consume")
	cobra.CheckErr(viper.BindPFlag(configs.KAFKA_TOPICS, Cmd.Flags().Lookup("kafka-topics")))

	// Database flags
	Cmd.Flags().StringVar(&databaseUrl, "database-url", "", "MongoDB connection URL")
	cobra.CheckErr(viper.BindPFlag(configs.DATABASE_URL, Cmd.Flags().Lookup("database-url")))
	Cmd.Flags().StringVar(&databaseUrlFile, "database-url-file", "", "Path to file containing MongoDB URL")
	cobra.CheckErr(viper.BindPFlag(configs.DATABASE_URL_FILE, Cmd.Flags().Lookup("database-url-file")))
	Cmd.Flags().StringVar(&databaseName, "database-name", "", "MongoDB database name")
	cobra.CheckErr(viper.BindPFlag(configs.DATABASE_NAME, Cmd.Flags().Lookup("database-name")))
	Cmd.Flags().StringVar(&databaseCollection, "database-collection", "", "MongoDB collection name")
	cobra.CheckErr(viper.BindPFlag(configs.DATABASE_COLLECTION, Cmd.Flags().Lookup("database-collection")))

	// Service flags
	Cmd.Flags().IntVar(&maxStartupTime, "max-startup-time", 15, "Maximum startup time in seconds")
	cobra.CheckErr(viper.BindPFlag(configs.MAX_STARTUP_TIME, Cmd.Flags().Lookup("max-startup-time")))
	Cmd.Flags().StringVar(&telemetryAddress, "telemetry-address", ":10001", "Telemetry address")
	cobra.CheckErr(viper.BindPFlag(configs.TELEMETRY_ADDRESS, Cmd.Flags().Lookup("telemetry-address")))

	// Contracts flags
	Cmd.Flags().StringVar(&rewardToken, "reward-token-address", "", "Reward token address")
	cobra.CheckErr(viper.BindPFlag(configs.REWARD_TOKEN_ADDRESS, Cmd.Flags().Lookup("reward-token-address")))

	Cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = configs.LoadRelayerConfig()
		if err != nil {
			return err
		}
		return nil
	}
}

func run(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	createInfo := relayer.CreateInfo{
		CreateInfo: service.CreateInfo{
			Name:                 serviceName,
			LogLevel:             cfg.LogLevel,
			LogColor:             cfg.LogColor,
			EnableSignalHandling: true,
			TelemetryCreate:      true,
			TelemetryAddress:     cfg.TelemetryAddress,
			Context:              ctx,
		},
		Config: *cfg,
	}

	rclient := retryablehttp.NewClient()
	rclient.Logger = service.NewLogger(cfg.LogLevel, cfg.LogColor).With("service", serviceName)
	rclient.RetryMax = int(cfg.BlockchainHttpMaxRetries)
	rclient.RetryWaitMin = cfg.BlockchainHttpRetryMinWait
	rclient.RetryWaitMax = cfg.BlockchainHttpRetryMaxWait

	clientOptions := []rpc.ClientOption{
		rpc.WithHTTPClient(rclient.StandardClient()),
	}

	var err error
	rpcClient, err := rpc.DialOptions(ctx, cfg.BlockchainHttpEndpoint.String(), clientOptions...)
	cobra.CheckErr(err)
	createInfo.EthClient = ethclient.NewClient(rpcClient)

	createInfo.Repository, err = factory.NewRepositoryFromConnectionString(
		ctx,
		cfg.DatabaseUrl.String(),
		cfg.DatabaseName.Value,
		cfg.DatabaseCollection.Value,
	)
	cobra.CheckErr(err)

	defer createInfo.Repository.Close()

	configMap := &ckafka.ConfigMap{
		"bootstrap.servers":  cfg.KafkaBroker.String(),
		"group.id":           "city-fun-" + serviceName,
		"auto.offset.reset":  "latest",
		"enable.auto.commit": false,
	}

	createInfo.KafkaConsumer = kafka.NewKafkaConsumer(configMap, cfg.KafkaTopics)

	relayer, err := relayer.Create(ctx, &createInfo)
	cobra.CheckErr(err)

	cobra.CheckErr(relayer.Serve())
}
