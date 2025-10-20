package relayer

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"

	ckafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/henriquemarlon/city.fun/relayer/configs"
	"github.com/henriquemarlon/city.fun/relayer/configs/auth"
	"github.com/henriquemarlon/city.fun/relayer/internal/infra/repository"
	"github.com/henriquemarlon/city.fun/relayer/internal/usecase"
	"github.com/henriquemarlon/city.fun/relayer/pkg/contracts/rewardtoken"
	"github.com/henriquemarlon/city.fun/relayer/pkg/kafka"
	"github.com/henriquemarlon/city.fun/relayer/pkg/service"
	"github.com/henriquemarlon/city.fun/relayer/pkg/workerpool"
)

type RewardResult struct {
	MessageId string
	Success   bool
	Error     error
	Output    *usecase.CreateRewardOutputDTO
	KafkaMsg  *ckafka.Message
}

type Service struct {
	service.Service
	token         common.Address
	kafkaConsumer *kafka.KafkaConsumer
	repository    repository.Repository
	workerPool    workerpool.WorkerPool
	jobChan       chan workerpool.Job
	txChan        chan *usecase.CreateRewardOutputDTO
	sigintChan    chan struct{}
	wg            sync.WaitGroup
	ethClient     *ethclient.Client
	txOpts        *bind.TransactOpts
}

type CreateInfo struct {
	service.CreateInfo
	Config        configs.RelayerConfig
	KafkaConsumer *kafka.KafkaConsumer
	Repository    repository.Repository
	EthClient     *ethclient.Client
}

func Create(ctx context.Context, createInfo *CreateInfo) (*Service, error) {
	var err error
	if err = ctx.Err(); err != nil {
		return nil, err
	}

	s := &Service{}
	createInfo.Impl = s

	err = service.Create(ctx, &createInfo.CreateInfo, &s.Service)
	if err != nil {
		return nil, err
	}

	s.repository = createInfo.Repository
	if s.repository == nil {
		return nil, fmt.Errorf("repository on relayer service create is nil")
	}

	s.kafkaConsumer = createInfo.KafkaConsumer
	if s.kafkaConsumer == nil {
		return nil, fmt.Errorf("kafka consumer on relayer service create is nil")
	}

	s.ethClient = createInfo.EthClient
	if s.ethClient == nil {
		return nil, fmt.Errorf("eth client on relayer service create is nil")
	}

	chainId, err := s.ethClient.ChainID(ctx)
	if err != nil {
		return nil, err
	}
	if chainId.Uint64() != createInfo.Config.BlockchainId {
		return nil, fmt.Errorf("chainId mismatch: network %d != provided %d", chainId.Uint64(), createInfo.Config.BlockchainId)
	}

	s.txOpts, err = auth.GetTransactOpts(chainId)
	if err != nil {
		return nil, err
	}

	s.token = createInfo.Config.RewardToken
	if s.token == (common.Address{}) {
		return nil, fmt.Errorf("token address on relayer service create is nil")
	}

	s.jobChan = make(chan workerpool.Job, 100)
	s.txChan = make(chan *usecase.CreateRewardOutputDTO, 100)
	s.sigintChan = make(chan struct{})

	processFunc := func(ctx context.Context, job workerpool.Job) workerpool.Result {
		msg := job.(*ckafka.Message)

		result := RewardResult{
			MessageId: string(msg.Key),
			Success:   false,
			KafkaMsg:  msg,
		}

		var input usecase.CreateRewardInputDTO
		if err := json.Unmarshal(msg.Value, &input); err != nil {
			result.Error = fmt.Errorf("failed to unmarshal message: %w", err)
			s.Logger.Error("Failed to unmarshal message",
				"error", err,
				"message", string(msg.Value),
				"partition", msg.TopicPartition.Partition,
				"offset", msg.TopicPartition.Offset)
			return result
		}

		input.Token = s.token

		createRewardUseCase := usecase.NewCreateRewardUseCase(s.repository)
		output, err := createRewardUseCase.Execute(ctx, &input)
		if err != nil {
			result.Error = fmt.Errorf("failed to create reward: %w", err)
			s.Logger.Error("Failed to save reward to DB",
				"error", err,
				"receiver", input.Receiver,
				"amount", input.Amount)
			return result
		}

		s.Logger.Info("Reward processed in DB",
			"id", output.Id.Hex(),
			"receiver", output.Receiver,
			"amount", output.Amount,
			"latitude", output.Latitude,
			"longitude", output.Longitude)

		select {
		case s.txChan <- output:
			s.Logger.Debug("Transaction queued for blockchain", "id", output.Id.Hex())
		case <-ctx.Done():
			s.Logger.Warn("Context cancelled, transaction not queued", "id", output.Id.Hex())
		}

		result.Success = true
		result.Output = output

		return result
	}

	config := workerpool.Config{
		WorkerCount: 5,
		Logger:      s.Logger,
	}
	s.workerPool = workerpool.New(processFunc, config)

	s.Logger.Info("Relayer service created",
		"workers", config.WorkerCount,
		"job_buffer", cap(s.jobChan))

	return s, nil
}

func (s *Service) Alive() bool {
	if s.workerPool == nil {
		return false
	}
	return s.workerPool.IsRunning()
}

func (s *Service) Ready() bool {
	if s.workerPool == nil {
		return false
	}
	return s.workerPool.IsRunning()
}

func (s *Service) Reload() []error {
	return nil
}

func (s *Service) Tick() []error {
	return nil
}

func (s *Service) Serve() error {
	resultChan, err := s.workerPool.Start(s.Context, s.jobChan)
	if err != nil {
		return fmt.Errorf("failed to start worker pool: %w", err)
	}

	s.wg.Add(1)
	go s.processBlockchainTransactions()

	s.wg.Add(1)
	go s.processWorkerResults(resultChan)

	s.wg.Add(1)
	go s.consumeKafkaMessages()

	return s.Service.Serve()
}

func (s *Service) processWorkerResults(resultChan <-chan workerpool.Result) {
	defer s.wg.Done()

	for {
		select {
		case result, ok := <-resultChan:
			if !ok {
				s.Logger.Info("Worker result channel closed")
				return
			}

			rewardResult, ok := result.(RewardResult)
			if !ok {
				s.Logger.Error("Invalid result type from worker")
				continue
			}

			if rewardResult.Success && rewardResult.KafkaMsg != nil {
				if err := s.kafkaConsumer.CommitMessage(rewardResult.KafkaMsg); err != nil {
					s.Logger.Error("Failed to commit Kafka offset",
						"error", err,
						"partition", rewardResult.KafkaMsg.TopicPartition.Partition,
						"offset", rewardResult.KafkaMsg.TopicPartition.Offset)
				} else {
					s.Logger.Debug("Kafka offset committed after successful DB save",
						"id", rewardResult.Output.Id.Hex(),
						"partition", rewardResult.KafkaMsg.TopicPartition.Partition,
						"offset", rewardResult.KafkaMsg.TopicPartition.Offset)
				}
			} else {
				s.Logger.Warn("Worker processing failed, not committing offset",
					"error", rewardResult.Error)
			}

		case <-s.sigintChan:
			s.Logger.Info("Worker result processor stopping")
			return
		case <-s.Context.Done():
			s.Logger.Info("Worker result processor cancelled")
			return
		}
	}
}

func (s *Service) consumeKafkaMessages() {
	defer s.wg.Done()

	inputChan := make(chan *ckafka.Message, 10)
	errChan := make(chan error, 1)

	go func() {
		if err := s.kafkaConsumer.Consume(inputChan); err != nil {
			s.Logger.Error("Kafka consumer fatal error", "error", err)
			errChan <- err
		}
	}()

	for {
		select {
		case msg := <-inputChan:
			select {
			case s.jobChan <- msg:
				s.Logger.Debug("Job enqueued",
					"topic", *msg.TopicPartition.Topic,
					"partition", msg.TopicPartition.Partition,
					"offset", msg.TopicPartition.Offset)
			case <-s.sigintChan:
				return
			case <-s.Context.Done():
				return
			}
		case err := <-errChan:
			s.Logger.Error("Kafka consumer failed, requesting shutdown", "error", err)
			select { case <-s.sigintChan: default: close(s.sigintChan) }
			return
		case <-s.sigintChan:
			return
		case <-s.Context.Done():
			return
		}
	}
}

func (s *Service) processBlockchainTransactions() {
	defer s.wg.Done()
	for {
		select {
		case reward := <-s.txChan:
			s.Logger.Debug("Processing blockchain transaction",
				"id", reward.Id.Hex(),
				"receiver", reward.Receiver,
				"amount", reward.Amount)

			txHash, err := s.mintReward(reward)
			if err != nil {
				s.Logger.Error("Failed to mint reward on blockchain",
					"error", err,
					"id", reward.Id.Hex(),
					"receiver", reward.Receiver,
					"amount", reward.Amount)
				continue
			}

			updateTxHashUseCase := usecase.NewUpdateRewardTxHashUseCase(s.repository)
			err = updateTxHashUseCase.Execute(s.Context, reward.Id, txHash.Hex())
			if err != nil {
				s.Logger.Error("Failed to update reward tx hash in DB",
					"error", err,
					"id", reward.Id.Hex(),
					"tx_hash", txHash.Hex())
			}

			s.Logger.Info("Reward minted on blockchain",
				"id", reward.Id.Hex(),
				"token", s.token,
				"amount", reward.Amount,
				"receiver", reward.Receiver,
				"tx_hash", txHash,
				"data", reward.Data)

		case <-s.sigintChan:
			s.Logger.Info("Blockchain transaction processor stopping")
			return
		case <-s.Context.Done():
			s.Logger.Info("Blockchain transaction processor cancelled")
			return
		}
	}
}

func (s *Service) mintReward(reward *usecase.CreateRewardOutputDTO) (common.Hash, error) {
	tokenAddr := s.token
	receiverAddr := common.HexToAddress(reward.Receiver)

	contract, err := rewardtoken.NewRewardToken(tokenAddr, s.ethClient)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to create contract: %w", err)
	}

	amount := new(big.Int)
	if _, ok := amount.SetString(reward.Amount, 10); !ok {
		return common.Hash{}, fmt.Errorf("invalid amount format: %s", reward.Amount)
	}

	txOpts := *s.txOpts // clone
	nonce, err := s.ethClient.PendingNonceAt(s.Context, txOpts.From)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to get nonce: %w", err)
	}
	txOpts.Nonce = big.NewInt(int64(nonce))

	tx, err := contract.Mint(&txOpts, receiverAddr, amount)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to mint: %w", err)
	}

	return tx.Hash(), nil
}

func (s *Service) Stop(force bool) []error {
	var errs []error

	if s.sigintChan != nil {
		close(s.sigintChan)
	}

	if s.jobChan != nil {
		close(s.jobChan)
	}

	if s.txChan != nil {
		close(s.txChan)
	}

	if err := s.workerPool.Stop(); err != nil {
		errs = append(errs, fmt.Errorf("failed to stop worker pool: %w", err))
	}

	s.wg.Wait()

	if s.repository != nil {
		if err := s.repository.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close repository: %w", err))
		}
	}
	return errs
}

func (s *Service) String() string {
	return s.Name
}
