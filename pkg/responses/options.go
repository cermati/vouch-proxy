package responses

// Option is an option type for responses package
type Option func(idx *Index)

// WithPrevURLOption sets previous URL to be used in response template
func WithPrevURLOption(prevURL string) Option {
	return func(idx *Index) {
		idx.PrevURL = prevURL
	}
}
