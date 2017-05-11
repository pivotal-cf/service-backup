package upload

type CACertLocator func() (string, error)

type opts struct {
	factory       UploaderFactory
	caCertLocator CACertLocator
}

type Option func(*opts)

func WithUploaderFactory(f UploaderFactory) Option {
	return func(o *opts) {
		o.factory = f
	}
}

func WithCACertLocator(l CACertLocator) Option {
	return func(o *opts) {
		o.caCertLocator = l
	}
}
