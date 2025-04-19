package healing_test

import (
	"github.com/cirglo.com/dfs/pkg/healing"
	"github.com/cirglo.com/dfs/pkg/mocks"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNewHealingService(t *testing.T) {
	logger := logrus.New()
	fileService := mocks.NewFileService(t)
	connectionFactory := mocks.NewConnectionFactory(t)
	opts := healing.Opts{
		Logger:            logger,
		NumReplicas:       1,
		FileService:       fileService,
		NodeExpiration:    24 * time.Hour,
		ConnectionFactory: connectionFactory,
	}
	service, err := healing.NewService(opts)
	assert.NoError(t, err)
	assert.NotNil(t, service)
}
