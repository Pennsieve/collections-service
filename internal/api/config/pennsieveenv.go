package config

import sharedconfig "github.com/pennsieve/collections-service/internal/shared/config"

const DiscoverServiceHostKey = "DISCOVER_SERVICE_HOST"
const PennsieveDOIPrefixKey = "PENNSIEVE_DOI_PREFIX"

type PennsieveEnvironmentSettings struct {
	DiscoverServiceHost sharedconfig.EnvironmentSetting
	DOIPrefix           sharedconfig.EnvironmentSetting
}

var DeployedPennsieveEnvironmentSettings = PennsieveEnvironmentSettings{
	DiscoverServiceHost: sharedconfig.NewEnvironmentSetting(DiscoverServiceHostKey),
	DOIPrefix:           sharedconfig.NewEnvironmentSetting(PennsieveDOIPrefixKey),
}
