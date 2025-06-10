package config

import sharedconfig "github.com/pennsieve/collections-service/internal/shared/config"

const DiscoverServiceHostKey = "DISCOVER_SERVICE_HOST"
const PennsieveDOIPrefixKey = "PENNSIEVE_DOI_PREFIX"
const CollectionNamespaceIDKey = "COLLECTION_NAMESPACE_ID"

const ServiceName = "collections-service"
const JWTSecretKeySSMName = "jwt-secret-key"

type PennsieveEnvironmentSettings struct {
	DiscoverServiceHost   sharedconfig.EnvironmentSetting
	DOIPrefix             sharedconfig.EnvironmentSetting
	CollectionNamespaceID sharedconfig.EnvironmentSetting
}

var DeployedPennsieveEnvironmentSettings = PennsieveEnvironmentSettings{
	DiscoverServiceHost:   sharedconfig.NewEnvironmentSetting(DiscoverServiceHostKey),
	DOIPrefix:             sharedconfig.NewEnvironmentSetting(PennsieveDOIPrefixKey),
	CollectionNamespaceID: sharedconfig.NewEnvironmentSetting(CollectionNamespaceIDKey),
}

var JWTSecretKeySetting = sharedconfig.NewSSMSetting(ServiceName, JWTSecretKeySSMName)
