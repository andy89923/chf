package consumer

import (
	"context"
	"fmt"
	"sync"

	"github.com/free5gc/openapi/models"
	"github.com/free5gc/openapi/nwdaf/AnalyticsInfo"
)

type nnwdafService struct {
	consumer *Consumer

	analyticsInfoMu sync.RWMutex

	analyticsInfoClients map[string]*AnalyticsInfo.APIClient
}

func (s *nnwdafService) getAnalyticsInfoClient(uri string) *AnalyticsInfo.APIClient {
	if uri == "" {
		return nil
	}
	s.analyticsInfoMu.RLock()
	client, ok := s.analyticsInfoClients[uri]
	if ok {
		s.analyticsInfoMu.RUnlock()
		return client
	}

	configuration := AnalyticsInfo.NewConfiguration()
	configuration.SetBasePath(uri)
	client = AnalyticsInfo.NewAPIClient(configuration)

	s.analyticsInfoMu.RUnlock()
	s.analyticsInfoMu.Lock()
	defer s.analyticsInfoMu.Unlock()
	s.analyticsInfoClients[uri] = client
	return client
}

func (s *nnwdafService) GetAnalytics(
	eventFilter *models.NwdafAnalyticsInfoEventFilter,
) (*models.NwdafAnalyticsInfoAnalyticsData, error) {
	nwdafUri := ""
	client := s.getAnalyticsInfoClient(nwdafUri)
	if client == nil {
		return nil, fmt.Errorf("NWDAF AnalyticsInfo client is nil")
	}

	req := &AnalyticsInfo.GetNWDAFAnalyticsRequest{
		EventFilter: eventFilter,
	}
	resp, err := client.NWDAFAnalyticsDocumentApi.GetNWDAFAnalytics(context.Background(), req)
	if err != nil {
		return nil, err
	}
	return &resp.NwdafAnalyticsInfoAnalyticsData, nil
}
