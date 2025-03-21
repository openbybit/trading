package generic

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"sort"
	"strings"
	"sync"

	// nolint
	"github.com/golang/protobuf/proto"
	descpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/desc/protoprint"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/grpcreflect"
)

var (
	_ sourceWithFiles = (*fileSource)(nil)

	printer = &protoprint.Printer{
		Compact:                  true,
		OmitComments:             protoprint.CommentsNonDoc,
		SortElements:             true,
		ForceFullyQualifiedNames: true,
	}
)

// Descriptor is a source of protobuf descriptor information. It can be backed by a FileDescriptorSet
// proto (like a file generated by protoc) or a remote server that supports the reflection API.
type Descriptor interface {
	// ListServices returns a list of fully-qualified service names. It will be all services in a set of
	// descriptor files or the set of all services exposed by a gRPC server.
	ListServices() ([]string, error)
	// FindSymbol returns a descriptor for the given fully-qualified symbol name.
	FindSymbol(fullyQualifiedName string) (desc.Descriptor, error)
	// AllExtensionsForType returns all known extension fields that extend the given message type name.
	AllExtensionsForType(typeName string) ([]*desc.FieldDescriptor, error)
}

type sourceWithFiles interface {
	GetAllFiles() ([]*desc.FileDescriptor, error)
}

// DescriptorFromProtoSetReader load from descriptor data
func DescriptorFromProtoSetReader(readers ...io.Reader) (Descriptor, error) {
	files := &descpb.FileDescriptorSet{}
	for _, reader := range readers {
		if reader == nil {
			continue
		}

		var fs descpb.FileDescriptorSet

		data, err := ioutil.ReadAll(reader)
		if err != nil {
			return nil, err
		}

		if err := proto.Unmarshal(data, &fs); err != nil {
			return nil, fmt.Errorf("could not parse contents of protoset %w", err)
		}

		files.File = append(files.File, fs.File...)
	}
	return DescriptorFromFileDescriptorSet(files)
}

// DescriptorFromProtoSetFiles creates a DescriptorSource that is backed by the named files, whose contents
// are encoded FileDescriptorSet protos.
func DescriptorFromProtoSetFiles(fileNames ...string) (Descriptor, error) {
	files := &descpb.FileDescriptorSet{}
	for _, fileName := range fileNames {
		b, err := ioutil.ReadFile(fileName)
		if err != nil {
			return nil, fmt.Errorf("could not load protoset file %q: %w", fileName, err)
		}

		var fs descpb.FileDescriptorSet
		err = proto.Unmarshal(b, &fs)
		if err != nil {
			return nil, fmt.Errorf("could not parse contents of protoset file %q: %w", fileName, err)
		}

		files.File = append(files.File, fs.File...)
	}
	return DescriptorFromFileDescriptorSet(files)
}

// DescriptorFromProtoFiles creates a DescriptorSource that is backed by the named files,
// whose contents are Protocol Buffer source files. The given importPaths are used to locate
// any imported files.
func DescriptorFromProtoFiles(importPaths []string, fileNames ...string) (Descriptor, error) {
	p := protoparse.Parser{
		ImportPaths:      importPaths,
		InferImportPaths: len(importPaths) == 0,
	}
	// parse proto source into descriptor
	fds, err := p.ParseFiles(fileNames...)
	if err != nil {
		return nil, fmt.Errorf("could not parse given files: %w", err)
	}

	return DescriptorFromFileDescriptors(fds...)
}

// DescriptorFromFileDescriptorSet creates a DescriptorSource that is backed by the FileDescriptorSet.
func DescriptorFromFileDescriptorSet(files *descpb.FileDescriptorSet) (Descriptor, error) {
	unresolved := map[string]*descpb.FileDescriptorProto{}
	for _, fd := range files.File {
		unresolved[fd.GetName()] = fd
	}

	resolved := map[string]*desc.FileDescriptor{}
	for _, fd := range files.File {
		_, err := resolveFileDescriptor(unresolved, resolved, fd.GetName())
		if err != nil {
			return nil, err
		}
	}

	return &fileSource{files: resolved}, nil
}

// resolveFileDescriptor resolve protoset files into descriptor in recursive
func resolveFileDescriptor(unresolved map[string]*descpb.FileDescriptorProto, resolved map[string]*desc.FileDescriptor, filename string) (*desc.FileDescriptor, error) {
	if r, ok := resolved[filename]; ok {
		return r, nil
	}

	fd, ok := unresolved[filename]
	if !ok {
		return nil, fmt.Errorf("no descriptor found for %q", filename)
	}

	deps := make([]*desc.FileDescriptor, 0, len(fd.GetDependency()))
	for _, dep := range fd.GetDependency() {
		// resolve dependency
		depFd, err := resolveFileDescriptor(unresolved, resolved, dep)
		if err != nil {
			return nil, err
		}
		deps = append(deps, depFd)
	}

	// create descriptor with dependency
	result, err := desc.CreateFileDescriptor(fd, deps...)
	if err != nil {
		return nil, err
	}

	// add resolved
	resolved[filename] = result
	return result, nil
}

// DescriptorFromFileDescriptors creates a DescriptorSource that is backed by the given
// file descriptors
func DescriptorFromFileDescriptors(files ...*desc.FileDescriptor) (Descriptor, error) {
	fds := map[string]*desc.FileDescriptor{}
	for _, fd := range files {
		if err := addFile(fd, fds); err != nil {
			return nil, err
		}
	}

	return &fileSource{files: fds}, nil
}

func addFile(fd *desc.FileDescriptor, fds map[string]*desc.FileDescriptor) error {
	name := fd.GetName()
	if existing, ok := fds[name]; ok {
		// already added this file
		if existing != fd {
			return fmt.Errorf("given files include multiple copies of %q", name)
		}
		return nil
	}

	fds[name] = fd
	for _, dep := range fd.GetDependencies() {
		if err := addFile(dep, fds); err != nil {
			return err
		}
	}

	return nil
}

// fileSource files of descriptor set
// implement Descriptor interface
// Load data from file descriptors
type fileSource struct {
	files  map[string]*desc.FileDescriptor
	er     *dynamic.ExtensionRegistry
	erInit sync.Once
}

func (fs *fileSource) ListServices() ([]string, error) {
	set := map[string]bool{}
	for _, fd := range fs.files {
		for _, svc := range fd.GetServices() {
			set[svc.GetFullyQualifiedName()] = true
		}
	}
	sl := make([]string, 0, len(set))
	for svc := range set {
		sl = append(sl, svc)
	}

	return sl, nil
}

func (fs *fileSource) ListMethods() (map[string][]string, error) {
	set := map[string][]string{}
	for _, fd := range fs.files {
		for _, svc := range fd.GetServices() {
			m, ok := set[svc.GetFullyQualifiedName()]
			if !ok {
				m = []string{}
			}
			for _, v := range svc.GetMethods() {
				m = append(m, v.GetName())
				// m = append(.GetFullyQualifiedName()], v.GetName())
			}

			set[svc.GetFullyQualifiedName()] = m
		}
	}

	return set, nil
}

// GetAllFiles returns all of the underlying file descriptors. This is
// more thorough and more efficient than the fallback strategy used by
// the GetAllFiles package method, for enumerating all files from a
// descriptor source.
func (fs *fileSource) GetAllFiles() ([]*desc.FileDescriptor, error) {
	files := make([]*desc.FileDescriptor, len(fs.files))
	i := 0
	for _, fd := range fs.files {
		files[i] = fd
		i++
	}

	return files, nil
}

func (fs *fileSource) FindSymbol(fullyQualifiedName string) (desc.Descriptor, error) {
	for _, fd := range fs.files {
		if dsc := fd.FindSymbol(fullyQualifiedName); dsc != nil {
			return dsc, nil
		}
	}

	return nil, notFound("Symbol", fullyQualifiedName)
}

func (fs *fileSource) AllExtensionsForType(typeName string) ([]*desc.FieldDescriptor, error) {
	fs.erInit.Do(func() {
		fs.er = &dynamic.ExtensionRegistry{}
		for _, fd := range fs.files {
			fs.er.AddExtensionsFromFile(fd)
		}
	})
	return fs.er.AllExtensionsForType(typeName), nil
}

// DescriptorFromServer creates a DescriptorSource that uses the given gRPC reflection client
// to interrogate a server for descriptor information. If the server does not support the reflection
// API then the various DescriptorSource methods will return ErrReflectionNotSupported
func DescriptorFromServer(_ context.Context, refClient *grpcreflect.Client) Descriptor {
	return serverSource{client: refClient}
}

type serverSource struct {
	client *grpcreflect.Client
}

func (ss serverSource) ListServices() ([]string, error) {
	svcs, err := ss.client.ListServices()
	return svcs, parseGRPCError(err)
}

func (ss serverSource) FindSymbol(fullyQualifiedName string) (desc.Descriptor, error) {
	file, err := ss.client.FileContainingSymbol(fullyQualifiedName)
	if err != nil {
		return nil, parseGRPCError(err)
	}

	d := file.FindSymbol(fullyQualifiedName)
	if d == nil {
		return nil, notFound("Symbol", fullyQualifiedName)
	}

	return d, nil
}

func (ss serverSource) AllExtensionsForType(typeName string) ([]*desc.FieldDescriptor, error) {
	var exts []*desc.FieldDescriptor
	nums, err := ss.client.AllExtensionNumbersForType(typeName)
	if err != nil {
		return nil, parseGRPCError(err)
	}

	for _, fieldNum := range nums {
		ext, err := ss.client.ResolveExtension(typeName, fieldNum)
		if err != nil {
			return nil, parseGRPCError(err)
		}
		exts = append(exts, ext)
	}

	return exts, nil
}

// ListServices uses the given descriptor source to return a sorted list of fully-qualified
// service names.
func ListServices(source Descriptor) ([]string, error) {
	if source == nil {
		return nil, nil
	}
	svcs, err := source.ListServices()
	if err != nil {
		return nil, err
	}

	sort.Strings(svcs)
	return svcs, nil
}

// GetAllFiles uses the given descriptor source to return a list of file descriptors.
func GetAllFiles(source Descriptor) ([]*desc.FileDescriptor, error) {
	var files []*desc.FileDescriptor
	srcFiles, ok := source.(sourceWithFiles)

	// If an error occurs, we still try to load as many files as we can, so that
	// caller can decide whether to ignore error or not.
	var firstError error
	if ok {
		files, firstError = srcFiles.GetAllFiles()
	} else {
		// Source does not implement GetAllFiles method, so use ListServices
		// and grab files from there.
		svcNames, err := source.ListServices()
		if err != nil {
			firstError = err
		} else {
			allFiles := map[string]*desc.FileDescriptor{}
			for _, name := range svcNames {
				d, err := source.FindSymbol(name)
				if err != nil {
					if firstError == nil {
						firstError = err
					}
				} else {
					addAllFilesToSet(d.GetFile(), allFiles)
				}
			}
			files = make([]*desc.FileDescriptor, len(allFiles))
			i := 0
			for _, fd := range allFiles {
				files[i] = fd
				i++
			}
		}
	}

	return files, firstError
}

func addAllFilesToSet(fd *desc.FileDescriptor, all map[string]*desc.FileDescriptor) {
	if _, ok := all[fd.GetName()]; ok {
		// already added
		return
	}
	all[fd.GetName()] = fd
	for _, dep := range fd.GetDependencies() {
		addAllFilesToSet(dep, all)
	}
}

// ListMethods uses the given descriptor source to return a sorted list of method names
// for the specified fully-qualified service name.
func ListMethods(source Descriptor, serviceName string) ([]string, error) {
	dsc, err := source.FindSymbol(serviceName)
	if err != nil {
		return nil, err
	}

	sd, ok := dsc.(*desc.ServiceDescriptor)
	if !ok {
		return nil, notFound("Service", serviceName)
	}

	methods := make([]string, 0, len(sd.GetMethods()))
	for _, method := range sd.GetMethods() {
		methods = append(methods, method.GetFullyQualifiedName())
	}
	sort.Strings(methods)
	return methods, nil
}

// GetDescriptorText returns a string representation of the given descriptor.
// This returns a snippet of proto source that describes the given element.
func GetDescriptorText(dsc desc.Descriptor, _ Descriptor) (string, error) {
	// Note: DescriptorSource is not used, but remains an argument for backwards
	// compatibility with previous implementation.
	txt, err := printer.PrintProtoToString(dsc)
	if err != nil {
		return "", err
	}
	// callers don't expect trailing newlines
	if txt[len(txt)-1] == '\n' {
		txt = txt[:len(txt)-1]
	}
	return txt, nil
}

// ensureExtensions uses the given descriptor source to download extensions for
// the given message. It returns a copy of the given message, but as a dynamic
// message that knows about all extensions known to the given descriptor source.
// nolint
func ensureExtensions(source Descriptor, msg proto.Message) proto.Message {
	// load any server extensions so we can properly describe custom options
	dsc, err := desc.LoadMessageDescriptorForMessage(msg)
	if err != nil {
		return msg
	}

	var ext dynamic.ExtensionRegistry
	if err = fetchAllExtensions(source, &ext, dsc, map[string]bool{}); err != nil {
		return msg
	}

	// convert message into dynamic message that knows about applicable extensions
	// (that way we can show meaningful info for custom options instead of printing as unknown)
	msgFactory := dynamic.NewMessageFactoryWithExtensionRegistry(&ext)
	dm, err := fullyConvertToDynamic(msgFactory, msg)
	if err != nil {
		return msg
	}
	return dm
}

// fetchAllExtensions recursively fetches from the server extensions for the given message type as well as
// for all message types of nested fields. The extensions are added to the given dynamic registry of extensions
// so that all server-known extensions can be correctly parsed by grpcurl.
func fetchAllExtensions(source Descriptor, ext *dynamic.ExtensionRegistry, md *desc.MessageDescriptor, alreadyFetched map[string]bool) error {
	msgTypeName := md.GetFullyQualifiedName()
	if alreadyFetched[msgTypeName] {
		return nil
	}
	alreadyFetched[msgTypeName] = true
	if len(md.GetExtensionRanges()) > 0 {
		fds, err := source.AllExtensionsForType(msgTypeName)
		if err != nil {
			return fmt.Errorf("failed to query for extensions of type %s: %w", msgTypeName, err)
		}
		for _, fd := range fds {
			if err := ext.AddExtension(fd); err != nil {
				return fmt.Errorf("could not register extension %s of type %s: %w", fd.GetFullyQualifiedName(), msgTypeName, err)
			}
		}
	}
	// recursively fetch extensions for the types of any message fields
	for _, fd := range md.GetFields() {
		if fd.GetMessageType() != nil {
			err := fetchAllExtensions(source, ext, fd.GetMessageType(), alreadyFetched)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// fullConvertToDynamic attempts to convert the given message to a dynamic message as well
// as any nested messages it may contain as field values. If the given message factory has
// extensions registered that were not known when the given message was parsed, this effectively
// allows re-parsing to identify those extensions.
// nolint
func fullyConvertToDynamic(msgFact *dynamic.MessageFactory, msg proto.Message) (proto.Message, error) {
	if _, ok := msg.(*dynamic.Message); ok {
		return msg, nil // already a dynamic message
	}
	md, err := desc.LoadMessageDescriptorForMessage(msg)
	if err != nil {
		return nil, err
	}
	newMsg := msgFact.NewMessage(md)
	dm, ok := newMsg.(*dynamic.Message)
	if !ok {
		// if message factory didn't produce a dynamic message, then we should leave msg as is
		return msg, nil
	}

	if err := dm.ConvertFrom(msg); err != nil {
		return nil, err
	}

	// recursively convert all field values, too
	for _, fd := range md.GetFields() {
		if fd.IsMap() {
			if fd.GetMapValueType().GetMessageType() != nil {
				m := dm.GetField(fd).(map[interface{}]interface{})
				for k, v := range m {
					// keys can't be nested messages; so we only need to recurse through map values, not keys
					newVal, err := fullyConvertToDynamic(msgFact, v.(proto.Message))
					if err != nil {
						return nil, err
					}
					dm.PutMapField(fd, k, newVal)
				}
			}
		} else if fd.IsRepeated() {
			if fd.GetMessageType() != nil {
				s := dm.GetField(fd).([]interface{})
				for i, e := range s {
					newVal, err := fullyConvertToDynamic(msgFact, e.(proto.Message))
					if err != nil {
						return nil, err
					}
					dm.SetRepeatedField(fd, i, newVal)
				}
			}
		} else {
			if fd.GetMessageType() != nil {
				v := dm.GetField(fd)
				newVal, err := fullyConvertToDynamic(msgFact, v.(proto.Message))
				if err != nil {
					return nil, err
				}
				dm.SetField(fd, newVal)
			}
		}
	}
	return dm, nil
}

type multiString []string

func (s *multiString) String() string {
	return strings.Join(*s, ",")
}

func (s *multiString) IsEmpty() bool {
	return len(*s) <= 0
}

func (s *multiString) Set(value string) {
	*s = append(*s, value)
}
