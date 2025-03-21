package generic

import (
	"io"

	// nolint
	"github.com/golang/protobuf/jsonpb"
	// nolint
	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/dynamic"
)

// Codec jsonpb marshal & unmarshal
type Codec interface {
	// Marshal message into json writer
	Marshal(w io.Writer, m proto.Message) error
	// Unmarshal read r and unmarshal into message
	Unmarshal(r io.Reader, m proto.Message) error
}

// jsonpbCodec codec between json and messages
type jsonpbCodec struct {
	resolver jsonpb.AnyResolver
}

// anyResolver returns a jsonpb.AnyResolver that uses the given file descriptors
// to resolve message names. It uses the given factory, which may be nil, to
// instantiate messages. The messages that it returns when resolving a type name
// may often be dynamic messages.
func anyResolver(source Descriptor) (jsonpb.AnyResolver, error) {
	files, err := GetAllFiles(source)
	if err != nil {
		return nil, err
	}

	var er dynamic.ExtensionRegistry
	for _, fd := range files {
		er.AddExtensionsFromFile(fd)
	}
	mf := dynamic.NewMessageFactoryWithExtensionRegistry(&er)

	// use descriptor as message factory, to construct any message
	return dynamic.AnyResolver(mf, files...), nil
}

// newJsonpbCodec returns a RequestParser that reads data in JSON format
// from the given reader.
func newJsonpbCodec(descSource Descriptor) (Codec, error) {
	resolver, err := anyResolver(descSource)
	if err != nil {
		return nil, err
	}

	return &jsonpbCodec{
		resolver: resolver,
	}, nil
}

// Unmarshal json into message
func (j *jsonpbCodec) Unmarshal(r io.Reader, m proto.Message) error {
	unmarshaler := jsonpb.Unmarshaler{
		AnyResolver:        j.resolver,
		AllowUnknownFields: true,
	}

	return unmarshaler.Unmarshal(r, m)
}

// Marshal message into json
func (j *jsonpbCodec) Marshal(w io.Writer, m proto.Message) error {
	marshaler := jsonpb.Marshaler{
		AnyResolver:  j.resolver,
		EmitDefaults: true,
	}

	return marshaler.Marshal(w, m)
}
