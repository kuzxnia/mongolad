package driver

import (
	"github.com/kuzxnia/mongoload/pkg/config"
	"github.com/kuzxnia/mongoload/pkg/database"
	"github.com/kuzxnia/mongoload/pkg/schema"
)

type JobHandler interface {
	Handle() (bool, error)
}

func NewJobHandler(cfg *config.Job, client database.Client) JobHandler {
	handler := BaseHandler{
		client: client,
    provider: schema.NewDataProvider(cfg.GetTemplateSchema()),
	}
	switch cfg.Type {
	case string(config.Write):
		return JobHandler(&WriteHandler{BaseHandler: &handler})
	case string(config.Read):
		return JobHandler(&ReadHandler{BaseHandler: &handler})
	case string(config.Update):
		return JobHandler(&UpdateHandler{BaseHandler: &handler})
	case string(config.BulkWrite):
		return JobHandler(&BulkWriteHandler{BaseHandler: &handler})
	default:
		// todo change
    panic("Invalid job type: " + cfg.Type)
	}
}

type BaseHandler struct {
	client   database.Client
	provider schema.DataProvider
}

type WriteHandler struct {
	*BaseHandler
}

func (h *WriteHandler) Handle() (bool, error) {
	return h.client.InsertOne(h.provider.GetSingleItem())
}

type BulkWriteHandler struct {
	*BaseHandler
}

func (h *BulkWriteHandler) Handle() (bool, error) {
	return h.client.InsertMany(h.provider.GetBatch(100))
}

type ReadHandler struct {
	*BaseHandler
}

func (h *ReadHandler) Handle() (bool, error) {
	return h.client.ReadOne(h.provider.GetFilter())
}

type UpdateHandler struct {
	*BaseHandler
}

func (h *UpdateHandler) Handle() (bool, error) {
	return h.client.UpdateOne(h.provider.GetFilter(), h.provider.GetSingleItem())
}
