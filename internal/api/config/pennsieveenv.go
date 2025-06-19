package config

import sharedconfig "github.com/pennsieve/collections-service/internal/shared/config"

const DiscoverServiceHostKey = "DISCOVER_SERVICE_HOST"
const PennsieveDOIPrefixKey = "PENNSIEVE_DOI_PREFIX"
const CollectionNamespaceIDKey = "COLLECTION_NAMESPACE_ID"
const PublishBucketKey = "PUBLISH_BUCKET"

const ServiceName = "collections-service"
const JWTSecretKeySSMName = "jwt-secret-key"

type PennsieveEnvironmentSettings struct {
	DiscoverServiceHost   sharedconfig.EnvironmentSetting
	DOIPrefix             sharedconfig.EnvironmentSetting
	CollectionNamespaceID sharedconfig.EnvironmentSetting
	PublishBucket         sharedconfig.EnvironmentSetting
}

var DeployedPennsieveEnvironmentSettings = PennsieveEnvironmentSettings{
	DiscoverServiceHost:   sharedconfig.NewEnvironmentSetting(DiscoverServiceHostKey),
	DOIPrefix:             sharedconfig.NewEnvironmentSetting(PennsieveDOIPrefixKey),
	CollectionNamespaceID: sharedconfig.NewEnvironmentSetting(CollectionNamespaceIDKey),
	PublishBucket:         sharedconfig.NewEnvironmentSetting(PublishBucketKey),
}

var JWTSecretKeySetting = sharedconfig.NewSSMSetting(ServiceName, JWTSecretKeySSMName)
