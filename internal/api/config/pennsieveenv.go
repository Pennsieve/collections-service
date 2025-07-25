package config

import sharedconfig "github.com/pennsieve/collections-service/internal/shared/config"

const DiscoverServiceHostKey = "DISCOVER_SERVICE_HOST"
const PennsieveDOIPrefixKey = "PENNSIEVE_DOI_PREFIX"
const CollectionsIDSpaceIDKey = "COLLECTIONS_ID_SPACE_ID"
const CollectionsIDSpaceNameKey = "COLLECTIONS_ID_SPACE_NAME"
const PublishBucketKey = "PUBLISH_BUCKET"

const ServiceName = "collections-service"
const JWTSecretKeySSMName = "jwt-secret-key"

type PennsieveSettings struct {
	DiscoverServiceHost    sharedconfig.EnvironmentSetting
	DOIPrefix              sharedconfig.EnvironmentSetting
	CollectionsIDSpaceID   sharedconfig.EnvironmentSetting
	CollectionsIDSpaceName sharedconfig.EnvironmentSetting
	PublishBucket          sharedconfig.EnvironmentSetting
	JWTSecretKey           *sharedconfig.SSMSetting
}

var DeployedPennsieveSettings = PennsieveSettings{
	DiscoverServiceHost:    sharedconfig.NewEnvironmentSetting(DiscoverServiceHostKey),
	DOIPrefix:              sharedconfig.NewEnvironmentSetting(PennsieveDOIPrefixKey),
	CollectionsIDSpaceID:   sharedconfig.NewEnvironmentSetting(CollectionsIDSpaceIDKey),
	CollectionsIDSpaceName: sharedconfig.NewEnvironmentSetting(CollectionsIDSpaceNameKey),
	PublishBucket:          sharedconfig.NewEnvironmentSetting(PublishBucketKey),
	JWTSecretKey:           NewJWTSecretKeySetting(),
}

func NewJWTSecretKeySetting() *sharedconfig.SSMSetting {
	return sharedconfig.NewSSMSetting(ServiceName, JWTSecretKeySSMName)
}
