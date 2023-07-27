package spec

import "go.keploy.io/server/pkg/models"

type PostgresSpec struct {
	PostgresReq models.PostgresReq `json:"postgresReq" yaml:"postgresReq"`
	PostgresResp models.PostgresResp `json:"postgresResp" yaml:"postgresResp"`
}
 