package name_test

import (
	"github.com/cirglo.com/dfs/pkg/mocks"
	"github.com/cirglo.com/dfs/pkg/name"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNewHealingService(t *testing.T) {
	logger := logrus.New()
	fileService := mocks.NewFileService(t)
	connectionFactory := mocks.NewConnectionFactory(t)
	opts := name.HealingOpts{
		Logger:            logger,
		NumReplicas:       1,
		FileService:       fileService,
		NodeExpiration:    24 * time.Hour,
		ConnectionFactory: connectionFactory,
	}
	service, error := name.NewHealingService(opts)
	assert.NoError(t, error)
	assert.NotNil(t, service)
}
