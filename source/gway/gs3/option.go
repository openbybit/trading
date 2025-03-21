package gs3

type option struct {
	bucket string
	path   string
}

// Options s3 session option
type Options func(*option)

// WithBucket set s3 bucket
func WithBucket(bucket string) Options {
	return func(o *option) {
		o.bucket = bucket
	}
}

// WithPath set s3 path
func WithPath(path string) Options {
	return func(o *option) {
		o.path = path
	}
}
