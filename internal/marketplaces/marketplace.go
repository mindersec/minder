package marketplaces

import "context"

type Marketplace interface {
	GetLastBundleVersion(ctx context.Context, namespace string, bundle string)
	ListBundles(ctx context.Context)
}

type MarketplaceService interface {
}
