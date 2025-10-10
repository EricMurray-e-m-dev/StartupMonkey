package adapter

import (
	"database/sql"
)

type PostGresAdapter struct {
	connectionString string
	conn             *sql.DB
}

func (p *PostGresAdapter) Connect() (_ error) {
	panic("not implemented") // TODO: Implement
}

func (p *PostGresAdapter) CollectMetrics() (_ *RawMetrics, _ error) {
	panic("not implemented") // TODO: Implement
}

func (p *PostGresAdapter) Close() (_ error) {
	panic("not implemented") // TODO: Implement
}

func (p *PostGresAdapter) HealthCheck() (_ error) {
	panic("not implemented") // TODO: Implement
}
