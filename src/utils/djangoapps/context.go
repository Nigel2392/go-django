package djangoapps

import "context"

type appContextKey struct{}

func AppFromContext(ctx context.Context) (BaseAppConfig, bool) {
	app, ok := ctx.Value(appContextKey{}).(BaseAppConfig)
	return app, ok
}

func ContextWithApp(ctx context.Context, app BaseAppConfig) context.Context {
	return context.WithValue(ctx, appContextKey{}, app)
}
