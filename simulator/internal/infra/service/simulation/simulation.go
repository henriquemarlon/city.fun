package simulation

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/rs/cors"

	"github.com/henriquemarlon/city.fun/simulator/configs"
	"github.com/henriquemarlon/city.fun/simulator/pkg/service"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/henriquemarlon/city.fun/simulator/internal/domain/entity"
	"github.com/henriquemarlon/city.fun/simulator/internal/domain/event"
	event_handler "github.com/henriquemarlon/city.fun/simulator/internal/domain/event/handler"
	"github.com/henriquemarlon/city.fun/simulator/internal/infra/repository"
	"github.com/henriquemarlon/city.fun/simulator/internal/infra/service/simulation/handler"
	"github.com/henriquemarlon/city.fun/simulator/internal/usecase"
	"github.com/henriquemarlon/city.fun/simulator/pkg/events"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Service struct {
	service.Service
	mqttTopic       string
	mqttClient      MQTT.Client
	sensorServer    *http.Server
	stopWorkerPool  chan struct{}
	wg              sync.WaitGroup
	sensorChannel   chan *entity.Sensor
	repository      repository.Repository
	eventDispatcher events.EventDispatcherInterface
	pushInterval    time.Duration
}

type CreateInfo struct {
	service.CreateInfo
	MqttClient      MQTT.Client
	Repository      repository.Repository
	Config          configs.SimulatorConfig
	EventDispatcher events.EventDispatcherInterface
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
		return nil, fmt.Errorf("repository on simulation service create is nil")
	}

	s.mqttClient = createInfo.MqttClient
	if s.mqttClient == nil {
		return nil, fmt.Errorf("mqtt client on simulation service create is nil")
	}

	s.eventDispatcher = createInfo.EventDispatcher
	if s.eventDispatcher == nil {
		return nil, fmt.Errorf("event dispatcher on simulation service create is nil")
	}

	s.sensorChannel = make(chan *entity.Sensor)
	s.stopWorkerPool = make(chan struct{})
	s.pushInterval = createInfo.Config.PushInterval
	s.mqttTopic = createInfo.Config.HivemqMqttTopic

	go s.runWorkerPool()

	findAllSensors := usecase.NewFindAllSensorsUseCase(s.repository)
	sensors, err := findAllSensors.Execute(ctx)
	if err != nil {
		return nil, err
	}

	s.Logger.Info("Sensors loaded from database", "count", len(sensors))

	for _, sensor := range sensors {
		s.Logger.Debug("Enqueuing sensor", "id", sensor.Id.Hex(), "name", sensor.Name)
		s.sensorChannel <- &entity.Sensor{
			Id:        sensor.Id,
			Name:      sensor.Name,
			Latitude:  sensor.Latitude,
			Longitude: sensor.Longitude,
			Receiver:  sensor.Receiver,
			Amount:    sensor.Amount,
			Params:    sensor.Params,
		}
		s.Logger.Debug("Sensor enqueued", "id", sensor.Id.Hex())
	}

	sensorCreatedEvent := event.NewSensorCreated()
	h := handler.NewSensorHandlers(sensorCreatedEvent, s.repository, s.eventDispatcher, s.sensorChannel)

	mux := http.NewServeMux()
	mux.HandleFunc("/sensor", h.CreateSensor)
	s.sensorServer = &http.Server{
		Addr:    createInfo.Config.SensorServerAddress,
		Handler: cors.Default().Handler(mux),
	}

	return s, nil
}

func (s *Service) Alive() bool     { return s.mqttClient != nil && s.mqttClient.IsConnected() }
func (s *Service) Ready() bool     { return s.mqttClient != nil && s.mqttClient.IsConnected() }
func (s *Service) Reload() []error { return nil }
func (s *Service) Tick() []error {
	return nil
}

func (s *Service) Serve() error {
	go func() {
		s.Logger.Info("Listening", "addr", s.sensorServer.Addr)
		if err := s.sensorServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.Logger.Error("SensorServer error", "error", err)
		}
	}()
	return s.Service.Serve()
}

func (s *Service) String() string {
	return s.Name
}

func (s *Service) Stop(force bool) []error {
	var errs []error

	if s.stopWorkerPool != nil {
		close(s.stopWorkerPool)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.sensorServer.Shutdown(ctx); err != nil {
		errs = append(errs, err)
	}

	if s.mqttClient != nil && s.mqttClient.IsConnected() {
		s.mqttClient.Disconnect(250)
	}

	return errs
}

func (s *Service) runWorkerPool() {
	activeSensors := make(map[string]context.CancelFunc)
	var mu sync.RWMutex

	for {
		select {
		case sensor := <-s.sensorChannel:
			s.Logger.Debug("Received sensor from channel", "id", sensor.Id.Hex(), "name", sensor.Name)

			mu.RLock()
			_, exists := activeSensors[sensor.Id.Hex()]
			mu.RUnlock()

			if exists {
				s.Logger.Debug("Sensor worker already running", "id", sensor.Id.Hex())
				continue
			}

			s.wg.Add(1)
			go func(id string) {
				defer s.wg.Done()

				workerCtx, cancel := context.WithCancel(s.Context)

				mu.Lock()
				activeSensors[id] = cancel
				mu.Unlock()

				defer func() {
					mu.Lock()
					delete(activeSensors, id)
					mu.Unlock()
				}()

				objectID, err := primitive.ObjectIDFromHex(id)
				if err != nil {
					s.Logger.Error("Failed to parse sensor ID", "id", id, "error", err)
					return
				}

				findSensorById := usecase.NewFindSensorByIdUseCase(s.repository)
				sensorOutput, err := findSensorById.Execute(workerCtx, &usecase.FindSensorByIdInputDTO{
					Id: objectID,
				})
				if err != nil {
					s.Logger.Error("Failed to find sensor", "id", id, "error", err)
					return
				}

				sensor := &entity.Sensor{
					Id:        sensorOutput.Id,
					Name:      sensorOutput.Name,
					Latitude:  sensorOutput.Latitude,
					Longitude: sensorOutput.Longitude,
					Params:    sensorOutput.Params,
				}

				dataEmittedEvent := event.NewDataEmitted(sensor.Id.Hex())
				dataEmittedHandler := event_handler.NewDataEmittedHandler(s.mqttClient, s.mqttTopic)
				if err := s.eventDispatcher.Register(dataEmittedEvent.GetName(), dataEmittedHandler); err != nil {
					s.Logger.Error("Failed to register event handler", "id", sensor.Id.Hex(), "error", err)
					return
				}

				s.Logger.Info("Starting sensor worker", "id", sensor.Id.Hex(), "name", sensor.Name)

				emitData := usecase.NewEmitDataUseCase(dataEmittedEvent, s.repository, s.eventDispatcher)

				ticker := time.NewTicker(s.pushInterval)
				defer ticker.Stop()

				for {
					select {
					case <-workerCtx.Done():
						s.Logger.Info("Stopping sensor worker", "id", sensor.Id.Hex())
						return

					case <-ticker.C:
						res, err := emitData.Execute(workerCtx, &usecase.EmitDataInputDTO{
							Id: sensor.Id,
						})
						if err != nil {
							s.Logger.Error("Failed to emit data", "id", sensor.Id.Hex(), "error", err)
							continue
						}

						s.Logger.Info(
							"Data emitted",
							"id", sensor.Id.Hex(),
							"name", sensor.Name,
							"latitude", sensor.Latitude,
							"longitude", sensor.Longitude,
							"data", string(res.Data),
						)
					}
				}
			}(sensor.Id.Hex())

		case <-s.stopWorkerPool:
			mu.Lock()
			for _, cancel := range activeSensors {
				cancel()
			}
			mu.Unlock()

			s.wg.Wait()
			return
		}
	}
}
